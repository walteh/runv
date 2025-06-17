package gvnet

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"

	"github.com/containers/gvisor-tap-vsock/pkg/services/forwarder"
	"github.com/containers/gvisor-tap-vsock/pkg/tcpproxy"
	"github.com/containers/gvisor-tap-vsock/pkg/transport"
	"github.com/soheilhy/cmux"
	"github.com/walteh/run"
	"gitlab.com/tozd/go/errors"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type MagicHostPort struct {
	mux      cmux.CMux
	addr     net.Addr
	grp      *run.Group
	toClose  []io.Closer
	alive    bool
	listener net.Listener
}

func NewMagicHostPortStream(ctx context.Context, magicHostPort string) (*MagicHostPort, error) {
	l, err := transport.Listen(magicHostPort)
	if err != nil {
		return nil, errors.Errorf("listen: %w", err)
	}

	grp := run.New(run.WithLogger(slog.Default()))

	cmux := cmux.New(l)

	cmux.HandleError(func(err error) bool {
		slog.ErrorContext(ctx, "cmux error", "error", err)
		return true
	})

	grp.Always(NewCmuxServer(fmt.Sprintf("globalhostport_cmux(%s)", magicHostPort), cmux))

	return &MagicHostPort{mux: cmux, addr: l.Addr(), grp: grp, toClose: []io.Closer{l}, listener: l}, nil
}

func (g *MagicHostPort) ApplyRestMux(name string, mux http.Handler) {
	g.grp.Always(NewHTTPServer("globalhostport_"+name, mux, g.mux.Match(cmux.Any())))
}

func (g *MagicHostPort) Run(ctx context.Context) error {
	g.alive = true
	defer func() {
		g.alive = false
	}()

	err := g.grp.RunContext(ctx)
	if err != nil {
		return errors.Errorf("run: %w", err)
	}
	return nil
}

func (g *MagicHostPort) Alive() bool {
	return g.alive
}

func (g *MagicHostPort) Close(ctx context.Context) error {
	for _, c := range g.toClose {
		go c.Close()
	}
	return nil
}

func (g *MagicHostPort) Fields() []slog.Attr {
	return []slog.Attr{
		slog.Group("globalhostport",
			slog.String("addr", g.addr.String()),
			slog.Bool("alive", g.alive),
		),
	}
}

func (g *MagicHostPort) Name() string {
	return fmt.Sprintf("globalhostport(%s)", g.addr.String())
}

func (g *MagicHostPort) ForwardCMUXMatchToGuestPort(ctx context.Context, switc *stack.Stack, guestPortTarget uint16, matcher cmux.Matcher) error {
	listener := g.mux.Match(matcher)

	hostAddress := fmt.Sprintf("cmux_match:%d", guestPortTarget)

	guestPortTargetStr := fmt.Sprintf("%s:%d", VIRTUAL_GUEST_IP, guestPortTarget)

	guestAddress, err := forwarder.TCPIPAddress(1, guestPortTargetStr)
	if err != nil {
		listener.Close()
		return errors.Errorf("failed to get tcpip address: %w", err)
	}

	// honestly not sure if this is needed, just interested in playing more with switch
	nic := switc.CheckLocalAddress(1, ipv4.ProtocolNumber, guestAddress.Addr)
	if nic != 1 {
		guestAddress.NIC = nic
	}

	var proxy tcpproxy.Proxy
	proxy.ListenFunc = func(network, laddr string) (net.Listener, error) {
		// slog.InfoContext(ctx, "listening", slog.Group("ignored",
		// 	slog.String("network", network),
		// 	slog.String("address", laddr),
		// ), "hostAddress", hostAddress)
		return listener, nil
	}

	proxy.AddRoute(hostAddress, &tcpproxy.DialProxy{
		Addr: guestPortTargetStr,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			// slog.InfoContext(ctx, "dialing", slog.Group("ignored",
			// 	slog.String("network", network),
			// 	slog.String("address", address),
			// ), "guestAddress", guestAddress.Addr.String(), "hostAddress", hostAddress)
			return gonet.DialContextTCP(ctx, switc, guestAddress, ipv4.ProtocolNumber)
		},
		OnDialError: func(src net.Conn, dstDialErr error) {
			slog.ErrorContext(ctx, "failed to dial", "error", dstDialErr)
			src.Close()
		},
	})

	tcpproxyRunner, err := NewTCPProxyRunner(hostAddress, guestPortTargetStr, &proxy)
	if err != nil {
		return errors.Errorf("failed to create tcpproxy runner: %w", err)
	}

	g.grp.Always(tcpproxyRunner)
	g.toClose = append(g.toClose, listener)

	return nil
}
