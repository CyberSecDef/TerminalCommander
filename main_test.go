package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	
	// Create source file
	content := []byte("test content")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Test copy operation
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	
	// Verify destination file exists
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Fatal("Destination file was not created")
	}
	
	// Verify content
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	
	if string(dstContent) != string(content) {
		t.Fatalf("Content mismatch: got %q, want %q", dstContent, content)
	}
}

func TestCopyDir(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	
	srcDir := filepath.Join(tmpDir, "source")
	dstDir := filepath.Join(tmpDir, "dest")
	
	// Create source directory with files
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	
	testFile := filepath.Join(srcDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	subFile := filepath.Join(srcDir, "subdir", "sub.txt")
	if err := os.WriteFile(subFile, []byte("subtest"), 0644); err != nil {
		t.Fatalf("Failed to create sub file: %v", err)
	}
	
	// Test directory copy
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}
	
	// Verify destination directory exists
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Fatal("Destination directory was not created")
	}
	
	// Verify files were copied
	dstFile := filepath.Join(dstDir, "test.txt")
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Fatal("File in destination directory was not created")
	}
	
	dstSubFile := filepath.Join(dstDir, "subdir", "sub.txt")
	if _, err := os.Stat(dstSubFile); os.IsNotExist(err) {
		t.Fatal("File in subdirectory was not created")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
	}
	
	for _, tt := range tests {
		got := formatSize(tt.size)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestPane_RefreshPane(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	
	// Create test files and directories
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "dir1"), 0755)
	
	// Create a minimal commander instance
	pane := &Pane{
		CurrentPath: tmpDir,
	}
	
	// Create a minimal Commander just for testing refreshPane
	cmd := &Commander{
		leftPane:  pane,
		rightPane: &Pane{},
	}
	
	// Test refresh
	if err := cmd.refreshPane(pane); err != nil {
		t.Fatalf("refreshPane failed: %v", err)
	}
	
	// Verify files were loaded (should have .., dir1, file1.txt, file2.txt)
	if len(pane.Files) != 4 {
		t.Errorf("Expected 4 items (including ..), got %d", len(pane.Files))
	}
	
	// Verify parent directory is first
	if pane.Files[0].Name != ".." {
		t.Errorf("First item should be '..', got %q", pane.Files[0].Name)
	}
	
	// Verify directory comes before files (after ..)
	if !pane.Files[1].IsDir {
		t.Errorf("Second item should be a directory")
	}
}
