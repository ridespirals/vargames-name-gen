package igdb

import (
	"fmt"
	"io"
	"log"
	"os"
)

// Logger is used for optional progress and debug logging. When nil, no logging is performed.
type Logger interface {
	Logf(format string, args ...interface{})
}

// NoOpLogger discards all log output.
type NoOpLogger struct{}

func (NoOpLogger) Logf(string, ...interface{}) {}

// StdLogger writes to an io.Writer with a prefix (e.g. "igdb ").
type StdLogger struct {
	*log.Logger
}

// NewStdLogger creates a Logger that writes to w with the given prefix.
// If w is nil, os.Stderr is used.
func NewStdLogger(prefix string, w io.Writer) *StdLogger {
	if w == nil {
		w = os.Stderr
	}
	return &StdLogger{Logger: log.New(w, prefix, log.LstdFlags)}
}

// Logf implements Logger.
func (s *StdLogger) Logf(format string, args ...interface{}) {
	s.Output(2, fmt.Sprintf(format, args...))
}

// VerboseLogger only logs when Enabled is true; otherwise it no-ops.
type VerboseLogger struct {
	Logger  Logger
	Enabled bool
}

// Logf implements Logger. No-op when Enabled is false.
func (v *VerboseLogger) Logf(format string, args ...interface{}) {
	if !v.Enabled || v.Logger == nil {
		return
	}
	v.Logger.Logf(format, args...)
}

// LoggerFromVerbose returns a Logger that logs to stderr when verbose is true, otherwise a no-op.
// Use this with config.Verbose or a -verbose flag.
func LoggerFromVerbose(verbose bool) Logger {
	if !verbose {
		return NoOpLogger{}
	}
	return &VerboseLogger{Logger: NewStdLogger("igdb ", os.Stderr), Enabled: true}
}
