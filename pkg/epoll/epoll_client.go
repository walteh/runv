package epoll

import "github.com/containerd/console"

type Epoller struct {
}

func NewEpoller() (*Epoller, error) {
	return &Epoller{}, nil
}

func (e *Epoller) Add(console console.Console) (*EpollConsole, error) {
	panic("unimplemented")
}

func (e *Epoller) CloseConsole(fd int) error {
	panic("unimplemented")
}

func (e *Epoller) Wait() error {
	panic("unimplemented")
}

func (e *Epoller) Close() error {
	panic("unimplemented")
}
