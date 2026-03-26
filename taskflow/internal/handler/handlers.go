package handler

import (
	"github.com/chaitin/MonkeyCode/taskflow/internal/runner"
	"github.com/chaitin/MonkeyCode/taskflow/internal/store"
)

type Handlers struct {
	Host        *HostHandler
	VM          *VMHandler
	Task        *TaskHandler
	Stats       *StatsHandler
	PortForward *PortForwardHandler
}

func NewHandlers(s *store.RedisStore, m *runner.Manager, sm *runner.StreamManager) *Handlers {
	return &Handlers{
		Host:        NewHostHandler(s),
		VM:          NewVMHandler(s, m, sm),
		Task:        NewTaskHandler(s, sm),
		Stats:       NewStatsHandler(s, m),
		PortForward: NewPortForwardHandler(s, m),
	}
}
