package bootstrap

import (
	"context"
	"errors"
	"testing"
)

type mockComponent struct {
	name      string
	initFn    func(ctx context.Context) error
	startFn   func(ctx context.Context) error
	stopFn    func(ctx context.Context) error
	initialized bool
	started     bool
	stopped     bool
}

func (m *mockComponent) Name() string { return m.name }
func (m *mockComponent) Init(ctx context.Context) error {
	m.initialized = true
	if m.initFn != nil {
		return m.initFn(ctx)
	}
	return nil
}
func (m *mockComponent) Start(ctx context.Context) error {
	if m.startFn != nil {
		err := m.startFn(ctx)
		if err == nil {
			m.started = true
		}
		return err
	}
	m.started = true
	return nil
}
func (m *mockComponent) Stop(ctx context.Context) error {
	m.stopped = true
	if m.stopFn != nil {
		return m.stopFn(ctx)
	}
	return nil
}

func TestLifecycleManager_SuccessSequence(t *testing.T) {
	mgr := NewLifecycleManager()
	ctx := context.Background()

	var sequence []string

	c1 := &mockComponent{
		name: "comp1",
		initFn: func(ctx context.Context) error {
			sequence = append(sequence, "init1")
			return nil
		},
		startFn: func(ctx context.Context) error {
			sequence = append(sequence, "start1")
			return nil
		},
		stopFn: func(ctx context.Context) error {
			sequence = append(sequence, "stop1")
			return nil
		},
	}

	c2 := &mockComponent{
		name: "comp2",
		initFn: func(ctx context.Context) error {
			sequence = append(sequence, "init2")
			return nil
		},
		startFn: func(ctx context.Context) error {
			sequence = append(sequence, "start2")
			return nil
		},
		stopFn: func(ctx context.Context) error {
			sequence = append(sequence, "stop2")
			return nil
		},
	}

	mgr.Register(c1)
	mgr.Register(c2)

	if err := mgr.InitAll(ctx); err != nil {
		t.Fatalf("InitAll failed: %v", err)
	}

	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("StartAll failed: %v", err)
	}

	mgr.StopAll(ctx)

	// Verify sequential orders
	expected := []string{"init1", "init2", "start1", "start2", "stop2", "stop1"}
	if len(sequence) != len(expected) {
		t.Fatalf("expected sequence of length %d, got %d: %v", len(expected), len(sequence), sequence)
	}

	for i, v := range expected {
		if sequence[i] != v {
			t.Errorf("sequence step %d = %q, want %q", i, sequence[i], v)
		}
	}

	if !c1.initialized || !c1.started || !c1.stopped {
		t.Error("comp1 lifecycle incomplete")
	}
	if !c2.initialized || !c2.started || !c2.stopped {
		t.Error("comp2 lifecycle incomplete")
	}
}

func TestLifecycleManager_StartFailureRollback(t *testing.T) {
	mgr := NewLifecycleManager()
	ctx := context.Background()

	var sequence []string

	c1 := &mockComponent{
		name: "comp1",
		startFn: func(ctx context.Context) error {
			sequence = append(sequence, "start1")
			return nil
		},
		stopFn: func(ctx context.Context) error {
			sequence = append(sequence, "stop1")
			return nil
		},
	}

	c2 := &mockComponent{
		name: "comp2",
		startFn: func(ctx context.Context) error {
			return errors.New("failed to start")
		},
		stopFn: func(ctx context.Context) error {
			sequence = append(sequence, "stop2")
			return nil
		},
	}

	mgr.Register(c1)
	mgr.Register(c2)

	_ = mgr.InitAll(ctx)
	err := mgr.StartAll(ctx)

	if err == nil {
		t.Fatal("expected StartAll to return error, got nil")
	}

	// comp1 should have started and then stopped due to comp2 failing to start
	if !c1.started {
		t.Error("comp1 should have started")
	}
	if !c1.stopped {
		t.Error("comp1 should have stopped automatically on startup rollback")
	}
	if c2.started {
		t.Error("comp2 should have returned error on start")
	}
}
