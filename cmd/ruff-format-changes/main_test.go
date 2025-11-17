package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)


// TestBranchExists tests the branchExists function
func TestBranchExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	if err := createEmptyCommit("main"); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	tests := []struct {
		name       string
		branch     string
		shouldExist bool
		setup      func() error
	}{
		{
			name:        "main branch exists",
			branch:      "main",
			shouldExist: true,
			setup:       nil,
		},
		{
			name:        "non-existent branch",
			branch:      "does-not-exist",
			shouldExist: false,
			setup:       nil,
		},
		{
			name:        "master branch exists after creating it",
			branch:      "master",
			shouldExist: true,
			setup: func() error {
				return exec.Command("git", "checkout", "-b", "master").Run()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			exec.Command("git", "checkout", "main").Run()

			exists := branchExists(tt.branch)
			if exists != tt.shouldExist {
				t.Errorf("branchExists(%q) = %v, want %v", tt.branch, exists, tt.shouldExist)
			}
		})
	}
}

// TestDetermineBaseBranchWithMaster tests branch detection when master is the primary branch
func TestDetermineBaseBranchWithMaster(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	if err := createEmptyCommit("master"); err != nil {
		t.Fatalf("Failed to create master commit: %v", err)
	}

	if err := exec.Command("git", "checkout", "-b", "master").Run(); err != nil {
		exec.Command("git", "checkout", "master").Run()
	}

	if err := exec.Command("git", "checkout", "-b", "feature/test").Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	baseBranch := determineBaseBranch(nil)

	if baseBranch != "master" && baseBranch != "main" {
		t.Errorf("determineBaseBranch() = %q, want either 'master' or 'main' (found master exists)", baseBranch)
	}

	if !branchExists("master") {
		t.Errorf("Expected master branch to exist")
	}
}

// TestDetermineBaseBranchWithMain tests branch detection when main is the primary branch
func TestDetermineBaseBranchWithMain(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	if err := createEmptyCommit("main"); err != nil {
		t.Fatalf("Failed to create main commit: %v", err)
	}

	if err := exec.Command("git", "checkout", "-b", "feature/test").Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	baseBranch := determineBaseBranch(nil)

	if baseBranch != "main" {
		t.Errorf("determineBaseBranch() = %q, want 'main'", baseBranch)
	}
}

// TestDetermineBaseBranchWithDevelop tests branch detection with develop branch
func TestDetermineBaseBranchWithDevelop(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	if err := createEmptyCommit("develop"); err != nil {
		t.Fatalf("Failed to create develop commit: %v", err)
	}

	if err := exec.Command("git", "checkout", "-b", "feature/test").Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	baseBranch := determineBaseBranch(nil)

	if baseBranch != "develop" {
		t.Errorf("determineBaseBranch() = %q, want 'develop'", baseBranch)
	}
}

// TestFindParentBranchFromMain tests parent branch detection when created from main
func TestFindParentBranchFromMain(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	if err := createEmptyCommit("main"); err != nil {
		t.Fatalf("Failed to create main commit: %v", err)
	}

	if err := exec.Command("git", "checkout", "-b", "feature/test").Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	if err := createEmptyCommit("feature/test"); err != nil {
		t.Fatalf("Failed to create feature commit: %v", err)
	}

	parentBranch := findParentBranch()

	if parentBranch != "main" {
		t.Errorf("findParentBranch() = %q, want 'main'", parentBranch)
	}
}

// TestFindParentBranchFromMaster tests parent branch detection when created from master
func TestFindParentBranchFromMaster(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	if err := createEmptyCommit("master"); err != nil {
		t.Fatalf("Failed to create master commit: %v", err)
	}

	exec.Command("git", "checkout", "master").Run()

	if err := exec.Command("git", "checkout", "-b", "feature/test").Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	if err := createEmptyCommit("feature/test"); err != nil {
		t.Fatalf("Failed to create feature commit: %v", err)
	}

	parentBranch := findParentBranch()

	if parentBranch != "master" {
		t.Errorf("findParentBranch() = %q, want 'master'", parentBranch)
	}
}

// TestFindParentBranchFromNonStandardBranch tests parent detection from a non-standard branch
func TestFindParentBranchFromNonStandardBranch(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	if err := createEmptyCommit("develop"); err != nil {
		t.Fatalf("Failed to create develop commit: %v", err)
	}

	if err := exec.Command("git", "checkout", "-b", "feature/from-develop").Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	parentBranch := findParentBranch()

	if parentBranch != "develop" {
		t.Errorf("findParentBranch() = %q, want 'develop'", parentBranch)
	}
}

// Helper function to create an empty commit
func createEmptyCommit(branchName string) error {
	if err := exec.Command("git", "checkout", "-b", branchName).Run(); err != nil {
		if err := exec.Command("git", "checkout", branchName).Run(); err != nil {
			return err
		}
	}

	cleanBranchName := strings.ReplaceAll(branchName, "/", "-")
	fileName := filepath.Join(".", "test_"+cleanBranchName+".txt")
	if err := os.WriteFile(fileName, []byte("test"), 0644); err != nil {
		return err
	}

	if err := exec.Command("git", "add", fileName).Run(); err != nil {
		return err
	}

	cmd := exec.Command("git", "commit", "-m", "Initial commit on "+branchName)
	return cmd.Run()
}
