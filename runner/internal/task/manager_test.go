package task

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.tasks == nil {
		t.Error("tasks map is nil")
	}

	if mgr.Count() != 0 {
		t.Errorf("expected count 0, got %d", mgr.Count())
	}
}

func TestCreate(t *testing.T) {
	mgr := NewManager()

	tests := []struct {
		name    string
		opts    CreateOptions
		wantErr bool
	}{
		{
			name: "valid task",
			opts: CreateOptions{
				VMID:   "vm-123",
				UserID: "user-1",
				Text:   "帮我写一个 Hello World",
				Model:  "gpt-4",
				Agent:  "opencode",
			},
			wantErr: false,
		},
		{
			name: "missing vm_id",
			opts: CreateOptions{
				VMID:   "",
				UserID: "user-1",
				Text:   "帮我写一个 Hello World",
			},
			wantErr: true,
		},
		{
			name: "missing text",
			opts: CreateOptions{
				VMID:   "vm-123",
				UserID: "user-1",
				Text:   "",
			},
			wantErr: true,
		},
		{
			name: "default agent",
			opts: CreateOptions{
				VMID:   "vm-123",
				UserID: "user-1",
				Text:   "帮我写一个 Hello World",
				Agent:  "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := mgr.Create(tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if task.ID == "" {
				t.Error("task.ID is empty")
			}

			if task.VMID != tt.opts.VMID {
				t.Errorf("VMID = %s; want %s", task.VMID, tt.opts.VMID)
			}

			if task.UserID != tt.opts.UserID {
				t.Errorf("UserID = %s; want %s", task.UserID, tt.opts.UserID)
			}

			if task.Status != StatusPending {
				t.Errorf("Status = %s; want %s", task.Status, StatusPending)
			}

			expectedAgent := tt.opts.Agent
			if expectedAgent == "" {
				expectedAgent = "opencode"
			}
			if task.Agent != expectedAgent {
				t.Errorf("Agent = %s; want %s", task.Agent, expectedAgent)
			}

			if task.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})
	}
}

