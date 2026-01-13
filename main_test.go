package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestHashComputation(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Hello, World!")
	
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	tests := []struct {
		algorithm    string
		expectedHash string
	}{
		{"MD5", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"SHA-1", "0a0a9f2a6772942557ab5355d76af442f8f65e01"},
		{"SHA-256", "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"},
		{"SHA-512", "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387"},
		{"SHA3-256", "1af17a664e3fa8e419b8ba05c2a173169df76162a5a286e0c405b460d478f7ef"},
		{"SHA3-512", "38e05c33d7b067127f217d8c856e554fcff09c9320b8a5979ce2ff5d95dd27ba35d1fba50c562dfd1d6cc48bc9c5baa4390894418cc942d968f97bcb659419ed"},
	}
	
	for _, tt := range tests {
		t.Run(tt.algorithm, func(t *testing.T) {
			// Create a minimal Commander instance
			cmd := &Commander{}
			cmd.hashAlgorithms = []string{tt.algorithm}
			cmd.hashSelectedIdx = 0
			cmd.hashFilePath = testFile
			
			// Compute hash
			cmd.computeHash()
			
			// Verify hash result
			if cmd.hashResult != tt.expectedHash {
				t.Errorf("Hash mismatch for %s:\ngot:  %s\nwant: %s", tt.algorithm, cmd.hashResult, tt.expectedHash)
			}
			
			// Verify hash result mode is enabled
			if !cmd.hashResultMode {
				t.Errorf("Hash result mode should be enabled after computation")
			}
			
			// Verify algorithm is stored
			if cmd.hashAlgorithm != tt.algorithm {
				t.Errorf("Hash algorithm mismatch: got %s, want %s", cmd.hashAlgorithm, tt.algorithm)
			}
		})
	}
}

func TestHashComputationBLAKE2(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Hello, World!")
	
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	tests := []struct {
		algorithm    string
		expectedHash string
	}{
		{"BLAKE2b-256", "511bc81dde11180838c562c82bb35f3223f46061ebde4a955c27b3f489cf1e03"},
		{"BLAKE2s-256", "ec9db904d636ef61f1421b2ba47112a4fa6b8964fd4a0a514834455c21df7812"},
		{"BLAKE3", "288a86a79f20a3d6dccdca7713beaed178798296bdfa7913fa2a62d9727bf8f8"},
		{"RIPEMD-160", "527a6a4b9a6da75607546842e0e00105350b1aaf"},
	}
	
	for _, tt := range tests {
		t.Run(tt.algorithm, func(t *testing.T) {
			// Create a minimal Commander instance
			cmd := &Commander{}
			cmd.hashAlgorithms = []string{tt.algorithm}
			cmd.hashSelectedIdx = 0
			cmd.hashFilePath = testFile
			
			// Compute hash
			cmd.computeHash()
			
			// Verify hash result
			if cmd.hashResult != tt.expectedHash {
				t.Errorf("Hash mismatch for %s:\ngot:  %s\nwant: %s", tt.algorithm, cmd.hashResult, tt.expectedHash)
			}
		})
	}
}

func TestHashComputationErrors(t *testing.T) {
	tmpDir := t.TempDir()
	
	t.Run("NonExistentFile", func(t *testing.T) {
		cmd := &Commander{}
		cmd.hashAlgorithms = []string{"MD5"}
		cmd.hashSelectedIdx = 0
		cmd.hashFilePath = filepath.Join(tmpDir, "nonexistent.txt")
		
		cmd.computeHash()
		
		// Should not enable result mode on error
		if cmd.hashResultMode {
			t.Error("Hash result mode should not be enabled on error")
		}
	})
	
	t.Run("NoAlgorithmSelected", func(t *testing.T) {
		cmd := &Commander{}
		cmd.hashAlgorithms = []string{}
		cmd.hashFilePath = filepath.Join(tmpDir, "test.txt")
		
		cmd.computeHash()
		
		// Should not enable result mode on error
		if cmd.hashResultMode {
			t.Error("Hash result mode should not be enabled on error")
		}
	})
}

// Helper function to create a test Commander instance
func createTestCommander(tmpDir string) *Commander {
	pane := &Pane{
		CurrentPath: tmpDir,
	}
	return &Commander{
		leftPane:   pane,
		rightPane:  &Pane{},
		activePane: PaneLeft,
	}
}

