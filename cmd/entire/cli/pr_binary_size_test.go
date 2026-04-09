package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/testutil"
	"github.com/stretchr/testify/require"
)

func TestCheckPRBinaries_AddThenDeleteStillFails(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	testutil.InitRepo(t, repoDir)
	testutil.WriteFile(t, repoDir, "README.md", "init\n")
	testutil.GitAdd(t, repoDir, "README.md")
	testutil.GitCommit(t, repoDir, "init")
	baseSHA := testutil.GetHeadHash(t, repoDir)

	largeBinary := bytes.Repeat([]byte{0}, 1048577)
	err := os.WriteFile(filepath.Join(repoDir, "oversized.bin"), largeBinary, 0o644)
	require.NoError(t, err)
	testutil.GitAdd(t, repoDir, "oversized.bin")
	testutil.GitCommit(t, repoDir, "add oversized binary")

	headWithBinary := testutil.GetHeadHash(t, repoDir)
	output, err := runBinaryCheckScript(t, repoDir, baseSHA, headWithBinary)
	require.Error(t, err)
	require.Contains(t, output, "oversized.bin")

	runGitCommand(t, repoDir, "rm", "oversized.bin")
	testutil.GitCommit(t, repoDir, "delete oversized binary")

	output, err = runBinaryCheckScript(t, repoDir, baseSHA, testutil.GetHeadHash(t, repoDir))
	require.Error(t, err)
	require.Contains(t, output, "oversized.bin")
}

func runBinaryCheckScript(t *testing.T, repoDir, baseSHA, headSHA string) (string, error) {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)

	scriptPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "scripts", "check-pr-binaries.sh")
	cmd := exec.CommandContext(context.Background(), "bash", scriptPath, baseSHA, headSHA)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runGitCommand(t *testing.T, repoDir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v failed: %s", args, output)
}