func TestGet(t *testing.T) {
	mgr := NewManager()

	task, err := mgr.Create(CreateOptions{
		VMID:   "vm-123",
		UserID: "user-1",
		Text:   "test task",
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	got, err := mgr.Get(task.ID)
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}

	if got.ID != task.ID {
		t.Errorf("ID = %s; want %s", got.ID, task.ID)
	}

	_, err = mgr.Get("non-existent-id")
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}

func TestUpdateStatus(t *testing.T) {
	mgr := NewManager()

	task, err := mgr.Create(CreateOptions{
		VMID:   "vm-123",
		UserID: "user-1",
		Text:   "test task",
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	tests := []struct {
		name       string
		status     Status
		wantErr    bool
		checkTimes bool
	}{
		{
			name:    "update to running",
			status:  StatusRunning,
			wantErr: false,
		},
		{
			name:    "update to completed",
			status:  StatusCompleted,
			wantErr: false,
		},
		{
			name:    "update to failed",
			status:  StatusFailed,
			wantErr: false,
		},
		{
			name:    "non-existent task",
			status:  StatusRunning,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "non-existent task" {
				err := mgr.UpdateStatus("non-existent", tt.status)
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			err := mgr.UpdateStatus(task.ID, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

			got, _ := mgr.Get(task.ID)
			if got.Status != tt.status {
				t.Errorf("Status = %s; want %s", got.Status, tt.status)
			}
		})
	}
}

func TestSetError(t *testing.T) {
	mgr := NewManager()

	task, _ := mgr.Create(CreateOptions{
		VMID:   "vm-123",
		UserID: "user-1",
		Text:   "test task",
	})

	errMsg := "task execution failed"
	err := mgr.SetError(task.ID, errMsg)
	if err != nil {
		t.Errorf("SetError() error = %v", err)
	}

	got, _ := mgr.Get(task.ID)
	if got.Status != StatusFailed {
		t.Errorf("Status = %s; want %s", got.Status, StatusFailed)
	}

	if got.Error != errMsg {
		t.Errorf("Error = %s; want %s", got.Error, errMsg)
	}

	if got.EndedAt == nil {
		t.Error("EndedAt should be set")
	}
}

func TestSetResult(t *testing.T) {
	mgr := NewManager()

	task, _ := mgr.Create(CreateOptions{
		VMID:   "vm-123",
		UserID: "user-1",
		Text:   "test task",
	})

	result := "task completed successfully"
	err := mgr.SetResult(task.ID, result)
	if err != nil {
		t.Errorf("SetResult() error = %v", err)
	}

	got, _ := mgr.Get(task.ID)
	if got.Status != StatusCompleted {
		t.Errorf("Status = %s; want %s", got.Status, StatusCompleted)
	}

	if got.Result != result {
		t.Errorf("Result = %s; want %s", got.Result, result)
	}

	if got.EndedAt == nil {
		t.Error("EndedAt should be set")
	}
}

func TestCancel(t *testing.T) {
	mgr := NewManager()

	tests := []struct {
		name        string
		initialStat Status
		wantCancel  bool
	}{
		{
			name:        "cancel pending task",
			initialStat: StatusPending,
			wantCancel:  true,
		},
		{
			name:        "cancel running task",
			initialStat: StatusRunning,
			wantCancel:  true,
		},
		{
			name:        "cancel completed task",
			initialStat: StatusCompleted,
			wantCancel:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, _ := mgr.Create(CreateOptions{
				VMID:   "vm-123",
				UserID: "user-1",
				Text:   "test task",
			})

			mgr.UpdateStatus(task.ID, tt.initialStat)
			mgr.Cancel(task.ID)

			got, _ := mgr.Get(task.ID)
			if tt.wantCancel && got.Status != StatusCancelled {
				t.Errorf("Status = %s; want %s", got.Status, StatusCancelled)
			}
			if !tt.wantCancel && got.Status != tt.initialStat {
				t.Errorf("Status changed from %s", tt.initialStat)
			}
		})
	}
}

func TestList(t *testing.T) {
	mgr := NewManager()

	mgr.Create(CreateOptions{
		VMID:   "vm-1",
		UserID: "user-1",
		Text:   "task 1",
	})
	mgr.Create(CreateOptions{
		VMID:   "vm-2",
		UserID: "user-1",
		Text:   "task 2",
	})
	mgr.Create(CreateOptions{
		VMID:   "vm-3",
		UserID: "user-2",
		Text:   "task 3",
	})

	tests := []struct {
		name     string
		userID   string
		expected int
	}{
		{
			name:     "list all tasks",
			userID:   "",
			expected: 3,
		},
		{
			name:     "list user-1 tasks",
			userID:   "user-1",
			expected: 2,
		},
		{
			name:     "list user-2 tasks",
			userID:   "user-2",
			expected: 1,
		},
		{
			name:     "list non-existent user tasks",
			userID:   "user-99",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := mgr.List(tt.userID)
			if len(tasks) != tt.expected {
				t.Errorf("List(%s) returned %d tasks; want %d", tt.userID, len(tasks), tt.expected)
			}
		})
	}
}

func TestListByVM(t *testing.T) {
	mgr := NewManager()

	mgr.Create(CreateOptions{
		VMID:   "vm-1",
		UserID: "user-1",
		Text:   "task 1",
	})
	mgr.Create(CreateOptions{
		VMID:   "vm-1",
		UserID: "user-1",
		Text:   "task 2",
	})
	mgr.Create(CreateOptions{
		VMID:   "vm-2",
		UserID: "user-1",
		Text:   "task 3",
	})

	tests := []struct {
		name     string
		vmID     string
		expected int
	}{
		{
			name:     "list vm-1 tasks",
			vmID:     "vm-1",
			expected: 2,
		},
		{
			name:     "list vm-2 tasks",
			vmID:     "vm-2",
			expected: 1,
		},
		{
			name:     "list non-existent vm tasks",
			vmID:     "vm-99",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := mgr.ListByVM(tt.vmID)
			if len(tasks) != tt.expected {
				t.Errorf("ListByVM(%s) returned %d tasks; want %d", tt.vmID, len(tasks), tt.expected)
			}
		})
	}
}

func TestCount(t *testing.T) {
	mgr := NewManager()

	if mgr.Count() != 0 {
		t.Errorf("expected count 0, got %d", mgr.Count())
	}

	mgr.Create(CreateOptions{
		VMID:   "vm-1",
		UserID: "user-1",
		Text:   "task 1",
	})

	if mgr.Count() != 1 {
		t.Errorf("expected count 1, got %d", mgr.Count())
	}

	mgr.Create(CreateOptions{
		VMID:   "vm-2",
		UserID: "user-1",
		Text:   "task 2",
	})

	if mgr.Count() != 2 {
		t.Errorf("expected count 2, got %d", mgr.Count())
	}
}

func TestRunningCount(t *testing.T) {
	mgr := NewManager()

	task1, _ := mgr.Create(CreateOptions{
		VMID:   "vm-1",
		UserID: "user-1",
		Text:   "task 1",
	})
	mgr.Create(CreateOptions{
		VMID:   "vm-2",
		UserID: "user-1",
		Text:   "task 2",
	})

	if mgr.RunningCount() != 0 {
		t.Errorf("expected running count 0, got %d", mgr.RunningCount())
	}

	mgr.UpdateStatus(task1.ID, StatusRunning)

	if mgr.RunningCount() != 1 {
		t.Errorf("expected running count 1, got %d", mgr.RunningCount())
	}
}

func TestDelete(t *testing.T) {
	mgr := NewManager()

	task, _ := mgr.Create(CreateOptions{
		VMID:   "vm-1",
		UserID: "user-1",
		Text:   "task to delete",
	})

	if mgr.Count() != 1 {
		t.Errorf("expected count 1, got %d", mgr.Count())
	}

	err := mgr.Delete(task.ID)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	if mgr.Count() != 0 {
		t.Errorf("expected count 0, got %d", mgr.Count())
	}

	_, err = mgr.Get(task.ID)
	if err == nil {
		t.Error("expected error after delete")
	}

	err = mgr.Delete("non-existent")
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr := NewManager()
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				task, _ := mgr.Create(CreateOptions{
					VMID:   "vm-1",
					UserID: "user-1",
					Text:   "concurrent task",
				})
				mgr.Get(task.ID)
				mgr.UpdateStatus(task.ID, StatusRunning)
				mgr.SetResult(task.ID, "done")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if mgr.Count() != 1000 {
		t.Errorf("expected count 1000 after concurrent access, got %d", mgr.Count())
	}
}

func TestTaskLifecycle(t *testing.T) {
	mgr := NewManager()

	task, err := mgr.Create(CreateOptions{
		VMID:   "vm-123",
		UserID: "user-1",
		Text:   "test lifecycle",
		Model:  "gpt-4",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if task.Status != StatusPending {
		t.Errorf("initial status = %s; want %s", task.Status, StatusPending)
	}
	if task.StartedAt != nil {
		t.Error("StartedAt should be nil initially")
	}
	if task.EndedAt != nil {
		t.Error("EndedAt should be nil initially")
	}

	mgr.UpdateStatus(task.ID, StatusRunning)
	task, _ = mgr.Get(task.ID)
	if task.Status != StatusRunning {
		t.Errorf("status = %s; want %s", task.Status, StatusRunning)
	}
	if task.StartedAt == nil {
		t.Error("StartedAt should be set after running")
	}

	mgr.SetResult(task.ID, "success")
	task, _ = mgr.Get(task.ID)
	if task.Status != StatusCompleted {
		t.Errorf("status = %s; want %s", task.Status, StatusCompleted)
	}
	if task.Result != "success" {
		t.Errorf("result = %s; want %s", task.Result, "success")
	}
	if task.EndedAt == nil {
		t.Error("EndedAt should be set after completion")
	}

	elapsed := task.EndedAt.Sub(*task.StartedAt)
	if elapsed < 0 {
		t.Error("EndedAt should be after StartedAt")
	}
}
