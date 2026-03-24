package vm

import (
	"fmt"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager(nil)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.vms == nil {
		t.Error("vms map is nil")
	}

	if mgr.Count() != 0 {
		t.Errorf("expected count 0, got %d", mgr.Count())
	}

	if mgr.RunningCount() != 0 {
		t.Errorf("expected running count 0, got %d", mgr.RunningCount())
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPending, "pending"},
		{StatusCreating, "creating"},
		{StatusRunning, "running"},
		{StatusStopped, "stopped"},
		{StatusError, "error"},
		{StatusDeleted, "deleted"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Status %v = %s; want %s", tt.status, string(tt.status), tt.expected)
		}
	}
}

func TestVMFields(t *testing.T) {
	now := time.Now()
	vm := &VM{
		ID:          "vm-123",
		ContainerID: "container-456",
		UserID:      "user-1",
		Status:      StatusRunning,
		ImageURL:    "ubuntu:22.04",
		GitURL:      "https://github.com/example/repo",
		CreatedAt:   now,
		Error:       "",
	}

	if vm.ID != "vm-123" {
		t.Errorf("ID = %s; want %s", vm.ID, "vm-123")
	}

	if vm.ContainerID != "container-456" {
		t.Errorf("ContainerID = %s; want %s", vm.ContainerID, "container-456")
	}

	if vm.Status != StatusRunning {
		t.Errorf("Status = %s; want %s", vm.Status, StatusRunning)
	}

	if vm.CreatedAt != now {
		t.Errorf("CreatedAt mismatch")
	}
}

func TestCreateOptions(t *testing.T) {
	opts := CreateOptions{
		UserID:   "user-1",
		ImageURL: "ubuntu:22.04",
		GitURL:   "https://github.com/example/repo",
		GitToken: "ghp_token123",
		Cores:    4,
		Memory:   8192,
		EnvVars: map[string]string{
			"ENV1": "value1",
			"ENV2": "value2",
		},
		TTLSeconds: 3600,
	}

	if opts.UserID != "user-1" {
		t.Errorf("UserID = %s; want %s", opts.UserID, "user-1")
	}

	if opts.Cores != 4 {
		t.Errorf("Cores = %d; want %d", opts.Cores, 4)
	}

	if opts.Memory != 8192 {
		t.Errorf("Memory = %d; want %d", opts.Memory, 8192)
	}

	if len(opts.EnvVars) != 2 {
		t.Errorf("EnvVars length = %d; want %d", len(opts.EnvVars), 2)
	}

	if opts.TTLSeconds != 3600 {
		t.Errorf("TTLSeconds = %d; want %d", opts.TTLSeconds, 3600)
	}
}

func TestVMLifecycle(t *testing.T) {
	mgr := NewManager(nil)

	vm := &VM{
		ID:        "test-vm-1",
		UserID:    "user-1",
		Status:    StatusPending,
		ImageURL:  "ubuntu:22.04",
		CreatedAt: time.Now(),
	}

	mgr.vms[vm.ID] = vm

	if mgr.Count() != 1 {
		t.Errorf("Count() = %d; want 1", mgr.Count())
	}

	got, err := mgr.Get(vm.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.ID != vm.ID {
		t.Errorf("Get() returned wrong VM")
	}

	vms := mgr.List("user-1")
	if len(vms) != 1 {
		t.Errorf("List() returned %d VMs; want 1", len(vms))
	}

	count := mgr.Count()
	if count != 1 {
		t.Errorf("Count() = %d; want 1", count)
	}

	delete(mgr.vms, vm.ID)

	_, err = mgr.Get(vm.ID)
	if err == nil {
		t.Error("VM should not exist after delete")
	}
}

func TestVMList(t *testing.T) {
	mgr := NewManager(nil)

	vm1 := &VM{ID: "vm-1", UserID: "user-1", Status: StatusRunning}
	vm2 := &VM{ID: "vm-2", UserID: "user-1", Status: StatusPending}
	vm3 := &VM{ID: "vm-3", UserID: "user-2", Status: StatusRunning}

	mgr.vms[vm1.ID] = vm1
	mgr.vms[vm2.ID] = vm2
	mgr.vms[vm3.ID] = vm3

	tests := []struct {
		name     string
		userID   string
		expected int
	}{
		{
			name:     "list all",
			userID:   "",
			expected: 3,
		},
		{
			name:     "list user-1",
			userID:   "user-1",
			expected: 2,
		},
		{
			name:     "list user-2",
			userID:   "user-2",
			expected: 1,
		},
		{
			name:     "list non-existent user",
			userID:   "user-99",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vms := mgr.List(tt.userID)
			if len(vms) != tt.expected {
				t.Errorf("List(%s) returned %d VMs; want %d", tt.userID, len(vms), tt.expected)
			}
		})
	}
}

func TestVMRunningCount(t *testing.T) {
	mgr := NewManager(nil)

	vm1 := &VM{ID: "vm-1", Status: StatusRunning}
	vm2 := &VM{ID: "vm-2", Status: StatusPending}
	vm3 := &VM{ID: "vm-3", Status: StatusRunning}

	mgr.vms[vm1.ID] = vm1
	mgr.vms[vm2.ID] = vm2
	mgr.vms[vm3.ID] = vm3

	if mgr.RunningCount() != 2 {
		t.Errorf("RunningCount() = %d; want 2", mgr.RunningCount())
	}

	vm4 := &VM{ID: "vm-4", Status: StatusRunning}
	mgr.vms[vm4.ID] = vm4

	if mgr.RunningCount() != 3 {
		t.Errorf("RunningCount() = %d; want 3", mgr.RunningCount())
	}
}

func TestVMStatusUpdate(t *testing.T) {
	mgr := NewManager(nil)

	vm := &VM{ID: "vm-1", Status: StatusPending}
	mgr.vms[vm.ID] = vm

	mgr.updateStatus("vm-1", StatusCreating)

	if vm.Status != StatusCreating {
		t.Errorf("Status = %s; want %s", vm.Status, StatusCreating)
	}

	mgr.updateStatus("vm-1", StatusRunning)

	if vm.Status != StatusRunning {
		t.Errorf("Status = %s; want %s", vm.Status, StatusRunning)
	}
}

func TestVMError(t *testing.T) {
	mgr := NewManager(nil)

	vm := &VM{ID: "vm-1", Status: StatusCreating}
	mgr.vms[vm.ID] = vm

	mgr.setError("vm-1", "failed to pull image")

	if vm.Status != StatusError {
		t.Errorf("Status = %s; want %s", vm.Status, StatusError)
	}

	if vm.Error != "failed to pull image" {
		t.Errorf("Error = %s; want %s", vm.Error, "failed to pull image")
	}
}

func TestConcurrentCount(t *testing.T) {
	for iteration := 0; iteration < 3; iteration++ {
		mgr := NewManager(nil)
		
		for i := 0; i < 10; i++ {
			vm := &VM{
				ID:        fmt.Sprintf("vm-%d-%d", iteration, i),
				UserID:    "user-1",
				Status:    StatusPending,
				CreatedAt: time.Now(),
			}
			mgr.vms[vm.ID] = vm
		}

		if mgr.Count() != 10 {
			t.Errorf("Count() = %d; want 10", mgr.Count())
		}
	}
}
