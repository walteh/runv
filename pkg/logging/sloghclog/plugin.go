package sloghclog

import (
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/go-hclog"
)

var _ hclog.Logger = &slogPluginServerClientInterceptor{}

func NewSlogPluginServerClientInterceptor(writer io.Writer) hclog.Logger {
	return &slogPluginServerClientInterceptor{
		writer: writer,
	}
}

type slogPluginServerClientInterceptor struct {
	writer io.Writer
}

// Debug implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) Debug(msg string, args ...interface{}) {
	_, _ = fmt.Fprintln(s.writer, msg)
}

// Error implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) Error(msg string, args ...interface{}) {
	panic("unimplemented")
}

func (s *slogPluginServerClientInterceptor) Trace(msg string, args ...interface{}) {
	panic("unimplemented")
}

// Warn implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) Warn(msg string, args ...interface{}) {
	panic("unimplemented")
}

func (s *slogPluginServerClientInterceptor) Info(msg string, args ...interface{}) {
	panic("unimplemented")
}

// GetLevel implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) GetLevel() hclog.Level {
	panic("unimplemented")
}

// ImpliedArgs implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) ImpliedArgs() []interface{} {
	panic("unimplemented")
}

// Info implements hclog.Logger.

// IsDebug implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) IsDebug() bool {
	panic("unimplemented")
}

// IsError implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) IsError() bool {
	panic("unimplemented")
}

// IsInfo implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) IsInfo() bool {
	panic("unimplemented")
}

// IsTrace implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) IsTrace() bool {
	panic("unimplemented")
}

// IsWarn implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) IsWarn() bool {
	panic("unimplemented")
}

// Log implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) Log(level hclog.Level, msg string, args ...interface{}) {
	panic("unimplemented")
}

// Name implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) Name() string {
	panic("unimplemented")
}

// Named implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) Named(name string) hclog.Logger {
	panic("unimplemented")
}

// ResetNamed implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) ResetNamed(name string) hclog.Logger {
	panic("unimplemented")
}

// SetLevel implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) SetLevel(level hclog.Level) {
	panic("unimplemented")
}

// StandardLogger implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	panic("unimplemented")
}

// StandardWriter implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	panic("unimplemented")
}

// Trace implements hclog.Logger.
// With implements hclog.Logger.
func (s *slogPluginServerClientInterceptor) With(args ...interface{}) hclog.Logger {
	panic("unimplemented")
}
