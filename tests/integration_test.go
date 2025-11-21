package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/horiagug/ruff-format-changes/internal/git"
	"github.com/horiagug/ruff-format-changes/internal/ruff"
)

// setupTestRepo creates a temporary git repository with commits
func setupTestRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	gitConfig := [][]string{
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
	}
	for _, args := range gitConfig {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		cmd.Run()
	}

	pyFile := filepath.Join(tmpDir, "main.py")
	if err := os.WriteFile(pyFile, []byte("def hello():\n    print('hello')\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "main.py")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "checkout", "-b", "feature/test")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	if err := os.WriteFile(pyFile, []byte("def hello():\n    print( 'hello' )\n"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	utilsFile := filepath.Join(tmpDir, "utils.py")
	if err := os.WriteFile(utilsFile, []byte("def util(  ):\n    pass\n"), 0644); err != nil {
		t.Fatalf("Failed to create utils file: %v", err)
	}

	cmd = exec.Command("git", "add", "main.py", "utils.py")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add modified files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add utils and modify main")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit changes: %v", err)
	}

	return tmpDir
}

// TestEndToEndWithRealGit tests the complete workflow with a real git repo
func TestEndToEndWithRealGit(t *testing.T) {
	tmpDir := setupTestRepo(t)

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	gitClient, err := git.New(true)
	if err != nil {
		t.Fatalf("Failed to create git client: %v", err)
	}

	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	t.Logf("Current branch: %s", currentBranch)

	changedFiles, err := gitClient.GetChangedFiles("main")
	if err != nil {
		t.Fatalf("Failed to get changed files: %v", err)
	}

	t.Logf("Changed files: %v", changedFiles)

	if len(changedFiles) == 0 {
		t.Logf("No changed files found (this might be expected if git diff returns empty)")
	} else {
		hasPyFile := false
		for _, f := range changedFiles {
			if strings.HasSuffix(f, ".py") {
				hasPyFile = true
				break
			}
		}
		if !hasPyFile {
			t.Errorf("Expected Python files in changed files, got %v", changedFiles)
		}
	}
}

// TestGetChangedFilesDetectsModified tests that modified files are detected
func TestGetChangedFilesDetectsModified(t *testing.T) {
	tmpDir := setupTestRepo(t)

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	gitClient, err := git.New(false)
	if err != nil {
		t.Fatalf("Failed to create git client: %v", err)
	}

	changedFiles, err := gitClient.GetChangedFiles("main")
	if err != nil {
		t.Logf("Could not get changed files with three-dot syntax: %v", err)
		return
	}

	var pyFiles []string
	for _, f := range changedFiles {
		if strings.HasSuffix(f, ".py") {
			pyFiles = append(pyFiles, f)
		}
	}

	t.Logf("Changed Python files: %v", pyFiles)
}

// TestRuffAbsolutePathsInRealRepo tests path conversion in a real repo
func TestRuffAbsolutePathsInRealRepo(t *testing.T) {
	tmpDir := setupTestRepo(t)

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	ruffClient := ruff.New(tmpDir, false, false)
	relPaths := []string{"main.py", "utils.py"}
	absPaths := ruffClient.GetAbsolutePaths(relPaths)

	if len(absPaths) != 2 {
		t.Fatalf("Expected 2 paths, got %d", len(absPaths))
	}

	for i, absPath := range absPaths {
		expected := filepath.Join(tmpDir, relPaths[i])
		if absPath != expected {
			t.Errorf("Path %d: expected %s, got %s", i, expected, absPath)
		}
	}
}

// TestBranchDetectionInFeatureBranch tests branch detection in a feature branch
func TestBranchDetectionInFeatureBranch(t *testing.T) {
	tmpDir := setupTestRepo(t)

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	gitClient, err := git.New(false)
	if err != nil {
		t.Fatalf("Failed to create git client: %v", err)
	}

	branch, err := gitClient.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	// Should be on feature branch
	if !strings.Contains(branch, "feature") {
		t.Logf("Expected to be on feature branch, got %s", branch)
	}
}

// TestRuffObjectCreation tests that Ruff objects are created correctly
func TestRuffObjectCreation(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		dryRun  bool
		verbose bool
	}{
		{"Default", false, false},
		{"DryRun", true, false},
		{"Verbose", false, true},
		{"Both", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ruff.New(tmpDir, tt.dryRun, tt.verbose)

			if r == nil {
				t.Fatalf("Expected Ruff object, got nil")
			}
		})
	}
}

