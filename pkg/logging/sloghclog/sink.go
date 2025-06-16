package sloghclog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
)

type slogSink struct {
	logger *slog.Logger
}

var _ hclog.SinkAdapter = &slogSink{}

func InterceptHclog(slog *slog.Logger) {
	intercept := hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Level:  hclog.NoLevel,
		Output: io.Discard,
		Color:  hclog.ColorOff,
	})

	intercept.RegisterSink(&slogSink{logger: slog})

	hclog.SetDefault(intercept)
}

func (s *slogSink) Accept(name string, hlevel hclog.Level, msg string, args ...interface{}) {
	now := time.Now()
	// Map logrus levels to slog levels
	var level slog.Level
	switch hlevel {
	case hclog.Error:
		level = slog.LevelError
	case hclog.Warn:
		level = slog.LevelWarn
	case hclog.Info:
		level = slog.LevelInfo
	case hclog.Debug:
		level = slog.LevelDebug
	case hclog.Trace:
		level = slog.LevelDebug - 4
	default:
		level = slog.LevelInfo
	}

	// // Prepare slog attributes
	// attrs := make([]slog.Attr, 0, len(entry.Data))
	// for k, v := range entry.Data {
	// 	attrs = append(attrs, slog.Any(k, v))
	// }

	// slices.SortFunc(attrs, func(a, b slog.Attr) int {
	// 	return strings.Compare(a.Key, b.Key)
	// })

	attrs := make([]slog.Attr, 0, len(args))
	for i := 0; i < len(args); i += 2 {
		var s string
		if sd, ok := args[i].(string); ok {
			s = sd
		} else {
			s = fmt.Sprintf("%v", args[i])
		}
		switch v := args[i+1].(type) {
		case string:
			attrs = append(attrs, slog.String(s, v))
		case int:
			attrs = append(attrs, slog.Int(s, v))
		case bool:
			attrs = append(attrs, slog.Bool(s, v))
		case float64:
			attrs = append(attrs, slog.Float64(s, v))
		case time.Time:
			attrs = append(attrs, slog.Time(s, v))
		case error:
			attrs = append(attrs, slog.Any(s, v))
		default:
			attrs = append(attrs, slog.Any(s, v))
		}
	}

	slices.SortFunc(attrs, func(a, b slog.Attr) int {
		return strings.Compare(b.Key, b.Key)
	})

	caller, _, _, _ := runtime.Caller(3)
	record := slog.NewRecord(now, level, msg, caller)
	record.AddAttrs(attrs...)

	ctx := context.Background()

	if s.logger == nil {
		_ = slog.Default().Handler().Handle(ctx, record)
	} else {
		_ = s.logger.Handler().Handle(ctx, record)
	}

}
