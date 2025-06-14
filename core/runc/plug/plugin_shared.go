package plug

import (
	"context"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/walteh/runv/core/runc/client"
	"github.com/walteh/runv/core/runc/runtime"
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
var PluginMap = map[string]plugin.GRPCPlugin{
	"runc": &RuntimePlugin{},
}

var _ plugin.Plugin = (*RuntimePlugin)(nil)

var _ plugin.GRPCPlugin = (*RuntimePlugin)(nil)

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type RuntimePlugin struct {

	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl runtime.Runtime
}

// GRPCPlugin must still implement the Plugin interface
func (p *RuntimePlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	panic("unimplemented")
}

func (p *RuntimePlugin) Server(broker *plugin.MuxBroker) (interface{}, error) {
	panic("unimplemented")
}

func (p *RuntimePlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	runcServer := server.NewServer(p.Impl, nil)
	runvv1.RegisterRuncServiceServer(s, runcServer)
	return nil
}

func (p *RuntimePlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return client.NewClientFromConn(c)
}
