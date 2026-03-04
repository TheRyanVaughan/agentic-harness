package harness

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ryanvaughan/agentic-harness/internal/backend"
	"github.com/ryanvaughan/agentic-harness/internal/config"
)

// mockBackend implements backend.Backend for testing.
type mockBackend struct {
	name      string
	runFunc   func(ctx context.Context, opts backend.RunOpts) (*backend.RunResult, error)
	reviewFn  func(ctx context.Context, opts backend.ReviewOpts) (*backend.RunResult, error)
}

func (m *mockBackend) Name() string { return m.name }

func (m *mockBackend) Run(ctx context.Context, opts backend.RunOpts) (*backend.RunResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, opts)
	}
	return &backend.RunResult{
		ExitCode: 0,
		CostUSD:  0.50,
		Duration: time.Second,
	}, nil
}

func (m *mockBackend) ReviewRun(ctx context.Context, opts backend.ReviewOpts) (*backend.RunResult, error) {
	if m.reviewFn != nil {
		return m.reviewFn(ctx, opts)
	}
	return &backend.RunResult{
		ExitCode: 0,
		CostUSD:  0.25,
		Duration: time.Second,
	}, nil
}

func TestStateTransitions(t *testing.T) {
	// Test that each state has valid transitions
	for state, transitions := range Transitions {
		for _, tr := range transitions {
			if _, ok := StateInfos[tr.To]; !ok {
				t.Errorf("state %s transitions to unknown state %s", state, tr.To)
			}
		}
	}
}

func TestStateNeedsHuman(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StateSpecReady, true},
		{StateExecuting, false},
		{StateVerifying, false},
		{StateRetrying, false},
		{StateReviewing, false},
		{StateFixing, false},
		{StateHumanReview, true},
		{StateComplete, false},
		{StateStuck, true},
	}

	for _, tt := range tests {
		if got := tt.state.NeedsHuman(); got != tt.want {
			t.Errorf("State(%s).NeedsHuman() = %v, want %v", tt.state, got, tt.want)
		}
	}
}

func TestStateStringRoundtrip(t *testing.T) {
	for state, str := range stateStrings {
		parsed, err := ParseState(str)
		if err != nil {
			t.Errorf("ParseState(%q) error: %v", str, err)
		}
		if parsed != state {
			t.Errorf("ParseState(%q) = %v, want %v", str, parsed, state)
		}
	}
}

func TestStateJSONRoundtrip(t *testing.T) {
	for state := range stateStrings {
		data, err := json.Marshal(state)
		if err != nil {
			t.Errorf("Marshal(%s) error: %v", state, err)
		}
		var parsed State
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Errorf("Unmarshal(%s) error: %v", string(data), err)
		}
		if parsed != state {
			t.Errorf("JSON roundtrip: got %v, want %v", parsed, state)
		}
	}
}

func TestStateFilePersistence(t *testing.T) {
	dir := t.TempDir()
	sf := NewStateFile(dir)
	sf.Task = "test-task"
	sf.Spec = "spec.md"
	sf.State = StateExecuting
	sf.NeedsHuman = false
	sf.Attempt = 2
	sf.CostUSD = 1.23
	sf.Backend = "claude"
	sf.Model = "opus"
	sf.History = []HistoryEntry{
		{State: StateSpecReady, At: time.Now()},
		{State: StateExecuting, At: time.Now()},
	}

	if err := sf.Persist(); err != nil {
		t.Fatalf("Persist() error: %v", err)
	}

	loaded, err := LoadStateFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("LoadStateFile() error: %v", err)
	}

	if loaded.Task != sf.Task {
		t.Errorf("Task = %q, want %q", loaded.Task, sf.Task)
	}
	if loaded.State != sf.State {
		t.Errorf("State = %v, want %v", loaded.State, sf.State)
	}
	if loaded.Attempt != sf.Attempt {
		t.Errorf("Attempt = %d, want %d", loaded.Attempt, sf.Attempt)
	}
	if loaded.CostUSD != sf.CostUSD {
		t.Errorf("CostUSD = %f, want %f", loaded.CostUSD, sf.CostUSD)
	}
	if len(loaded.History) != len(sf.History) {
		t.Errorf("History length = %d, want %d", len(loaded.History), len(sf.History))
	}
}

func TestPromptAssembly(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".agent")

	// Create spec file
	specFile := filepath.Join(dir, "spec.md")
	os.WriteFile(specFile, []byte("# Test Spec\n\nDo the thing."), 0o644)

	// Create AGENTS.md
	agentsFile := filepath.Join(dir, "AGENTS.md")
	os.WriteFile(agentsFile, []byte("# Agent Instructions\n\nFollow the spec."), 0o644)

	prompt, err := NewPrompt(agentDir, specFile, agentsFile, "")
	if err != nil {
		t.Fatalf("NewPrompt() error: %v", err)
	}

	data, err := os.ReadFile(prompt.Path())
	if err != nil {
		t.Fatalf("reading prompt: %v", err)
	}

	content := string(data)
	if !contains(content, "Test Spec") {
		t.Error("prompt missing spec content")
	}
	if !contains(content, "Agent Instructions") {
		t.Error("prompt missing agents content")
	}
}

func TestPromptRetryContext(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".agent")

	specFile := filepath.Join(dir, "spec.md")
	os.WriteFile(specFile, []byte("# Spec"), 0o644)

	prompt, err := NewPrompt(agentDir, specFile, "", "")
	if err != nil {
		t.Fatalf("NewPrompt() error: %v", err)
	}

	err = prompt.AppendRetryContext(
		os.ErrNotExist,
		"FAIL: TestFoo - expected 42, got 0",
	)
	if err != nil {
		t.Fatalf("AppendRetryContext() error: %v", err)
	}

	data, err := os.ReadFile(prompt.Path())
	if err != nil {
		t.Fatalf("reading prompt: %v", err)
	}

	content := string(data)
	if !contains(content, "Retry Context") {
		t.Error("prompt missing retry context")
	}
	if !contains(content, "TestFoo") {
		t.Error("prompt missing error output")
	}
}

func TestExitCodes(t *testing.T) {
	tests := []struct {
		state State
		code  int
	}{
		{StateComplete, 0},
		{StateHumanReview, 2},
		{StateStuck, 3},
		{StateExecuting, 1},
	}

	for _, tt := range tests {
		r := &Result{State: tt.state}
		if got := r.ExitCode(); got != tt.code {
			t.Errorf("ExitCode() for %s = %d, want %d", tt.state, got, tt.code)
		}
	}
}

func TestNewRunner(t *testing.T) {
	dir := t.TempDir()

	specFile := filepath.Join(dir, "spec.md")
	os.WriteFile(specFile, []byte("# Test Spec"), 0o644)

	cfg := config.DefaultConfig()
	mb := &mockBackend{name: "mock"}

	runner, err := NewRunner(RunnerOpts{
		Backend:  mb,
		Config:   cfg,
		WorkDir:  dir,
		SpecFile: specFile,
		TaskName: "test",
	})
	if err != nil {
		t.Fatalf("NewRunner() error: %v", err)
	}
	if runner == nil {
		t.Fatal("NewRunner() returned nil")
	}

	// Check .agent dir was created
	agentDir := filepath.Join(dir, ".agent")
	if _, err := os.Stat(agentDir); err != nil {
		t.Errorf(".agent dir not created: %v", err)
	}

	// Check prompt was assembled
	promptFile := filepath.Join(agentDir, "prompt.md")
	if _, err := os.Stat(promptFile); err != nil {
		t.Errorf("prompt.md not created: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
