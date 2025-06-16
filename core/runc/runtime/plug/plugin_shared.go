package plug

import (
	"context"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/walteh/runv/core/runc/runtime"
	grpcruntime "github.com/walteh/runv/core/runc/runtime/grpc"
	"github.com/walteh/runv/core/runc/server"
	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/grpc"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = plugin.PluginSet{
	"runc": &RuntimePlugin{},
}

const (
	PluginName = "runc"
)

func NewRuntimePluginSet(impl runtime.Runtime, runtimeExtras runtime.RuntimeExtras, socketAllocator runtime.SocketAllocator) plugin.PluginSet {
	return plugin.PluginSet{
		PluginName: &RuntimePlugin{
			GuestRuntime:         impl,
			GuestRuntimeExtras:   runtimeExtras,
			GuestSocketAllocator: socketAllocator,
		},
	}
}

var _ plugin.Plugin = (*RuntimePlugin)(nil)

var _ plugin.GRPCPlugin = (*RuntimePlugin)(nil)

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type RuntimePlugin struct {

	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	GuestRuntime         runtime.Runtime
	GuestRuntimeExtras   runtime.RuntimeExtras
	GuestSocketAllocator runtime.SocketAllocator
}

// GRPCPlugin must still implement the Plugin interface
func (p *RuntimePlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	panic(runtime.ReflectNotImplementedError())
}

func (p *RuntimePlugin) Server(broker *plugin.MuxBroker) (interface{}, error) {
	panic(runtime.ReflectNotImplementedError())
}

func (p *RuntimePlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	runcServer := server.NewServer(p.GuestRuntime, p.GuestRuntimeExtras, p.GuestSocketAllocator)
	runvv1.RegisterRuncServiceServer(s, runcServer)
	return nil
}

func (p *RuntimePlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return grpcruntime.NewGRPCClientRuntimeFromConn(c)
}
