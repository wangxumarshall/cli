package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/entireio/cli/cmd/entire/cli/paths"
	"github.com/entireio/cli/cmd/entire/cli/session"
	"github.com/entireio/cli/cmd/entire/cli/strategy"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	var sessionFlag string
	var allFlag bool
	var forceFlag bool

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop one or more active sessions",
		Long: `Mark one or more active sessions as ended.

This is a pure state mutation — no checkpoints are written, no condensation happens.

Examples:
  entire stop                     Stop the one active session, or show selector if multiple
  entire stop --session <id>      Stop a specific session by ID
  entire stop --all               Stop all active sessions in current worktree
  entire stop --force             Skip confirmation prompt`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			if allFlag && sessionFlag != "" {
				return errors.New("--all and --session are mutually exclusive")
			}

			// Check if in git repository
			if _, err := paths.WorktreeRoot(ctx); err != nil {
				cmd.SilenceUsage = true
				fmt.Fprintln(cmd.ErrOrStderr(), "Not a git repository.")
				return NewSilentError(errors.New("not a git repository"))
			}

			return runStop(ctx, cmd, sessionFlag, allFlag, forceFlag)
		},
	}

	cmd.Flags().StringVar(&sessionFlag, "session", "", "Stop a specific session by ID")
	cmd.Flags().BoolVar(&allFlag, "all", false, "Stop all active sessions in current worktree")
	cmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// runStop is the main logic for the stop command.
func runStop(ctx context.Context, cmd *cobra.Command, sessionID string, all, force bool) error {
	// --session path: stop a specific session
	if sessionID != "" {
		return runStopSession(ctx, cmd, sessionID, force)
	}

	// List all session states
	states, err := strategy.ListSessionStates(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	// Filter to active sessions (Phase != PhaseEnded)
	activeSessions := filterActiveSessions(states)

	if len(activeSessions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No active sessions.")
		return nil
	}

	// --all path: stop all active sessions in current worktree
	if all {
		return runStopAll(ctx, cmd, activeSessions, force)
	}

	// No flags: one active session → confirm + stop; multiple → TUI selector
	if len(activeSessions) == 1 {
		return runStopSession(ctx, cmd, activeSessions[0].SessionID, force)
	}

	// Multiple active sessions: show TUI multi-select
	return runStopMultiSelect(ctx, cmd, activeSessions, force)
}

// filterActiveSessions returns sessions where Phase != PhaseEnded.
func filterActiveSessions(states []*strategy.SessionState) []*strategy.SessionState {
	var active []*strategy.SessionState
	for _, s := range states {
		if s.Phase != session.PhaseEnded {
			active = append(active, s)
		}
	}
	return active
}

// runStopSession stops a single session by ID, with optional confirmation.
func runStopSession(ctx context.Context, cmd *cobra.Command, sessionID string, force bool) error {
	state, err := strategy.LoadSessionState(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	if state == nil {
		cmd.SilenceUsage = true
		fmt.Fprintln(cmd.ErrOrStderr(), "Session not found.")
		return NewSilentError(fmt.Errorf("session not found: %s", sessionID))
	}

	if state.Phase == session.PhaseEnded {
		fmt.Fprintf(cmd.OutOrStdout(), "Session %s is already stopped.\n", sessionID)
		return nil
	}

	if !force {
		var confirmed bool
		form := NewAccessibleForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Stop session %s?", sessionID)).
					Value(&confirmed),
			),
		)
		if err := form.Run(); err != nil {
			return handleFormCancellation(cmd.OutOrStdout(), "Stop", err)
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Stop cancelled.")
			return nil
		}
	}

	return stopSessionAndPrint(ctx, cmd, state)
}

// runStopAll stops all active sessions scoped to the current worktree.
func runStopAll(ctx context.Context, cmd *cobra.Command, activeSessions []*strategy.SessionState, force bool) error {
	// Scope to current worktree: include sessions where WorktreePath matches
	// or WorktreePath is empty (safer than silently excluding).
	worktreePath, err := paths.WorktreeRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve worktree root: %w", err)
	}

	var toStop []*strategy.SessionState
	for _, s := range activeSessions {
		if s.WorktreePath == worktreePath || s.WorktreePath == "" {
			toStop = append(toStop, s)
		}
	}

	if len(toStop) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No active sessions.")
		return nil
	}

	if !force {
		var confirmed bool
		form := NewAccessibleForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Stop %d session(s)?", len(toStop))).
					Value(&confirmed),
			),
		)
		if err := form.Run(); err != nil {
			return handleFormCancellation(cmd.OutOrStdout(), "Stop", err)
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Stop cancelled.")
			return nil
		}
	}

	var stopErr error
	for _, s := range toStop {
		if err := stopSessionAndPrint(ctx, cmd, s); err != nil {
			stopErr = err
		}
	}
	return stopErr
}

// runStopMultiSelect shows a TUI multi-select for multiple active sessions.
func runStopMultiSelect(ctx context.Context, cmd *cobra.Command, activeSessions []*strategy.SessionState, force bool) error {
	options := make([]huh.Option[string], len(activeSessions))
	for i, s := range activeSessions {
		label := fmt.Sprintf("%s · %s", s.AgentType, s.SessionID)
		if s.LastPrompt != "" {
			label = fmt.Sprintf("%s · %q", label, s.LastPrompt)
		}
		options[i] = huh.NewOption(label, s.SessionID)
	}

	var selectedIDs []string
	form := NewAccessibleForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select sessions to stop").
				Description("Use space to select, enter to confirm.").
				Options(options...).
				Value(&selectedIDs),
		),
	)
	if err := form.Run(); err != nil {
		return handleFormCancellation(cmd.OutOrStdout(), "Stop", err)
	}

	if len(selectedIDs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Stop cancelled.")
		return nil
	}

	// Build a map for quick lookup
	stateByID := make(map[string]*strategy.SessionState, len(activeSessions))
	for _, s := range activeSessions {
		stateByID[s.SessionID] = s
	}

	// Confirm only if not forcing
	if !force {
		var confirmed bool
		form := NewAccessibleForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Stop %d session(s)?", len(selectedIDs))).
					Value(&confirmed),
			),
		)
		if err := form.Run(); err != nil {
			return handleFormCancellation(cmd.OutOrStdout(), "Stop", err)
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Stop cancelled.")
			return nil
		}
	}

	var stopErr error
	for _, id := range selectedIDs {
		s, ok := stateByID[id]
		if !ok {
			continue
		}
		if err := stopSessionAndPrint(ctx, cmd, s); err != nil {
			stopErr = err
		}
	}
	return stopErr
}

// stopSessionAndPrint stops a session and prints the result.
// It snapshots the needed fields before calling markSessionEnded.
func stopSessionAndPrint(ctx context.Context, cmd *cobra.Command, state *strategy.SessionState) error {
	// Snapshot fields needed for output before calling markSessionEnded
	sessionID := state.SessionID
	lastCheckpointID := state.LastCheckpointID
	stepCount := state.StepCount

	if err := markSessionEnded(ctx, nil, sessionID); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Session %s stopped.\n", sessionID)
	switch {
	case lastCheckpointID != "":
		fmt.Fprintf(cmd.OutOrStdout(), "  Checkpoint: %s\n", lastCheckpointID)
	case stepCount > 0:
		fmt.Fprintln(cmd.OutOrStdout(), "  Work will be captured in your next checkpoint.")
	default:
		fmt.Fprintln(cmd.OutOrStdout(), "  No work recorded.")
	}
	return nil
}
