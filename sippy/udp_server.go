package sippy

// #include <sys/socket.h>
//
// #ifdef SO_REUSEPORT
// #define SO_REUSEPORT_EXISTS 1
// #else
// #define SO_REUSEPORT_EXISTS 0
// #define SO_REUSEPORT 0 /* just a placeholder to keep the go code compilable */
// #endif
//
import "C"
import (
	"net"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/log"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/utils"
)

type write_req struct {
	address     net.Addr
	data        []byte
	on_complete func()
}

type resolv_req struct {
	hostPort *sippy_net.HostPort
	data     []byte
}

type asyncResolver struct {
	sem    chan int
	logger sippy_log.ErrorLogger
}

func NewAsyncResolver(userv *UdpServer, logger sippy_log.ErrorLogger) *asyncResolver {
	s := &asyncResolver{
		sem:    make(chan int, 2),
		logger: logger,
	}
	go s.run(userv)
	return s
}

func (s *asyncResolver) run(userv *UdpServer) {
	var wi *resolv_req
LOOP:
	for {
		wi = <-userv.wi_resolv
		if wi == nil {
			// Shutdown request, relay it further
			userv.wi_resolv <- nil
			break LOOP
		}
		start, _ := sippy_time.NewMonoTime()
		addr, err := net.ResolveUDPAddr("udp", wi.hostPort.String())
		delay, _ := start.OffsetFromNow()
		if err != nil {
			s.logger.Errorf("Udp_server: Cannot resolve '%s', dropping outgoing SIP message. Delay %s", wi.hostPort, delay.String())
			continue
		}
		if delay > time.Duration(.5*float64(time.Second)) {
			s.logger.Error("Udp_server: DNS resolve time for '%s' is too big: %s", wi.hostPort, delay.String())
		}
		userv._send_to(wi.data, addr, nil)
	}
	s.sem <- 1
}

type asyncSender struct {
	sem chan int
}

func NewAsyncSender(userv *UdpServer, n int) *asyncSender {
	s := &asyncSender{
		sem: make(chan int, 2),
	}
	go s.run(userv)
	return s
}

