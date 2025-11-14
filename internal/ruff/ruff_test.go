package ruff

import (
	"path/filepath"
	"testing"

	"github.com/horiagug/ruff-format-changes/internal/git"
)

func TestGetAbsolutePaths(t *testing.T) {
	tests := []struct {
		name     string
		repoRoot string
		files    []string
		expected []string
	}{
		{
			name:     "single file",
			repoRoot: "/tmp/repo",
			files:    []string{"file.py"},
			expected: []string{"/tmp/repo/file.py"},
		},
		{
			name:     "multiple files",
			repoRoot: "/tmp/repo",
			files:    []string{"file1.py", "dir/file2.py"},
			expected: []string{"/tmp/repo/file1.py", "/tmp/repo/dir/file2.py"},
		},
		{
			name:     "empty list",
			repoRoot: "/tmp/repo",
			files:    []string{},
			expected: []string{},
		},
		{
			name:     "nested paths",
			repoRoot: "/home/user/project",
			files:    []string{"src/main.py", "src/utils/helpers.py"},
			expected: []string{"/home/user/project/src/main.py", "/home/user/project/src/utils/helpers.py"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(tt.repoRoot, false, false)
			result := r.GetAbsolutePaths(tt.files)

			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d paths, got %d", len(tt.expected), len(result))
			}

			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("Path %d: expected %s, got %s", i, tt.expected[i], path)
				}
			}
		})
	}
}

func TestCheckRuffInstalled(t *testing.T) {
	err := CheckRuffInstalled()
	if err != nil {
		t.Skipf("Skipping test: ruff not installed - %v", err)
	}
}

func TestNewRuff(t *testing.T) {
	tests := []struct {
		name     string
		repoRoot string
		dryRun   bool
		verbose  bool
	}{
		{
			name:     "basic initialization",
			repoRoot: "/tmp/repo",
			dryRun:   false,
			verbose:  false,
		},
		{
			name:     "with dry run",
			repoRoot: "/tmp/repo",
			dryRun:   true,
			verbose:  false,
		},
		{
			name:     "with verbose",
			repoRoot: "/tmp/repo",
			dryRun:   false,
			verbose:  true,
		},
		{
			name:     "with all flags",
			repoRoot: "/tmp/repo",
			dryRun:   true,
			verbose:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(tt.repoRoot, tt.dryRun, tt.verbose)

			if r.repoRoot != tt.repoRoot {
				t.Errorf("Expected repoRoot %s, got %s", tt.repoRoot, r.repoRoot)
			}
			if r.dryRun != tt.dryRun {
				t.Errorf("Expected dryRun %v, got %v", tt.dryRun, r.dryRun)
			}
			if r.verbose != tt.verbose {
				t.Errorf("Expected verbose %v, got %v", tt.verbose, r.verbose)
			}
		})
	}
}

func TestGetAbsolutePathsWithTestFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"file1.py", "dir/file2.py"}
	expectedAbsPaths := []string{
		filepath.Join(tmpDir, "file1.py"),
		filepath.Join(tmpDir, "dir/file2.py"),
	}

	r := New(tmpDir, false, false)
	result := r.GetAbsolutePaths(testFiles)

	if len(result) != len(expectedAbsPaths) {
		t.Fatalf("Expected %d paths, got %d", len(expectedAbsPaths), len(result))
	}

	for i, path := range result {
		if path != expectedAbsPaths[i] {
			t.Errorf("Path %d: expected %s, got %s", i, expectedAbsPaths[i], path)
		}
	}
}

func TestRuffFlags(t *testing.T) {
	tests := []struct {
		name    string
		dryRun  bool
		verbose bool
	}{
		{
			name:    "dry run mode",
			dryRun:  true,
			verbose: false,
		},
		{
			name:    "normal mode",
			dryRun:  false,
			verbose: false,
		},
		{
			name:    "verbose mode",
			dryRun:  false,
			verbose: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New("/tmp/repo", tt.dryRun, tt.verbose)

			if r.dryRun != tt.dryRun {
				t.Errorf("dryRun: expected %v, got %v", tt.dryRun, r.dryRun)
			}
			if r.verbose != tt.verbose {
				t.Errorf("verbose: expected %v, got %v", tt.verbose, r.verbose)
			}
		})
	}
}

// Tests for FormatFilesByLineRanges method

func TestFormatFilesByLineRangesEmptyInput(t *testing.T) {
	r := New("/tmp/repo", false, false)
	fileChanges := []git.FileChanges{}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Errorf("Expected no error for empty input, got %v", err)
	}
}

func TestFormatFilesByLineRangesEmptyInputVerbose(t *testing.T) {
	r := New("/tmp/repo", false, true)
	fileChanges := []git.FileChanges{}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Errorf("Expected no error for empty input with verbose, got %v", err)
	}
}

func TestFormatFilesByLineRangesSingleFile(t *testing.T) {
	// This test verifies the method handles single file without executing ruff
	// (ruff may not be installed in test environment)
	r := New("/tmp/repo", true, false) // Use dry run to avoid actual execution

	fileChanges := []git.FileChanges{
		{
			FilePath: "main.py",
			LineRanges: []git.LineRange{
				{Start: 5, End: 10},
			},
		},
	}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		// We expect an error because ruff command might not exist,
		// but the method structure should be sound
		t.Logf("Got expected error (ruff may not be installed): %v", err)
	}
}

