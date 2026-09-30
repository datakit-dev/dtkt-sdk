package runtime

import (
	"context"
	"fmt"
	"slices"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/v1beta1/impl"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	Graph struct {
		*shared.Graph[*Node]
		proto *flowv1beta1.Graph
		env   *Env
	}
	GraphNode struct {
		*Env
		ident  string
		proto  *flowv1beta1.Graph_Node
		parsed shared.ParsedNode
	}
)

func NewGraph(proto *flowv1beta1.Graph, opts ...Option) (*Graph, error) {
	graph := &Graph{
		proto: proto,
	}

	graphNodes := proto.GetNodes()
	specNodes := getSpecNodes(proto.GetSpec())
	if len(graphNodes) != len(specNodes) {
		return nil, fmt.Errorf("graphNodes(%d) != specNodes(%d)", len(graphNodes), len(specNodes))
	}

	opts = append([]Option{
		WithEnv(&flowv1beta1.Env{
			Nodes: util.SliceMap(specNodes, func(node shared.SpecNode) *flowv1beta1.Env_Node {
				return &flowv1beta1.Env_Node{
					Ident: shared.GetNodeIdent(node),
				}
			}),
		}),
		WithVars(func(env shared.Env) (cel.Activation, error) {
			return cel.NewActivation(util.SliceToMap(graph.Nodes(), func(_ int, node *Node) (string, any) {
				return node.ident, func() ref.Val {
					vars, err := node.vars()
					if err != nil {
						return types.WrapErr(err)
					}
					return env.Types().NativeToValue(vars)
				}
			}))
		}),
	}, opts...)

	env, err := NewEnv(opts...)
	if err != nil {
		return nil, err
	}

	graph.env = env

	var nodes []*Node
	for _, specNode := range specNodes {
		ident := shared.GetNodeIdent(specNode)
		idx := slices.IndexFunc(graphNodes, func(node *flowv1beta1.Graph_Node) bool {
			return node.Ident == ident
		})
		if idx == -1 {
			return nil, fmt.Errorf("node not found in graph: %q", ident)
		}

		node, err := hydrateNode(env, specNode, graphNodes[idx])
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	edges, err := edgesFromProto(proto.GetEdges())
	if err != nil {
		return nil, err
	}

	graph.Graph, err = shared.NewGraph(
		shared.WithGraphNodes(nodes),
		shared.WithGraphEdges[*Node](edges),
		shared.WithGraphTraits[*Node](shared.DAGTraits()),
	)
	if err != nil {
		return nil, err
	}

	return graph, nil
}

func ParseGraph(spec *flowv1beta1.Flow, opts ...Option) (_ *Graph, err error) {
	graph := &Graph{
		proto: &flowv1beta1.Graph{
			Spec: spec,
		},
	}

	graph.Graph, err = shared.NewGraph(shared.WithGraphTraits[*Node](shared.DAGTraits()))
	if err != nil {
		return nil, err
	}

	specNodes := getSpecNodes(spec)
	opts = append([]Option{
		WithEnv(&flowv1beta1.Env{
			Nodes: util.SliceMap(specNodes, func(node shared.SpecNode) *flowv1beta1.Env_Node {
				return &flowv1beta1.Env_Node{
					Ident:      shared.GetNodeIdent(node),
					ResultCurr: &flowv1beta1.Env_Result{},
				}
			}),
		}),
		WithVars(func(env shared.Env) (cel.Activation, error) {
			return cel.NewActivation(util.SliceToMap(graph.Nodes(), func(_ int, node *Node) (string, any) {
				return node.ident, func() ref.Val {
					vars, err := node.vars()
					if err != nil {
						return types.WrapErr(err)
					}
					return env.Types().NativeToValue(vars)
				}
			}))
		}),
	}, opts...)

	env, err := NewEnv(opts...)
	if err != nil {
		return nil, err
	}

	graph.env = env

	err = graph.Parse(specNodes, func(target shared.SpecNode, visitor shared.NodeVisitor) (*Node, error) {
		return parseNode(graph.env, target, visitor)
	})
	if err != nil {
		return nil, err
	}

	graph.proto.Nodes = nodesToProto(graph.Nodes())
	graph.proto.Edges = edgesToProto(graph.Edges())

	return graph, nil
}

func hydrateNode(env *Env, spec shared.SpecNode, proto *flowv1beta1.Graph_Node) (*Node, error) {
	envNodes := env.proto.GetNodes()
	env, err := env.extend(specNodeEnvOpts(spec)...)
	if err != nil {
		return nil, err
	}

	node := &GraphNode{
		Env:   env,
		ident: proto.Ident,
		proto: proto,
	}

	parsed, err := impl.NewNode(node, spec, func(target string, ast *cel.Ast) {})
	if err != nil {
		return nil, err
	}

	node.parsed = parsed

	idx := slices.IndexFunc(envNodes, func(e *flowv1beta1.Env_Node) bool {
		return e.GetIdent() == node.ident
	})
	if idx == -1 {
		return nil, fmt.Errorf("node not found in env: %q", node.ident)
	}

	return newNode(node, envNodes[idx])
}

func parseNode(env *Env, spec shared.SpecNode, visitor shared.NodeVisitor) (*Node, error) {
	envNodes := env.proto.GetNodes()
	env, err := env.extend(specNodeEnvOpts(spec)...)
	if err != nil {
		return nil, err
	}

	ident := shared.GetNodeIdent(spec)
	node := &GraphNode{
		Env:   env,
		ident: ident,
		proto: &flowv1beta1.Graph_Node{
			Ident: ident,
		},
	}

	parsed, err := impl.NewNode(node, spec, visitor)
	if err != nil {
		return nil, err
	}

	node.parsed = parsed

	idx := slices.IndexFunc(envNodes, func(e *flowv1beta1.Env_Node) bool {
		return e.GetIdent() == node.ident
	})
	if idx == -1 {
		return nil, fmt.Errorf("node not found in env: %q", node.ident)
	}

	return newNode(node, envNodes[idx])
}

func (n *GraphNode) ParseExpr(id string, expr string, visitor shared.ExprVisitor) (*cel.Ast, error) {
	if n.proto.Exprs != nil && n.proto.Exprs[id] != nil && n.proto.Exprs[id].Parsed != nil {
		return cel.ParsedExprToAst(n.proto.Exprs[id].Parsed), nil
	}

	expr, valid := shared.IsValidExpr(expr)
	if !valid {
		return nil, shared.NewExprError("%s.%s: %s: %s", n.ident, id, shared.InvalidExprErrPrefix, expr)
	}

	ast, iss := n.Env.Parse(expr)
	if iss.Err() != nil {
		return nil, shared.NewExprError("%s.%s: %v", n.ident, id, iss.Err())
	}

	if visitor != nil {
		visitor(ast)
	}

	parsed, err := cel.AstToParsedExpr(ast)
	if err != nil {
		return nil, err
	}

	if n.proto.Exprs == nil {
		n.proto.Exprs = map[string]*flowv1beta1.Graph_Expr{}
	}

	n.proto.Exprs[id] = &flowv1beta1.Graph_Expr{
		Parsed: parsed,
	}

	return ast, nil
}

func (n *GraphNode) CompileExpr(id string, ast *cel.Ast) (shared.Program, error) {
	if n.proto.Exprs == nil || n.proto.Exprs[id] == nil {
		return nil, fmt.Errorf("%s: compile expression: parsed expression not found", id)
	} else if n.proto.Exprs[id].Checked != nil {
		ast = cel.CheckedExprToAst(n.proto.Exprs[id].Checked)
	} else if !ast.IsChecked() {
		var iss *cel.Issues
		ast, iss = n.Env.Check(ast)
		if iss.Err() != nil {
			return nil, iss.Err()
		}

		checked, err := cel.AstToCheckedExpr(ast)
		if err != nil {
			return nil, err
		}

		n.proto.Exprs[id].Checked = checked
	}

	prog, err := n.Env.Program(ast, cel.InterruptCheckFrequency(1))
	if err != nil {
		return nil, err
	}

	return shared.ProgramFunc(func(ctx context.Context, env shared.Env) ref.Val {
		vars, err := env.Vars()
		if err != nil {
			return types.WrapErr(err)
		}

		val, _, err := prog.ContextEval(ctx, vars)
		if err != nil {
			return types.WrapErr(err)
		}

		return val
	}), nil
}

func (n *GraphNode) Compile(ctx context.Context) error {
	return n.parsed.Compile(ctx, n)
}

func (g *Graph) Compile(ctx context.Context) (err error) {
	for _, n := range g.Nodes() {
		err = n.Compile(ctx)
		if err != nil {
			return
		}
	}
	return
}

func (g *Graph) Env() *Env {
	return g.env
}

func (g *Graph) Proto() *flowv1beta1.Graph {
	return proto.CloneOf(g.proto)
}

func (n *GraphNode) Proto() *flowv1beta1.Graph_Node {
	return proto.CloneOf(n.proto)
}

func (g *Graph) Parsed() bool {
	nodes := g.Nodes()
	return len(util.SliceReduce(nodes, func(n *Node) (*Node, bool) {
		return n, n.Parsed()
	})) == len(nodes)
}

func (g *Graph) Compiled() bool {
	nodes := g.Nodes()
	return len(util.SliceReduce(nodes, func(n *Node) (*Node, bool) {
		return n, n.Compiled()
	})) == len(nodes)
}

func (n *GraphNode) Parsed() bool {
	return len(util.ReduceMap(n.proto.GetExprs(), func(_ string, expr *flowv1beta1.Graph_Expr) bool {
		return expr.GetParsed() != nil
	})) == len(n.proto.GetExprs())
}

func (n *GraphNode) Compiled() bool {
	return len(util.ReduceMap(n.proto.GetExprs(), func(_ string, expr *flowv1beta1.Graph_Expr) bool {
		return expr.GetParsed() != nil && expr.GetChecked() != nil
	})) == len(n.proto.GetExprs())
}

func nodesToProto(nodes []*Node) []*flowv1beta1.Graph_Node {
	return util.SliceMap(nodes, func(node *Node) *flowv1beta1.Graph_Node {
		return node.GraphNode.proto
	})
}

func edgesToProto(edges []*shared.Edge) []*flowv1beta1.Graph_Edge {
	return util.SliceMap(edges, func(edge *shared.Edge) *flowv1beta1.Graph_Edge {
		return &flowv1beta1.Graph_Edge{
			Source: edge.Source(),
			Target: edge.Target(),
		}
	})
}

func edgesFromProto(edges []*flowv1beta1.Graph_Edge) ([]*shared.Edge, error) {
	return util.SliceReduceError(edges, func(proto *flowv1beta1.Graph_Edge) (*shared.Edge, bool, error) {
		edge, err := shared.NewEdge(proto.GetSource(), proto.GetTarget())
		if err != nil {
			return nil, false, err
		}
		return edge, true, nil
	})
}

func specNodeEnvOpts(node shared.SpecNode) (opts []cel.EnvOption) {
	opts = append(opts, cel.Variable("self", varsCELType))
	switch node := node.(type) {
	case *flowv1beta1.Action:
		if name, ok := callNodeContainer(node); ok {
			opts = append(opts, cel.Container(string(name)))
		}
	case *flowv1beta1.Stream:
		if name, ok := callNodeContainer(node); ok {
			opts = append(opts, cel.Container(string(name)))
		}
	}
	return
}

func callNodeContainer[T impl.Call](node impl.CallNode[T]) (protoreflect.FullName, bool) {
	if node.GetCall().GetMethod() == "" {
		return "", false
	}

	name := protoreflect.FullName(node.GetCall().GetMethod()).
		Parent(). // Parent is service
		Parent()  // Parent is package

	return name, name.IsValid()
}

func getSpecNode[T shared.SpecNode](node T) shared.SpecNode {
	return node
}

func getSpecNodes(flow *flowv1beta1.Flow) []shared.SpecNode {
	return util.JoinSlices(
		util.SliceMap(flow.Actions, getSpecNode),
		util.SliceMap(flow.Connections, getSpecNode),
		util.SliceMap(flow.Inputs, getSpecNode),
		util.SliceMap(flow.Outputs, getSpecNode),
		util.SliceMap(flow.Streams, getSpecNode),
		util.SliceMap(flow.Vars, getSpecNode),
	)
}
