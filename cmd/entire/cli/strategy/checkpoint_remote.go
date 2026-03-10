package strategy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/logging"
	"github.com/entireio/cli/cmd/entire/cli/paths"
	"github.com/entireio/cli/cmd/entire/cli/settings"

	"github.com/go-git/go-git/v6/plumbing"
)

// checkpointRemoteName is the git remote name used for the dedicated checkpoint remote.
const checkpointRemoteName = "entire-checkpoints"

// checkpointRemoteFetchTimeout is the timeout for fetching branches from the remote.
const checkpointRemoteFetchTimeout = 30 * time.Second

// resolveCheckpointRemote determines the remote to use for checkpoint branch operations.
// If a checkpoint_remote URL is configured in settings:
//   - Ensures a git remote named "entire-checkpoints" is configured with that URL
//   - If a checkpoint branch doesn't exist locally, attempts to fetch it from the remote
//   - Returns "entire-checkpoints" as the remote name
//
// The push itself handles failures gracefully (doPushBranch warns and continues),
// so no reachability check is needed here. This avoids adding latency on every push
// when the remote is temporarily unreachable.
//
// Falls back to the provided defaultRemote only if:
//   - No checkpoint_remote is configured
//   - The git remote could not be created/updated (local git config error)
func resolveCheckpointRemote(ctx context.Context, defaultRemote string) string {
	s, err := settings.Load(ctx)
	if err != nil {
		return defaultRemote
	}

	remoteURL := s.GetCheckpointRemote()
	if remoteURL == "" {
		return defaultRemote
	}

	// Ensure the git remote exists with the correct URL (local operation, no network)
	if err := ensureGitRemote(ctx, checkpointRemoteName, remoteURL); err != nil {
		logging.Warn(ctx, "checkpoint-remote: failed to configure git remote",
			slog.String("url", remoteURL),
			slog.String("error", err.Error()),
		)
		return defaultRemote
	}

	// If checkpoint branches don't exist locally, try to fetch them from the remote.
	// This is a one-time operation per branch - once the branch exists locally,
	// subsequent pushes skip the fetch entirely.
	for _, branchName := range []string{paths.MetadataBranchName, paths.TrailsBranchName} {
		if err := fetchBranchIfMissing(ctx, checkpointRemoteName, branchName); err != nil {
			logging.Warn(ctx, "checkpoint-remote: failed to fetch branch",
				slog.String("branch", branchName),
				slog.String("error", err.Error()),
			)
		}
	}

	return checkpointRemoteName
}

// ensureGitRemote creates or updates a git remote to point to the given URL.
// This is a local-only operation (no network calls).
func ensureGitRemote(ctx context.Context, name, url string) error {
	// Check if remote already exists and get its current URL
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", name)
	output, err := cmd.Output()
	if err != nil {
		// Remote doesn't exist, create it
		addCmd := exec.CommandContext(ctx, "git", "remote", "add", name, url)
		if addErr := addCmd.Run(); addErr != nil {
			return fmt.Errorf("failed to add remote: %w", addErr)
		}
		return nil
	}

	// Remote exists, check if URL matches
	currentURL := strings.TrimSpace(string(output))
	if currentURL == url {
		return nil
	}

	// URL differs, update it
	setCmd := exec.CommandContext(ctx, "git", "remote", "set-url", name, url)
	if setErr := setCmd.Run(); setErr != nil {
		return fmt.Errorf("failed to update remote URL: %w", setErr)
	}

	return nil
}

// fetchBranchIfMissing fetches a branch from the remote only if it doesn't exist locally.
// This avoids network calls on every push - once the branch exists locally, this is a no-op.
// If the fetch fails (remote unreachable, branch doesn't exist on remote), the error is
// returned but the caller should treat it as non-fatal: the push will handle it.
func fetchBranchIfMissing(ctx context.Context, remote, branchName string) error {
	repo, err := OpenRepository(ctx)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Check if branch already exists locally - if so, nothing to do
	branchRef := plumbing.NewBranchReferenceName(branchName)
	if _, err := repo.Reference(branchRef, true); err == nil {
		return nil // Branch exists locally, skip fetch
	}

	// Branch doesn't exist locally - try to fetch it from the remote
	fetchCtx, cancel := context.WithTimeout(ctx, checkpointRemoteFetchTimeout)
	defer cancel()

	refSpec := fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", branchName, remote, branchName)
	fetchCmd := exec.CommandContext(fetchCtx, "git", "fetch", "--no-tags", remote, refSpec)
	fetchCmd.Stdin = nil
	fetchCmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0", // Prevent interactive auth prompts
	)
	if err := fetchCmd.Run(); err != nil {
		// Fetch failed - remote may be unreachable or branch doesn't exist there yet.
		// Not fatal: push will create it on the remote when it succeeds.
		return nil
	}

	// Fetch succeeded - create local branch from the remote ref
	remoteRefName := plumbing.NewRemoteReferenceName(remote, branchName)
	remoteRef, err := repo.Reference(remoteRefName, true)
	if err != nil {
		// Fetch succeeded but remote ref not found - branch may not exist on remote
		return nil
	}

	newRef := plumbing.NewHashReference(branchRef, remoteRef.Hash())
	if err := repo.Storer.SetReference(newRef); err != nil {
		return fmt.Errorf("failed to create local branch from remote: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[entire] Fetched %s from %s\n", branchName, remote)
	return nil
}