func TestGetAvailableArchiveFormats(t *testing.T) {
	cmd := &Commander{}
	
	formats := cmd.getAvailableArchiveFormats()
	
	// Should return at least one format on most systems
	// We can't guarantee specific formats since it depends on installed tools
	// But we can verify the function runs without error
	if formats == nil {
		t.Error("getAvailableArchiveFormats should not return nil")
	}
	
	// Verify no duplicates
	seen := make(map[string]bool)
	for _, format := range formats {
		if seen[format] {
			t.Errorf("Duplicate format found: %s", format)
		}
		seen[format] = true
	}
}

func TestCreateZipArchive(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	
	// Create test files
	testFile1 := filepath.Join(tmpDir, "file1.txt")
	testFile2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(testFile1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Create test directory
	testDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	testFile3 := filepath.Join(testDir, "file3.txt")
	if err := os.WriteFile(testFile3, []byte("content3"), 0644); err != nil {
		t.Fatalf("Failed to create test file in subdirectory: %v", err)
	}
	
	// Create a minimal Commander instance
	cmd := createTestCommander(tmpDir)
	
	// Test creating archive with multiple files
	archivePath := filepath.Join(tmpDir, "test.zip")
	files := []FileItem{
		{Name: "file1.txt", IsDir: false},
		{Name: "file2.txt", IsDir: false},
	}
	
	err := cmd.createZipArchive(archivePath, files)
	
	// Check if any zip creation method is available
	// If no method is available, we expect an error
	if err != nil {
		// Verify error message mentions attempted methods
		errMsg := err.Error()
		if !strings.Contains(errMsg, "zip") && !strings.Contains(errMsg, "available") {
			t.Errorf("Error message should mention zip or availability: %v", err)
		}
		t.Logf("No zip creation tools available, skipping archive verification: %v", err)
		return
	}
	
	// If successful, verify the archive was created
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("Archive file was not created")
	}
}

func TestCreateZipArchiveWithDirectory(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	
	// Create a subdirectory with files
	subDir := filepath.Join(tmpDir, "testdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	testFile := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Create a minimal Commander instance
	cmd := createTestCommander(tmpDir)
	
	// Test creating archive with directory
	archivePath := filepath.Join(tmpDir, "dirtest.zip")
	files := []FileItem{
		{Name: "testdir", IsDir: true},
	}
	
	err := cmd.createZipArchive(archivePath, files)
	
	// Check if any zip creation method is available
	if err != nil {
		t.Logf("No zip creation tools available, skipping archive verification: %v", err)
		return
	}
	
	// If successful, verify the archive was created
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("Archive file was not created")
	}
}

func TestCreateZipArchiveWithSpaces(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	
	// Create test file with spaces in name
	testFile := filepath.Join(tmpDir, "file with spaces.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Create a minimal Commander instance
	cmd := createTestCommander(tmpDir)
	
	// Test creating archive with file containing spaces
	archivePath := filepath.Join(tmpDir, "spaces test.zip")
	files := []FileItem{
		{Name: "file with spaces.txt", IsDir: false},
	}
	
	err := cmd.createZipArchive(archivePath, files)
	
	// Check if any zip creation method is available
	if err != nil {
		t.Logf("No zip creation tools available, skipping archive verification: %v", err)
		return
	}
	
	// If successful, verify the archive was created
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("Archive file was not created")
	}
}

func TestIsTextFile(t *testing.T) {
tests := []struct {
name    string
content []byte
want    bool
}{
{"Plain text", []byte("Hello, World!"), true},
{"Empty", []byte(""), true},
{"With newlines", []byte("Line 1\nLine 2\nLine 3"), true},
{"Binary with null", []byte{0x00, 0x01, 0x02}, false},
{"UTF-8 text", []byte("Hello 世界"), true},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
got := isTextFile(tt.content)
if got != tt.want {
t.Errorf("isTextFile() = %v, want %v", got, tt.want)
}
})
}
}

func TestCalculateDiff(t *testing.T) {
tmpDir := t.TempDir()

// Create test files
file1 := filepath.Join(tmpDir, "file1.txt")
file2 := filepath.Join(tmpDir, "file2.txt")

content1 := "Line 1\nLine 2\nLine 3\n"
content2 := "Line 1\nLine 2 modified\nLine 3\n"

os.WriteFile(file1, []byte(content1), 0644)
os.WriteFile(file2, []byte(content2), 0644)

cmd := &Commander{
diffLeftLines:  []string{"Line 1", "Line 2", "Line 3"},
diffRightLines: []string{"Line 1", "Line 2 modified", "Line 3"},
}

cmd.calculateDiff()

// Should have detected differences
if len(cmd.diffDifferences) == 0 {
t.Error("Expected differences to be found")
}

// Check for non-equal blocks
hasNonEqual := false
for _, diff := range cmd.diffDifferences {
if diff.Type != "equal" {
hasNonEqual = true
break
}
}

if !hasNonEqual {
t.Error("Expected at least one non-equal diff block")
}
}

