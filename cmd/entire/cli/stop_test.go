package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/paths"
	"github.com/entireio/cli/cmd/entire/cli/session"
	"github.com/entireio/cli/cmd/entire/cli/strategy"
	"github.com/entireio/cli/cmd/entire/cli/testutil"
)

// setupStopTestRepo initializes a temporary git repo, changes to it, and clears
// path/session caches. Must NOT be used with t.Parallel() because it calls t.Chdir.
func setupStopTestRepo(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	testutil.InitRepo(t, tmpDir)
	testutil.WriteFile(t, tmpDir, "f.txt", "init")
	testutil.GitAdd(t, tmpDir, "f.txt")
	testutil.GitCommit(t, tmpDir, "init")
	t.Chdir(tmpDir)
	paths.ClearWorktreeRootCache()
	session.ClearGitCommonDirCache()
}

// makeSessionState returns a minimal SessionState suitable for test setup.
func makeSessionState(id string, phase session.Phase) *strategy.SessionState {
	return &strategy.SessionState{
		SessionID:  id,
		BaseCommit: "abc123",
		StartedAt:  time.Now(),
		Phase:      phase,
	}
}

func TestStopCmd_NoActiveSessions(t *testing.T) {
	setupStopTestRepo(t)

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(stdout.String(), "No active sessions.") {
		t.Errorf("expected 'No active sessions.' in output, got: %q", stdout.String())
	}
}

func TestStopCmd_SingleSession_Force(t *testing.T) {
	setupStopTestRepo(t)

	state := makeSessionState("test-stop-single-1", session.PhaseIdle)
	state.StepCount = 0
	if err := strategy.SaveSessionState(context.Background(), state); err != nil {
		t.Fatalf("SaveSessionState() error = %v", err)
	}

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--force"})

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "No work recorded.") {
		t.Errorf("expected 'No work recorded.' in output, got: %q", out)
	}

	loaded, err := strategy.LoadSessionState(context.Background(), "test-stop-single-1")
	if err != nil {
		t.Fatalf("LoadSessionState() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("expected session state to still exist after stop")
	}
	if loaded.Phase != session.PhaseEnded {
		t.Errorf("expected Phase=PhaseEnded, got: %v", loaded.Phase)
	}
}

func TestStopCmd_SingleSession_WithCheckpoint(t *testing.T) {
	setupStopTestRepo(t)

	state := makeSessionState("test-stop-checkpoint-1", session.PhaseIdle)
	state.LastCheckpointID = "a3b2c4d5e6f7"
	if err := strategy.SaveSessionState(context.Background(), state); err != nil {
		t.Fatalf("SaveSessionState() error = %v", err)
	}

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--session", "test-stop-checkpoint-1", "--force"})

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Checkpoint: a3b2c4d5e6f7") {
		t.Errorf("expected 'Checkpoint: a3b2c4d5e6f7' in output, got: %q", out)
	}
}

func TestStopCmd_SingleSession_UncommittedWork(t *testing.T) {
	setupStopTestRepo(t)

	state := makeSessionState("test-stop-uncommitted-1", session.PhaseIdle)
	state.StepCount = 2
	if err := strategy.SaveSessionState(context.Background(), state); err != nil {
		t.Fatalf("SaveSessionState() error = %v", err)
	}

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--session", "test-stop-uncommitted-1", "--force"})

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Work will be captured in your next checkpoint.") {
		t.Errorf("expected 'Work will be captured in your next checkpoint.' in output, got: %q", out)
	}
}

func TestStopCmd_AlreadyStopped(t *testing.T) {
	setupStopTestRepo(t)

	state := makeSessionState("test-stop-already-ended-1", session.PhaseEnded)
	now := time.Now()
	state.EndedAt = &now
	if err := strategy.SaveSessionState(context.Background(), state); err != nil {
		t.Fatalf("SaveSessionState() error = %v", err)
	}

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--session", "test-stop-already-ended-1", "--force"})

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Session test-stop-already-ended-1 is already stopped.") {
		t.Errorf("expected 'is already stopped.' in output, got: %q", out)
	}

	// State should be unchanged (still ended)
	loaded, err := strategy.LoadSessionState(context.Background(), "test-stop-already-ended-1")
	if err != nil {
		t.Fatalf("LoadSessionState() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("expected session state to still exist")
	}
	if loaded.Phase != session.PhaseEnded {
		t.Errorf("expected Phase=PhaseEnded unchanged, got: %v", loaded.Phase)
	}
}

func TestStopCmd_SessionFlag(t *testing.T) {
	setupStopTestRepo(t)

	state1 := makeSessionState("test-stop-target-session", session.PhaseIdle)
	state2 := makeSessionState("test-stop-other-session", session.PhaseIdle)
	for _, s := range []*strategy.SessionState{state1, state2} {
		if err := strategy.SaveSessionState(context.Background(), s); err != nil {
			t.Fatalf("SaveSessionState() error = %v", err)
		}
	}

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--session", "test-stop-target-session", "--force"})

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	target, err := strategy.LoadSessionState(context.Background(), "test-stop-target-session")
	if err != nil {
		t.Fatalf("LoadSessionState(target) error = %v", err)
	}
	if target == nil {
		t.Fatal("expected target session state to exist")
	}
	if target.Phase != session.PhaseEnded {
		t.Errorf("expected target Phase=PhaseEnded, got: %v", target.Phase)
	}

	other, err := strategy.LoadSessionState(context.Background(), "test-stop-other-session")
	if err != nil {
		t.Fatalf("LoadSessionState(other) error = %v", err)
	}
	if other == nil {
		t.Fatal("expected other session state to exist")
	}
	if other.Phase == session.PhaseEnded {
		t.Errorf("expected other session to remain non-ended, got: %v", other.Phase)
	}
}

