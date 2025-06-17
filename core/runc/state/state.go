package state

import (
	"github.com/walteh/runm/core/runc/runtime"
	"github.com/walteh/runm/pkg/syncmap"
)

type State struct {
	openIOs      *syncmap.Map[string, runtime.IO]
	openSockets  *syncmap.Map[string, runtime.AllocatedSocket]
	openConsoles *syncmap.Map[string, runtime.ConsoleSocket]
}

func NewState() *State {
	return &State{
		openIOs:      syncmap.NewMap[string, runtime.IO](),
		openSockets:  syncmap.NewMap[string, runtime.AllocatedSocket](),
		openConsoles: syncmap.NewMap[string, runtime.ConsoleSocket](),
	}
}

func (s *State) GetOpenIO(referenceId string) (runtime.IO, bool) {
	return s.openIOs.Load(referenceId)
}

func (s *State) GetOpenSocket(referenceId string) (runtime.AllocatedSocket, bool) {
	return s.openSockets.Load(referenceId)
}

func (s *State) GetOpenConsole(referenceId string) (runtime.ConsoleSocket, bool) {
	return s.openConsoles.Load(referenceId)
}

func (s *State) StoreOpenIO(referenceId string, io runtime.IO) {
	s.openIOs.Store(referenceId, io)
}

func (s *State) StoreOpenSocket(referenceId string, socket runtime.AllocatedSocket) {
	s.openSockets.Store(referenceId, socket)
}

func (s *State) StoreOpenConsole(referenceId string, console runtime.ConsoleSocket) {
	s.openConsoles.Store(referenceId, console)
}

func (s *State) DeleteOpenIO(referenceId string) {
	s.openIOs.Delete(referenceId)
}

func (s *State) DeleteOpenSocket(referenceId string) {
	s.openSockets.Delete(referenceId)
}

func (s *State) DeleteOpenConsole(referenceId string) {
	s.openConsoles.Delete(referenceId)
}
