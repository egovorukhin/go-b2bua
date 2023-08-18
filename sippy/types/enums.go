package sippy_types

type UaStateID int

const (
	UA_STATE_NONE = UaStateID(iota)

	UAS_STATE_IDLE
	UAS_STATE_TRYING
	UAS_STATE_RINGING
	UAS_STATE_UPDATING
	UAS_STATE_PRE_CONNECT

	UAC_STATE_IDLE
	UAC_STATE_TRYING
	UAC_STATE_RINGING
	UAC_STATE_UPDATING
	UAC_STATE_CANCELLING

	UA_STATE_CONNECTED
	UA_STATE_DISCONNECTED
	UA_STATE_FAILED
	UA_STATE_DEAD
)
