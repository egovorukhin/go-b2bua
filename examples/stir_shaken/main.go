package main

import (
	crand "crypto/rand"
	"flag"
	mrand "math/rand"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy"
	"github.com/egovorukhin/go-b2bua/sippy/log"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

func init() {
	Next_cc_id = make(chan int64)
	go func() {
		var id int64 = 1
		for {
			Next_cc_id <- id
			id++
		}
	}()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	buf := make([]byte, 8)
	crand.Read(buf)
	var salt int64
	for _, c := range buf {
		salt = (salt << 8) | int64(c)
	}
	mrand.Seed(salt)

	var laddr, nh_addr, logfile string
	var crt_roots_file string
	var lport int
	var attest, origid, x5u, crt_file, pkey_file string
	var verify bool

	flag.StringVar(&laddr, "l", "", "Local addr")
	flag.IntVar(&lport, "p", 5060, "Local port")
	flag.StringVar(&nh_addr, "n", "192.168.0.102:5060", "Next hop address")
	flag.StringVar(&logfile, "L", "/var/log/sip.log", "Log file")
	flag.StringVar(&crt_roots_file, "r", "", "Root certificate chain file")
	flag.StringVar(&attest, "a", "", "STIR/SHAKEN attest")
	flag.StringVar(&origid, "o", "", "STIR/SHAKEN origid")
	flag.StringVar(&crt_file, "c", "", "Certificate file")
	flag.StringVar(&pkey_file, "k", "", "Private key file")
	flag.StringVar(&x5u, "x", "", "STIR/SHAKEN x5u")
	flag.BoolVar(&verify, "vs", false, "Do verification")
	flag.Parse()

	error_logger := sippy_log.NewErrorLogger()
	sip_logger, err := sippy_log.NewSipLogger("b2bua", logfile)
	if err != nil {
		error_logger.Error(err)
		return
	}
	if crt_roots_file == "" || origid == "" || x5u == "" || crt_file == "" || pkey_file == "" {
		flag.Usage()
		return
	}
	config := NewMyConfig(error_logger, sip_logger)
	//config.SetIPV6Enabled(false)
	config.Attest = attest
	config.Origid = origid
	config.X5u = x5u
	config.Crt_file = crt_file
	config.Pkey_file = pkey_file
	config.Crt_roots_file = crt_roots_file
	config.Verify = verify

	if nh_addr != "" {
		var parts []string
		var addr string

		if strings.HasPrefix(nh_addr, "[") {
			parts = strings.SplitN(nh_addr, "]", 2)
			addr = parts[0] + "]"
			if len(parts) == 2 {
				parts = strings.SplitN(parts[1], ":", 2)
			}
		} else {
			parts = strings.SplitN(nh_addr, ":", 2)
			addr = parts[0]
		}
		port := "5060"
		if len(parts) == 2 {
			port = parts[1]
		}
		config.Nh_addr = sippy_net.NewHostPort(addr, port)
	}
	config.SetMyUAName("Sippy B2BUA (Simple)")
	config.SetAllowFormats([]int{0, 8, 18, 100, 101})
	if laddr != "" {
		config.SetMyAddress(sippy_net.NewMyAddress(laddr))
	}
	config.SetSipAddress(config.GetMyAddress())
	if lport > 0 {
		config.SetMyPort(sippy_net.NewMyPort(strconv.Itoa(lport)))
	}
	config.SetSipPort(config.GetMyPort())
	cmap, err := NewCallMap(config, error_logger)
	if err != nil {
		error_logger.Error(err)
		return
	}
	sip_tm, err := sippy.NewSipTransactionManager(config, cmap)
	if err != nil {
		error_logger.Error(err)
		return
	}
	cmap.Sip_tm = sip_tm
	cmap.Proxy = sippy.NewStatefulProxy(sip_tm, config.Nh_addr, config)
	go sip_tm.Run()

	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan, syscall.SIGTERM, syscall.SIGINT)
	signal.Ignore(syscall.SIGHUP, syscall.SIGPIPE)
	select {
	case <-signal_chan:
		cmap.Shutdown()
		sip_tm.Shutdown()
		time.Sleep(time.Second)
		break
	}
}
