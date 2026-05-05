package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TestSpec_FieldCoverage walks the descriptor for each top-level Flow node
// type and asserts every field is referenced by at least one testdata yaml.
// The intent is catching "added a new spec field, forgot to add coverage" --
// not validating semantics. The match is purely textual: we look for
// `<proto_field_name>:` somewhere in any testdata file. Fields that intentionally
// have no testdata footprint (purely internal proto plumbing, descriptor
// metadata) belong in the exemptions map below with a one-line reason.
func TestSpec_FieldCoverage(t *testing.T) {
	yamlCorpus := loadTestdataCorpus(t)

	// Top-level node messages whose fields drive runtime behavior. Each
	// must be covered by at least one yaml.
	roots := []protoreflect.MessageDescriptor{
		(&flowv1beta2.Flow{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.Connection{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.Input{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.Var{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.Action{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.Stream{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.Output{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.Interaction{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.Generator{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.FlowControl{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.NodeControl{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.RetryStrategy{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.Switch{}).ProtoReflect().Descriptor(),
		(&flowv1beta2.MethodCall{}).ProtoReflect().Descriptor(),
	}

	// Fields that are intentionally not yaml-visible. Keys are
	// `<MessageName>.<field_name>`; values are the reason. Add an entry only
	// when there is a principled reason -- "TODO" is not one.
	exemptions := map[string]string{
		// Flow.error_strategy is an enum exercised through Graph proto in tests
		// rather than yaml fixtures (loaded via flow control test scaffolding).
		"Flow.error_strategy": "exercised via Graph proto, not yaml fixtures",
	}

	var missing []string
	for _, md := range roots {
		fields := md.Fields()
		for i := 0; i < fields.Len(); i++ {
			fd := fields.Get(i)
			name := string(fd.Name())
			key := string(md.Name()) + "." + name
			if _, ok := exemptions[key]; ok {
				continue
			}
			needle := name + ":"
			if !strings.Contains(yamlCorpus, needle) {
				missing = append(missing, key)
			}
		}
	}

	if len(missing) > 0 {
		t.Errorf("spec fields with no testdata coverage (%d):\n  - %s\n\nAdd a testdata yaml that uses the field, or document the exemption in coverage_test.go.",
			len(missing), strings.Join(missing, "\n  - "))
	}
}

func loadTestdataCorpus(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join("testdata", e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String()
}