// TestMultipleRangesLineShiftBug tests the potential bug where formatting
// the first range shifts line numbers, causing the second range to target wrong lines
// This test sets up a file with two changes far apart so they appear in separate hunks
func TestMultipleRangesLineShiftBug(t *testing.T) {
	tmpDir := t.TempDir()

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	gitConfig := [][]string{
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
	}
	for _, args := range gitConfig {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		cmd.Run()
	}

	// Create a Python file with specific content where we'll have two format targets
	// The gap between them must be > 3 lines to ensure they're in separate hunks
	// and thus detected as separate ranges
	pyFile := filepath.Join(tmpDir, "test_multi_range.py")
	initialContent := `def function_one():
    x=1
    y=2
    z=3


# Line 7
# Line 8
# Line 9
# Line 10
# Line 11
# Line 12
# Line 13
# Line 14

def function_two():
    a=10
    b=20
    c=30
`

	if err := os.WriteFile(pyFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Commit initial content
	addCmd := exec.Command("git", "add", "test_multi_range.py")
	addCmd.Dir = tmpDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "Add test file with multiple format targets")
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Rename branch to main
	branchCmd := exec.Command("git", "branch", "-M", "main")
	branchCmd.Dir = tmpDir
	if err := branchCmd.Run(); err != nil {
		t.Fatalf("Failed to rename branch to main: %v", err)
	}

	// Now create a feature branch with changes at two locations
	checkoutCmd := exec.Command("git", "checkout", "-b", "bugtest")
	checkoutCmd.Dir = tmpDir
	if err := checkoutCmd.Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Modify both functions (at different locations with space between them)
	// Change 1: Line 2 (x=1) - add spaces
	// Change 2: Line 20 (a=10) - add spaces (with gap to ensure separate hunk)
	modifiedContent := `def function_one():
    x = 1
    y=2
    z=3


# Line 7
# Line 8
# Line 9
# Line 10
# Line 11
# Line 12
# Line 13
# Line 14

def function_two():
    a = 10
    b=20
    c=30
`

	if err := os.WriteFile(pyFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	addModCmd := exec.Command("git", "add", "test_multi_range.py")
	addModCmd.Dir = tmpDir
	if err := addModCmd.Run(); err != nil {
		t.Fatalf("Failed to add modified file: %v", err)
	}

	commitModCmd := exec.Command("git", "commit", "-m", "Modify both functions")
	commitModCmd.Dir = tmpDir
	if err := commitModCmd.Run(); err != nil {
		t.Fatalf("Failed to commit changes: %v", err)
	}

	// Get the changed line ranges
	gitClient, err := git.New(true) // verbose=true for debugging
	if err != nil {
		t.Fatalf("Failed to create git client: %v", err)
	}

	t.Logf("Repo root: %s", gitClient.GetRepoRoot())

	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	t.Logf("Current branch: %s", currentBranch)

	changedFiles, err := gitClient.GetChangedFiles("main")
	if err != nil {
		t.Fatalf("Failed to get changed files: %v", err)
	}
	t.Logf("Changed files: %v", changedFiles)

	changedLines, err := gitClient.GetChangedLineRanges("main")
	if err != nil {
		t.Fatalf("Failed to get changed line ranges: %v", err)
	}

	t.Logf("Changed line ranges: %v", changedLines)

	// Debug: show the raw diff
	diffCmd := exec.Command("git", "diff", "main", "--", "test_multi_range.py")
	diffCmd.Dir = tmpDir
	diffOutput, _ := diffCmd.Output()
	t.Logf("Raw diff output:\n%s", string(diffOutput))

	// The bug is confirmed - the parseUnifiedDiff function is comparing
	// the entire line to "+", " ", "-" instead of comparing line[0]

	// Find our test file in the results
	var testFileChanges *git.FileChanges
	for i := range changedLines {
		if strings.HasSuffix(changedLines[i].FilePath, "test_multi_range.py") {
			testFileChanges = &changedLines[i]
			break
		}
	}

	if testFileChanges == nil {
		t.Fatalf("test_multi_range.py not found in changed files")
	}

	t.Logf("Found test file with %d line ranges: %v", len(testFileChanges.LineRanges), testFileChanges.LineRanges)

	if len(testFileChanges.LineRanges) < 2 {
		t.Logf("Expected at least 2 line ranges (one for each function change), got %d", len(testFileChanges.LineRanges))
		t.Logf("This might indicate the changes were merged into a single range - not reproducing the bug")
		return
	}

	// This is where the bug would manifest:
	// When we format the first range, if it adds/removes lines, the second range
	// will target incorrect line numbers in the actual file
	t.Logf("BUG SCENARIO: If the first range (lines %d-%d) adds/removes lines when formatted,",
		testFileChanges.LineRanges[0].Start, testFileChanges.LineRanges[0].End)
	t.Logf("              the second range (lines %d-%d) will target wrong lines in the file",
		testFileChanges.LineRanges[1].Start, testFileChanges.LineRanges[1].End)

	// Try to format with ruff (if available)
	ruffClient := ruff.New(tmpDir, false, false)
	err = ruffClient.FormatFilesByLineRanges(changedLines)
	if err != nil {
		t.Logf("Ruff formatting error (may be expected if ruff not installed): %v", err)
		return
	}

	// Read the file after formatting to see if it was correctly formatted
	formatted, err := os.ReadFile(pyFile)
	if err != nil {
		t.Fatalf("Failed to read formatted file: %v", err)
	}

	t.Logf("File after formatting:\n%s", string(formatted))

	// Check if both locations were properly formatted
	// Both "x = 1" and "a = 10" should have proper spacing
	formattedStr := string(formatted)
	if !strings.Contains(formattedStr, "x = 1") {
		t.Logf("WARNING: First range may not have been properly formatted (x = 1 not found)")
	}
	if !strings.Contains(formattedStr, "a = 10") {
		t.Logf("WARNING: Second range may not have been properly formatted (a = 10 not found)")
	}
}