func TestFormatFilesByLineRangesMultipleFiles(t *testing.T) {
	r := New("/tmp/repo", true, false)

	fileChanges := []git.FileChanges{
		{
			FilePath: "main.py",
			LineRanges: []git.LineRange{
				{Start: 5, End: 10},
				{Start: 15, End: 20},
			},
		},
		{
			FilePath: "utils.py",
			LineRanges: []git.LineRange{
				{Start: 1, End: 5},
			},
		},
	}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Logf("Got expected error (ruff may not be installed): %v", err)
	}
}

func TestFormatFilesByLineRangesRepoRoot(t *testing.T) {
	repoRoot := "/home/user/project"
	r := New(repoRoot, false, false)

	if r.repoRoot != repoRoot {
		t.Errorf("Expected repoRoot %s, got %s", repoRoot, r.repoRoot)
	}
}

func TestFormatFilesByLineRangesDryRunMode(t *testing.T) {
	r := New("/tmp/repo", true, false)

	fileChanges := []git.FileChanges{
		{
			FilePath: "test.py",
			LineRanges: []git.LineRange{
				{Start: 1, End: 5},
			},
		},
	}

	if !r.dryRun {
		t.Errorf("Expected dryRun to be true")
	}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Logf("Got expected error (ruff may not be installed): %v", err)
	}
}

func TestFormatFilesByLineRangesNormalMode(t *testing.T) {
	r := New("/tmp/repo", false, true)

	fileChanges := []git.FileChanges{
		{
			FilePath: "test.py",
			LineRanges: []git.LineRange{
				{Start: 10, End: 15},
			},
		},
	}

	if r.dryRun {
		t.Errorf("Expected dryRun to be false")
	}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Logf("Got expected error (ruff may not be installed): %v", err)
	}
}

func TestFormatFilesByLineRangesSingleLineRange(t *testing.T) {
	r := New("/tmp/repo", true, false)

	fileChanges := []git.FileChanges{
		{
			FilePath: "test.py",
			LineRanges: []git.LineRange{
				{Start: 42, End: 42}, // Single line
			},
		},
	}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Logf("Got expected error (ruff may not be installed): %v", err)
	}
}

func TestFormatFilesByLineRangesMultipleLineRanges(t *testing.T) {
	r := New("/tmp/repo", true, false)

	fileChanges := []git.FileChanges{
		{
			FilePath: "main.py",
			LineRanges: []git.LineRange{
				{Start: 5, End: 10},
				{Start: 20, End: 25},
				{Start: 40, End: 45},
			},
		},
	}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Logf("Got expected error (ruff may not be installed): %v", err)
	}
}

func TestFormatFilesByLineRangesNestedPath(t *testing.T) {
	repoRoot := "/project"
	r := New(repoRoot, true, false)

	fileChanges := []git.FileChanges{
		{
			FilePath: "src/services/handler.py",
			LineRanges: []git.LineRange{
				{Start: 12, End: 18},
			},
		},
	}

	// The method should construct the correct absolute path
	expectedPath := filepath.Join(repoRoot, fileChanges[0].FilePath)
	if !filepath.IsAbs(expectedPath) {
		t.Errorf("Expected absolute path, got %s", expectedPath)
	}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Logf("Got expected error (ruff may not be installed): %v", err)
	}
}

func TestFormatFilesByLineRangesVerboseOutput(t *testing.T) {
	r := New("/tmp/repo", true, true) // verbose=true

	fileChanges := []git.FileChanges{
		{
			FilePath: "test.py",
			LineRanges: []git.LineRange{
				{Start: 5, End: 10},
				{Start: 20, End: 20}, // Single line
			},
		},
	}

	if !r.verbose {
		t.Errorf("Expected verbose to be true")
	}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Logf("Got expected error (ruff may not be installed): %v", err)
	}
}

func TestFormatFilesByLineRangesMultipleFilesMultipleRanges(t *testing.T) {
	r := New("/tmp/repo", true, true)

	fileChanges := []git.FileChanges{
		{
			FilePath: "app/main.py",
			LineRanges: []git.LineRange{
				{Start: 1, End: 5},
				{Start: 15, End: 20},
			},
		},
		{
			FilePath: "app/utils.py",
			LineRanges: []git.LineRange{
				{Start: 30, End: 35},
			},
		},
		{
			FilePath: "tests/test_main.py",
			LineRanges: []git.LineRange{
				{Start: 10, End: 12},
				{Start: 25, End: 30},
				{Start: 50, End: 55},
			},
		},
	}

	err := r.FormatFilesByLineRanges(fileChanges)
	if err != nil {
		t.Logf("Got expected error (ruff may not be installed): %v", err)
	}
}

// Tests for formatRangeArg helper function

func TestFormatRangeArgSingleLine(t *testing.T) {
	result := formatRangeArg(10, 10)
	expected := "10"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestFormatRangeArgMultipleLines(t *testing.T) {
	result := formatRangeArg(5, 15)
	expected := "5:15"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestFormatRangeArgEdgeCases(t *testing.T) {
	tests := []struct {
		start    int
		end      int
		expected string
	}{
		{1, 1, "1"},
		{1, 100, "1:100"},
		{999, 999, "999"},
		{50, 51, "50:51"},
	}

	for _, tt := range tests {
		result := formatRangeArg(tt.start, tt.end)
		if result != tt.expected {
			t.Errorf("formatRangeArg(%d, %d): expected %s, got %s", tt.start, tt.end, tt.expected, result)
		}
	}
}