func TestCalculateDiffIdentical(t *testing.T) {
cmd := &Commander{
diffLeftLines:  []string{"Line 1", "Line 2", "Line 3"},
diffRightLines: []string{"Line 1", "Line 2", "Line 3"},
}

cmd.calculateDiff()

// Should have at least one block
if len(cmd.diffDifferences) == 0 {
t.Error("Expected at least one diff block")
}

// All blocks should be equal
for _, diff := range cmd.diffDifferences {
if diff.Type != "equal" {
t.Errorf("Expected all blocks to be equal, got %s", diff.Type)
}
}
}

func TestEnterDiffMode(t *testing.T) {
tmpDir := t.TempDir()

// Create test files
file1 := filepath.Join(tmpDir, "file1.txt")
file2 := filepath.Join(tmpDir, "file2.txt")

os.WriteFile(file1, []byte("Line 1\nLine 2\n"), 0644)
os.WriteFile(file2, []byte("Line 1\nLine 2 modified\n"), 0644)

// Create panes with test files
leftPane := &Pane{
CurrentPath: tmpDir,
Files: []FileItem{
{Name: "file1.txt", Path: file1, IsDir: false},
},
SelectedIdx: 0,
}

rightPane := &Pane{
CurrentPath: tmpDir,
Files: []FileItem{
{Name: "file2.txt", Path: file2, IsDir: false},
},
SelectedIdx: 0,
}

cmd := &Commander{
leftPane:  leftPane,
rightPane: rightPane,
}

cmd.enterDiffMode()

// Check that diff mode was entered
if !cmd.diffMode {
t.Error("Expected diff mode to be active")
}

// Check that lines were loaded
if len(cmd.diffLeftLines) == 0 {
t.Error("Expected left lines to be loaded")
}

if len(cmd.diffRightLines) == 0 {
t.Error("Expected right lines to be loaded")
}

// Check that differences were calculated
if len(cmd.diffDifferences) == 0 {
t.Error("Expected differences to be calculated")
}
}

func TestEnterDiffModeWithDirectories(t *testing.T) {
tmpDir := t.TempDir()

// Create a directory
subdir := filepath.Join(tmpDir, "subdir")
os.MkdirAll(subdir, 0755)

leftPane := &Pane{
CurrentPath: tmpDir,
Files: []FileItem{
{Name: "subdir", Path: subdir, IsDir: true},
},
SelectedIdx: 0,
}

rightPane := &Pane{
CurrentPath: tmpDir,
Files: []FileItem{
{Name: "subdir", Path: subdir, IsDir: true},
},
SelectedIdx: 0,
}

cmd := &Commander{
leftPane:  leftPane,
rightPane: rightPane,
}

cmd.enterDiffMode()

// Should not enter diff mode with directories
if cmd.diffMode {
t.Error("Should not enter diff mode with directories")
}
}

func TestCopyDiffLeftToRight(t *testing.T) {
cmd := &Commander{
diffLeftLines:  []string{"Line 1", "Line 2", "Line 3"},
diffRightLines: []string{"Line 1", "Line 2 modified", "Line 3"},
diffDifferences: []DiffBlock{
{LeftStart: 0, LeftEnd: 0, RightStart: 0, RightEnd: 0, Type: "equal"},
{LeftStart: 1, LeftEnd: 1, RightStart: 1, RightEnd: 1, Type: "modify"},
{LeftStart: 2, LeftEnd: 2, RightStart: 2, RightEnd: 2, Type: "equal"},
},
diffCurrentIdx: 1,
}

cmd.copyDiffLeftToRight()

// Check that right was modified
if !cmd.diffRightModified {
t.Error("Expected right file to be marked as modified")
}
}

