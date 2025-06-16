package runtime

import (
	"path/filepath"
	"runtime"

	"gitlab.com/tozd/go/errors"
)

func ReflectNotImplementedError() error {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return errors.Errorf("not implemented: failed to get caller")
	}
	funcName := runtime.FuncForPC(pc).Name()
	return errors.Errorf("not implemented: %s", filepath.Base(funcName))
}