func TestStopCmd_AllFlag(t *testing.T) {
	setupStopTestRepo(t)

	// Resolve the worktree root as the command sees it (handles macOS symlinks like /var -> /private/var).
	ctx := context.Background()
	worktreePath, wtErr := paths.WorktreeRoot(ctx)
	if wtErr != nil {
		t.Fatalf("WorktreeRoot() error = %v", wtErr)
	}

	state1 := makeSessionState("test-stop-all-sess-1", session.PhaseIdle)
	state1.WorktreePath = worktreePath
	state2 := makeSessionState("test-stop-all-sess-2", session.PhaseIdle)
	state2.WorktreePath = worktreePath
	for _, s := range []*strategy.SessionState{state1, state2} {
		if err := strategy.SaveSessionState(ctx, s); err != nil {
			t.Fatalf("SaveSessionState() error = %v", err)
		}
	}

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--all", "--force"})

	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	out := stdout.String()
	for _, id := range []string{"test-stop-all-sess-1", "test-stop-all-sess-2"} {
		if !strings.Contains(out, id) {
			t.Errorf("expected session ID %q in output, got: %q", id, out)
		}

		loaded, err := strategy.LoadSessionState(context.Background(), id)
		if err != nil {
			t.Fatalf("LoadSessionState(%s) error = %v", id, err)
		}
		if loaded == nil {
			t.Fatalf("expected session %s to exist after stop", id)
		}
		if loaded.Phase != session.PhaseEnded {
			t.Errorf("expected session %s Phase=PhaseEnded, got: %v", id, loaded.Phase)
		}
	}
}

func TestStopCmd_AllAndSessionMutuallyExclusive(t *testing.T) {
	setupStopTestRepo(t)

	state := makeSessionState("test-stop-mutex-sess", session.PhaseIdle)
	if err := strategy.SaveSessionState(context.Background(), state); err != nil {
		t.Fatalf("SaveSessionState() error = %v", err)
	}

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--all", "--session", "test-stop-mutex-sess", "--force"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error for --all and --session together, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected error to mention 'mutually exclusive', got: %v", err)
	}

	// State should be unchanged
	loaded, err2 := strategy.LoadSessionState(context.Background(), "test-stop-mutex-sess")
	if err2 != nil {
		t.Fatalf("LoadSessionState() error = %v", err2)
	}
	if loaded == nil {
		t.Fatal("expected session state to still exist")
	}
	if loaded.Phase == session.PhaseEnded {
		t.Error("expected session to remain non-ended after mutual exclusion error")
	}
}

func TestStopCmd_SessionNotFound(t *testing.T) {
	setupStopTestRepo(t)

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--session", "doesnotexist", "--force"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error for unknown session, got nil")
	}

	var silentErr *SilentError
	if !errors.As(err, &silentErr) {
		t.Errorf("expected SilentError, got: %T %v", err, err)
	}

	if !strings.Contains(stderr.String(), "Session not found.") {
		t.Errorf("expected 'Session not found.' in stderr, got: %q", stderr.String())
	}
}

func TestStopCmd_NotGitRepo(t *testing.T) {
	// Use a plain temp dir with no git init
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	paths.ClearWorktreeRootCache()
	session.ClearGitCommonDirCache()

	cmd := newStopCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error for non-git directory, got nil")
	}

	var silentErr *SilentError
	if !errors.As(err, &silentErr) {
		t.Errorf("expected SilentError, got: %T %v", err, err)
	}

	if !strings.Contains(stderr.String(), "Not a git repository.") {
		t.Errorf("expected 'Not a git repository.' in stderr, got: %q", stderr.String())
	}
}

func TestStopCmd_MultiSession_NoFlags(t *testing.T) {
	setupStopTestRepo(t)

	// Create two active sessions. The TUI multi-select would normally hang,
	// so we do NOT execute the command. We just verify the session setup is
	// consistent: both sessions are non-ended.
	state1 := makeSessionState("test-stop-multi-sess-1", session.PhaseIdle)
	state2 := makeSessionState("test-stop-multi-sess-2", session.PhaseIdle)
	for _, s := range []*strategy.SessionState{state1, state2} {
		if err := strategy.SaveSessionState(context.Background(), s); err != nil {
			t.Fatalf("SaveSessionState() error = %v", err)
		}
	}

	// Verify both sessions exist and are non-ended, so the multi-select path
	// would be triggered by the command (not the no-sessions path).
	for _, id := range []string{"test-stop-multi-sess-1", "test-stop-multi-sess-2"} {
		loaded, err := strategy.LoadSessionState(context.Background(), id)
		if err != nil {
			t.Fatalf("LoadSessionState(%s) error = %v", id, err)
		}
		if loaded == nil {
			t.Fatalf("expected session %s to exist", id)
		}
		if loaded.Phase == session.PhaseEnded {
			t.Errorf("expected session %s to be non-ended, got PhaseEnded", id)
		}
	}
}
