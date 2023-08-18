package main

type CCState int

const (
	CCStateIdle = CCState(iota)
	CCStateWaitRoute
	CCStateARComplete
	CCStateConnected
	CCStateDead
	CCStateDisconnecting
)

func (s CCState) String() string {
	switch s {
	case CCStateIdle:
		return "Idle"
	case CCStateWaitRoute:
		return "WaitRoute"
	case CCStateARComplete:
		return "ARComplete"
	case CCStateConnected:
		return "Connected"
	case CCStateDead:
		return "Dead"
	case CCStateDisconnecting:
		return "Disconnecting"
	}
	return "Unknown"
}
