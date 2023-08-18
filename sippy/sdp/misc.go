package sippy_sdp

type SdpHeader interface {
	String() string
	LocalStr(hostPort *sippy_net.HostPort) string
}

type Sdp_header_and_name struct {
	Name   string
	Header SdpHeader
}
