package runtime

import (
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/uuid"
)

type PendingUserAction struct {
	uid        uuid.UUID
	nodeID     string
	userAction *flowv1beta1.UserAction
	values     map[string]any
}

func (u *PendingUserAction) NodeID() string {
	return u.nodeID
}

func (u *PendingUserAction) UserAction() *flowv1beta1.UserAction {
	return u.userAction
}

func (u *PendingUserAction) SetValues(values map[string]any) {
	u.values = values
}
