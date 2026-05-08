package runtime

import (
	"fmt"
	"os"
	"regexp"
	"sync"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// validatePublishedEvent runs a focused regex check on the per-node id
// field of every FlowEvent emitted by publishNode/publishStateEvent/
// publishFlowState when the DTKT_FLOW_VALIDATE_PUBLISH environment
// variable is non-empty. It catches the entire class of "wrong id format
// written into a proto field whose validator pattern is bare-only"
// regressions at the publish site, before the bug reaches the wire
// boundary where the only recourse is a server-side rejection.
//
// Pattern (Format A) on per-node id fields:
//
//	^[a-zA-Z][a-zA-Z0-9_]*$
//
// This matches every per-node {Input,Var,Action,Output,Stream,
// Generator,Interaction}Node.id field's buf-validate pattern. Category
// is implicit in the FlowEvent.data oneof case, so the bare id alone
// is the contract.
//
// Why a regex check instead of full protovalidate.Validate():
//
//   - Full protovalidate runs reflection over every field and is too
//     heavy for in-process spam paths (e.g. operator-API stress tests
//     that fire SuspendNode in tight loops).
//   - The systemic id-format bug class is what this hook defends
//     against; a single regex on the id field covers it.
//   - Production runs (DTKT_FLOW_VALIDATE_PUBLISH unset) skip the
//     check entirely.
//
// Tests should opt in via t.Setenv("DTKT_FLOW_VALIDATE_PUBLISH", "1")
// in TestMain or via build configuration; setting it for the whole
// package catches every emit across every test.
var (
	publishValidateOnce    sync.Once
	publishValidateEnabled bool
	bareIDFormat           = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
)

func validatePublishedEvent(evt *flowv1beta2.RunSnapshot_FlowEvent) error {
	publishValidateOnce.Do(func() {
		publishValidateEnabled = os.Getenv("DTKT_FLOW_VALIDATE_PUBLISH") != ""
	})
	if !publishValidateEnabled {
		return nil
	}

	id := publishedNodeID(evt)
	if id == "" {
		return nil // FLOW_UPDATE has no node id; nothing to check.
	}
	if !bareIDFormat.MatchString(id) {
		return fmt.Errorf(
			"publish validation failed (DTKT_FLOW_VALIDATE_PUBLISH=1): per-node id %q does not match bare-id pattern %s -- handler likely emitted a fully-qualified node id where the protobuf validator requires the bare spec id",
			id, bareIDFormat.String(),
		)
	}
	return nil
}

// publishedNodeID returns the id field on whichever per-node variant
// the FlowEvent carries. Returns "" for FLOW_UPDATE events (no node id).
func publishedNodeID(evt *flowv1beta2.RunSnapshot_FlowEvent) string {
	switch evt.WhichData() {
	case flowv1beta2.RunSnapshot_FlowEvent_Input_case:
		return evt.GetInput().GetId()
	case flowv1beta2.RunSnapshot_FlowEvent_Generator_case:
		return evt.GetGenerator().GetId()
	case flowv1beta2.RunSnapshot_FlowEvent_Var_case:
		return evt.GetVar().GetId()
	case flowv1beta2.RunSnapshot_FlowEvent_Action_case:
		return evt.GetAction().GetId()
	case flowv1beta2.RunSnapshot_FlowEvent_Stream_case:
		return evt.GetStream().GetId()
	case flowv1beta2.RunSnapshot_FlowEvent_Output_case:
		return evt.GetOutput().GetId()
	case flowv1beta2.RunSnapshot_FlowEvent_Interaction_case:
		return evt.GetInteraction().GetId()
	}
	return ""
}