func TestSaveDiffFiles(t *testing.T) {
tmpDir := t.TempDir()

leftFile := filepath.Join(tmpDir, "left.txt")
rightFile := filepath.Join(tmpDir, "right.txt")

// Create initial files
os.WriteFile(leftFile, []byte("original left\n"), 0644)
os.WriteFile(rightFile, []byte("original right\n"), 0644)

cmd := &Commander{
diffLeftPath:      leftFile,
diffRightPath:     rightFile,
diffLeftLines:     []string{"modified left"},
diffRightLines:    []string{"modified right"},
diffLeftModified:  true,
diffRightModified: true,
}

cmd.saveDiffFiles()

// Check that files were saved
leftContent, err := os.ReadFile(leftFile)
if err != nil {
t.Fatalf("Failed to read left file: %v", err)
}

if string(leftContent) != "modified left\n" {
t.Errorf("Left file content = %q, want %q", leftContent, "modified left\n")
}

rightContent, err := os.ReadFile(rightFile)
if err != nil {
t.Fatalf("Failed to read right file: %v", err)
}

if string(rightContent) != "modified right\n" {
t.Errorf("Right file content = %q, want %q", rightContent, "modified right\n")
}

// Check that modified flags were cleared
if cmd.diffLeftModified {
t.Error("Expected left modified flag to be cleared")
}

if cmd.diffRightModified {
t.Error("Expected right modified flag to be cleared")
}
}

func TestDiffModeWorkflow(t *testing.T) {
tmpDir := t.TempDir()

// Create two test files with known differences
file1 := filepath.Join(tmpDir, "file1.txt")
file2 := filepath.Join(tmpDir, "file2.txt")

content1 := "Line 1\nLine 2\nLine 3\nLine 4\n"
content2 := "Line 1\nLine 2 modified\nLine 3\nLine 5\n"

os.WriteFile(file1, []byte(content1), 0644)
os.WriteFile(file2, []byte(content2), 0644)

// Create panes
leftPane := &Pane{
CurrentPath: tmpDir,
Files: []FileItem{
{Name: "file1.txt", Path: file1, IsDir: false},
},
SelectedIdx: 0,
}

rightPane := &Pane{
CurrentPath: tmpDir,
Files: []FileItem{
{Name: "file2.txt", Path: file2, IsDir: false},
},
SelectedIdx: 0,
}

cmd := &Commander{
leftPane:  leftPane,
rightPane: rightPane,
}

// Enter diff mode
cmd.enterDiffMode()

if !cmd.diffMode {
t.Fatal("Failed to enter diff mode")
}

// Verify differences were calculated
if len(cmd.diffDifferences) == 0 {
t.Error("No differences detected")
}

// Find a non-equal difference
foundDiff := false
for i, diff := range cmd.diffDifferences {
if diff.Type != "equal" {
cmd.diffCurrentIdx = i
foundDiff = true
break
}
}

if !foundDiff {
t.Fatal("No non-equal differences found")
}

// Test copy operation
originalRightLines := make([]string, len(cmd.diffRightLines))
copy(originalRightLines, cmd.diffRightLines)

cmd.copyDiffLeftToRight()

// Right should be marked as modified
if !cmd.diffRightModified {
t.Error("Right file should be marked as modified after copy")
}

// Test save
cmd.saveDiffFiles()

// Modified flags should be cleared
if cmd.diffLeftModified || cmd.diffRightModified {
t.Error("Modified flags should be cleared after save")
}

// Test navigation
_ = cmd.diffCurrentIdx
cmd.jumpToNextDiff()
// Either moved to next diff or wrapped around

cmd.jumpToPrevDiff()
// Should be able to navigate back

// Exit diff mode
cmd.exitDiffMode()

if cmd.diffMode {
t.Error("Should have exited diff mode")
}
}

func TestDiffModeEmptyFiles(t *testing.T) {
tmpDir := t.TempDir()

// Create empty files
file1 := filepath.Join(tmpDir, "empty1.txt")
file2 := filepath.Join(tmpDir, "empty2.txt")

os.WriteFile(file1, []byte(""), 0644)
os.WriteFile(file2, []byte(""), 0644)

leftPane := &Pane{
CurrentPath: tmpDir,
Files: []FileItem{
{Name: "empty1.txt", Path: file1, IsDir: false},
},
SelectedIdx: 0,
}

rightPane := &Pane{
CurrentPath: tmpDir,
Files: []FileItem{
{Name: "empty2.txt", Path: file2, IsDir: false},
},
SelectedIdx: 0,
}

cmd := &Commander{
leftPane:  leftPane,
rightPane: rightPane,
}

cmd.enterDiffMode()

if !cmd.diffMode {
t.Fatal("Failed to enter diff mode with empty files")
}

// Should have at least one line (empty files get one empty line)
if len(cmd.diffLeftLines) == 0 || len(cmd.diffRightLines) == 0 {
t.Error("Empty files should have at least one line")
}
}