func (s *asyncSender) run(userv *UdpServer) {
	var wi *write_req
LOOP:
	for {
		wi = <-userv.wi
		if wi == nil { // shutdown req
			userv.wi <- nil
			break LOOP
		}
	SEND_LOOP:
		for i := 0; i < 20; i++ {
			if _, err := userv.skt.WriteTo(wi.data, wi.address); err == nil {
				if wi.on_complete != nil {
					wi.on_complete()
				}
				break SEND_LOOP
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	s.sem <- 1
}

type asyncReceiver struct {
	sem    chan int
	logger sippy_log.ErrorLogger
}

func NewAsyncReciever(userv *UdpServer, logger sippy_log.ErrorLogger) *asyncReceiver {
	s := &asyncReceiver{
		sem:    make(chan int, 2),
		logger: logger,
	}
	go s.run(userv)
	return s
}

func (s *asyncReceiver) run(userv *UdpServer) {
	buf := make([]byte, 8192)
	for {
		n, address, err := userv.skt.ReadFrom(buf)
		if err != nil {
			break
		}
		rtime, err := sippy_time.NewMonoTime()
		if err != nil {
			s.logger.Error("Cannot create MonoTime object")
			continue
		}
		msg := make([]byte, 0, n)
		msg = append(msg, buf[:n]...)
		sippy_utils.SafeCall(func() { userv.handle_read(msg, address, rtime) }, nil, s.logger)
	}
	s.sem <- 1
}

type udpServerOpts struct {
	laddress      *sippy_net.HostPort
	data_callback sippy_net.DataPacketReceiver
	nworkers      int
}

func NewUdpServerOpts(laddress *sippy_net.HostPort, data_callback sippy_net.DataPacketReceiver) *udpServerOpts {
	s := &udpServerOpts{
		laddress:      laddress,
		data_callback: data_callback,
		nworkers:      runtime.NumCPU() * 2,
	}
	return s
}

type UdpServer struct {
	uopts          udpServerOpts
	skt            net.PacketConn
	wi             chan *write_req
	wi_resolv      chan *resolv_req
	asenders       []*asyncSender
	areceivers     []*asyncReceiver
	aresolvers     []*asyncResolver
	packets_recvd  int
	packets_sent   int
	packets_queued int
}

func zoneToUint32(zone string) uint32 {
	if zone == "" {
		return 0
	}
	if ifi, err := net.InterfaceByName(zone); err == nil {
		return uint32(ifi.Index)
	}
	n, err := strconv.Atoi(zone)
	if err != nil {
		return 0
	}
	return uint32(n)
}

func NewUdpServer(config sippy_conf.Config, uopts *udpServerOpts) (*UdpServer, error) {
	var laddress *net.UDPAddr
	var err error
	var ip4 net.IP

	proto := syscall.AF_INET
	if uopts.laddress != nil {
		laddress, err = net.ResolveUDPAddr("udp", uopts.laddress.String())
		if err != nil {
			return nil, err
		}
		if sippy_net.IsIP4(laddress.IP) {
			ip4 = laddress.IP.To4()
		} else {
			proto = syscall.AF_INET6
		}
	}
	s, err := syscall.Socket(proto, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}
	if laddress != nil {
		if err = syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
			syscall.Close(s)
			return nil, err
		}
		if C.SO_REUSEPORT_EXISTS == 1 {
			if err = syscall.SetsockoptInt(s, syscall.SOL_SOCKET, C.SO_REUSEPORT, 1); err != nil {
				syscall.Close(s)
				return nil, err
			}
		}
		var sockaddr syscall.Sockaddr
		if ip4 != nil {
			sockaddr = &syscall.SockaddrInet4{
				Port: laddress.Port,
				Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
			}
		} else {
			sa6 := &syscall.SockaddrInet6{
				Port:   laddress.Port,
				ZoneId: zoneToUint32(laddress.Zone),
			}
			for i := 0; i < 16; i++ {
				sa6.Addr[i] = laddress.IP[i]
			}
			sockaddr = sa6
		}
		if err := syscall.Bind(s, sockaddr); err != nil {
			syscall.Close(s)
			return nil, err
		}
	}
	f := os.NewFile(uintptr(s), "")
	skt, err := net.FilePacketConn(f)
	f.Close()
	if err != nil {
		return nil, err
	}
	s := &UdpServer{
		uopts:      *uopts,
		skt:        skt,
		wi:         make(chan *write_req, 1000),
		wi_resolv:  make(chan *resolv_req, 1000),
		asenders:   make([]*asyncSender, 0, uopts.nworkers),
		areceivers: make([]*asyncReceiver, 0, uopts.nworkers),
		aresolvers: make([]*asyncResolver, 0, uopts.nworkers),
	}
	for n := 0; n < uopts.nworkers; n++ {
		s.asenders = append(s.asenders, NewAsyncSender(s, n))
		s.areceivers = append(s.areceivers, NewAsyncReciever(s, config.ErrorLogger()))
	}
	for n := 0; n < uopts.nworkers; n++ {
		s.aresolvers = append(s.aresolvers, NewAsyncResolver(s, config.ErrorLogger()))
	}
	return s, nil
}

func (s *UdpServer) SendTo(data []byte, hostPort *sippy_net.HostPort) {
	s.SendToWithCb(data, hostPort, nil)
}

func (s *UdpServer) SendToWithCb(data []byte, hostPort *sippy_net.HostPort, on_complete func()) {
	ip := hostPort.ParseIP()
	if ip == nil {
		s.wi_resolv <- &resolv_req{data: data, hostPort: hostPort}
		return
	}
	address, err := net.ResolveUDPAddr("udp", hostPort.String()) // in fact no resolving is done here
	if err != nil {
		return // not reached
	}
	s._send_to(data, address, on_complete)
}

func (s *UdpServer) _send_to(data []byte, address net.Addr, on_complete func()) {
	s.wi <- &write_req{
		data:        data,
		address:     address,
		on_complete: on_complete,
	}
}

func (s *UdpServer) handle_read(data []byte, address net.Addr, rtime *sippy_time.MonoTime) {
	if len(data) > 0 {
		s.packets_recvd++
		host, port, _ := net.SplitHostPort(address.String())
		s.uopts.data_callback(data, sippy_net.NewHostPort(host, port), s, rtime)
	}
}

func (s *UdpServer) Shutdown() {
	// shutdown the senders and resolvers first
	s.wi <- nil
	s.wi_resolv <- nil
	for _, worker := range s.asenders {
		<-worker.sem
	}
	for _, worker := range s.aresolvers {
		<-worker.sem
	}
	s.skt.Close()

	for _, worker := range s.areceivers {
		<-worker.sem
	}
	s.asenders = make([]*asyncSender, 0)
	s.areceivers = make([]*asyncReceiver, 0)
	s.aresolvers = make([]*asyncResolver, 0)
}

func (s *UdpServer) GetLAddress() *sippy_net.HostPort {
	return s.uopts.laddress
}
