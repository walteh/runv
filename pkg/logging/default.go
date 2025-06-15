package logging

import (
	"io"
	"os"
	"sync"
)

var logWriter io.Writer
var logWriterMutex sync.Mutex

func GetDefaultLogWriter() io.Writer {
	logWriterMutex.Lock()
	defer logWriterMutex.Unlock()
	if logWriter == nil {
		return os.Stdout
	}
	return logWriter
}

func SetDefaultLogWriter(w io.Writer) {
	logWriterMutex.Lock()
	defer logWriterMutex.Unlock()
	logWriter = w
}
