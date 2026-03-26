package runner

import (
	"sync"

	pb "github.com/chaitin/MonkeyCode/taskflow/pkg/proto"
)

type StreamManager struct {
	mu      sync.RWMutex
	streams map[string]*RunnerStream
}

type RunnerStream struct {
	RunnerID string
	Stream   pb.RunnerService_CommandStreamServer
	SendChan chan *pb.TaskflowCommand
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(map[string]*RunnerStream),
	}
}

func (m *StreamManager) Register(runnerID string, stream pb.RunnerService_CommandStreamServer) *RunnerStream {
	m.mu.Lock()
	defer m.mu.Unlock()

	rs := &RunnerStream{
		RunnerID: runnerID,
		Stream:   stream,
		SendChan: make(chan *pb.TaskflowCommand, 100),
	}
	m.streams[runnerID] = rs
	return rs
}

func (m *StreamManager) Unregister(runnerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rs, ok := m.streams[runnerID]; ok {
		close(rs.SendChan)
		delete(m.streams, runnerID)
	}
}

func (m *StreamManager) SendCommand(runnerID string, cmd *pb.TaskflowCommand) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rs, ok := m.streams[runnerID]
	if !ok {
		return ErrRunnerNotFound
	}

	select {
	case rs.SendChan <- cmd:
		return nil
	default:
		return ErrStreamFull
	}
}

func (m *StreamManager) IsOnline(runnerID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.streams[runnerID]
	return ok
}

func (m *StreamManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.streams))
	for id := range m.streams {
		ids = append(ids, id)
	}
	return ids
}

func (m *StreamManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.streams)
}
