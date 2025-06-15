package logging

import (
	"log/slog"

	smulti "github.com/samber/slog-multi"
	slcontext "github.com/veqryn/slog-context"
)

// there is a problem with goimports that breaks the options generation if these are imported in the same
// file as the options struct
func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return smulti.Fanout(handlers...)
}

func newContextHandler(handler slog.Handler) slog.Handler {
	return slcontext.NewHandler(handler, &slcontext.HandlerOptions{})
}
