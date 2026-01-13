package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/zeebo/blake3"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

const (
	PaneLeft = iota
	PaneRight
)

type FileItem struct {
	Name    string
	Ext     string
	IsDir   bool
	Size    int64
	ModTime time.Time
	Path    string
}

type Pane struct {
	CurrentPath  string
	Files        []FileItem
	SelectedIdx  int
	ScrollOffset int
	Width        int
	Height       int
}

type SearchResult struct {
	Name    string
	Path    string
	Dir     string
	IsDir   bool
	RelPath string
}

type Commander struct {
	screen        tcell.Screen
	leftPane      *Pane
	rightPane     *Pane
	activePane    int
	statusMsg     string
	statusMsgTime time.Time
	searchMode    bool
	searchQuery   string
	inputMode     string // "rename", "newdir", or ""
	inputBuffer   string
	inputPrompt   string
	// Editor state
	editorMode     bool
	editorLines    []string
	editorCursorX  int
	editorCursorY  int
	editorScrollY  int
	editorScrollX  int
	editorFilePath string
	editorModified bool
	// Search results state
	searchResultsMode  bool
	searchResults      []SearchResult
	searchResultIdx    int
	searchResultScroll int
	searchBaseDir      string
	// Hash selection state
	hashSelectionMode bool
	hashAlgorithms    []string
	hashSelectedIdx   int
	hashFilePath      string
	// Hash result state
	hashResultMode     bool
	hashResult         string
	hashAlgorithm      string
	hashResultFilePath string
}

func NewCommander() (*Commander, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}

	screen.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorBlack))
	screen.Clear()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	cmd := &Commander{
		screen:     screen,
		activePane: PaneLeft,
		leftPane: &Pane{
			CurrentPath: cwd,
		},
		rightPane: &Pane{
			CurrentPath: cwd,
		},
	}

	return cmd, nil
}

func (c *Commander) setStatus(msg string) {
	c.statusMsg = msg
	c.statusMsgTime = time.Now()
}

func (c *Commander) Run() error {
	defer c.screen.Fini()

	if err := c.refreshPane(c.leftPane); err != nil {
		return err
	}
	if err := c.refreshPane(c.rightPane); err != nil {
		return err
	}

	c.updateLayout()
	c.draw()

	for {
		ev := c.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			c.screen.Sync()
			c.updateLayout()
			c.draw()
		case *tcell.EventKey:
			if c.handleKeyEvent(ev) {
				return nil
			}
			c.draw()
		}
	}
}

func (c *Commander) handleKeyEvent(ev *tcell.EventKey) bool {
	if c.editorMode {
		return c.handleEditorKey(ev)
	}

	if c.searchResultsMode {
		return c.handleSearchResultsKey(ev)
	}

	if c.hashSelectionMode {
		return c.handleHashSelectionKey(ev)
	}

	if c.hashResultMode {
		return c.handleHashResultKey(ev)
	}

	if c.inputMode != "" {
		return c.handleInputKey(ev)
	}

	if c.searchMode {
		return c.handleSearchKey(ev)
	}

	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlQ:
		return true
	case tcell.KeyTab:
		if c.activePane == PaneLeft {
			c.activePane = PaneRight
		} else {
			c.activePane = PaneLeft
		}
	case tcell.KeyUp:
		c.moveSelection(-1)
	case tcell.KeyDown:
		c.moveSelection(1)
	case tcell.KeyEnter:
		c.enterDirectory()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		c.goToParent()
	case tcell.KeyCtrlF:
		c.startSearch()
	case tcell.KeyCtrlC:
		c.copyFile()
	case tcell.KeyCtrlX:
		c.moveFile()
	case tcell.KeyCtrlD, tcell.KeyDelete:
		c.deleteFile()
	case tcell.KeyCtrlR:
		c.renameFile()
	case tcell.KeyCtrlE:
		c.editFile()
	case tcell.KeyF5:
		c.copyFile()
	case tcell.KeyF6:
		c.moveFile()
	case tcell.KeyF8:
		c.deleteFile()
	case tcell.KeyCtrlN:
		c.createDirectory()
	case tcell.KeyCtrlG:
		c.gotoFolder()
	case tcell.KeyCtrlH:
		c.startHashSelection()
	}

	return false
}

func (c *Commander) handleSearchKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		c.searchMode = false
		c.searchQuery = ""
		c.setStatus("")
		return false
	case tcell.KeyEnter:
		c.performSearch()
		c.searchMode = false
		return false
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(c.searchQuery) > 0 {
			c.searchQuery = c.searchQuery[:len(c.searchQuery)-1]
		}
	case tcell.KeyRune:
		c.searchQuery += string(ev.Rune())
	}
	c.setStatus("Search: " + c.searchQuery)
	return false
}

func (c *Commander) handleInputKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		c.inputMode = ""
		c.inputBuffer = ""
		c.inputPrompt = ""
		c.setStatus("Cancelled")
		return false
	case tcell.KeyEnter:
		c.processInput()
		return false
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(c.inputBuffer) > 0 {
			c.inputBuffer = c.inputBuffer[:len(c.inputBuffer)-1]
		}
	case tcell.KeyRune:
		c.inputBuffer += string(ev.Rune())
	}
	c.setStatus(c.inputPrompt + c.inputBuffer)
	return false
}

func (c *Commander) processInput() {
	pane := c.getActivePane()

	switch c.inputMode {
	case "rename":
		if len(c.inputBuffer) == 0 {
			c.setStatus("Name cannot be empty")
			c.inputMode = ""
			c.inputBuffer = ""
			return
		}

		if len(pane.Files) == 0 {
			c.setStatus("No file selected")
			c.inputMode = ""
			c.inputBuffer = ""
			return
		}

		selected := pane.Files[pane.SelectedIdx]
		if selected.Name == ".." {
			c.setStatus("Cannot rename parent directory link")
			c.inputMode = ""
			c.inputBuffer = ""
			return
		}

		newPath := filepath.Join(filepath.Dir(selected.Path), c.inputBuffer)
		err := os.Rename(selected.Path, newPath)
		if err != nil {
			c.setStatus("Error renaming: " + err.Error())
		} else {
			c.setStatus("Renamed to: " + c.inputBuffer)
			c.refreshPane(pane)
		}

	case "newdir":
		if len(c.inputBuffer) == 0 {
			c.setStatus("Directory name cannot be empty")
			c.inputMode = ""
			c.inputBuffer = ""
			return
		}

		newPath := filepath.Join(pane.CurrentPath, c.inputBuffer)
		err := os.MkdirAll(newPath, 0755)
		if err != nil {
			c.setStatus("Error creating directory: " + err.Error())
		} else {
			c.setStatus("Created directory: " + c.inputBuffer)
			c.refreshPane(pane)
		}

	case "goto":
		if len(c.inputBuffer) == 0 {
			c.setStatus("Path cannot be empty")
			c.inputMode = ""
			c.inputBuffer = ""
			return
		}

		// Expand home directory
		path := c.inputBuffer
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				path = filepath.Join(home, path[2:])
			}
		} else if path == "~" {
			home, err := os.UserHomeDir()
			if err == nil {
				path = home
			}
		}

		// Make absolute if relative
		if !filepath.IsAbs(path) {
			path = filepath.Join(pane.CurrentPath, path)
		}

		// Clean the path
		path = filepath.Clean(path)

		// Check if directory exists
		info, err := os.Stat(path)
		if err != nil {
			c.setStatus("Error: " + err.Error())
		} else if !info.IsDir() {
			c.setStatus("Error: Not a directory")
		} else {
			pane.CurrentPath = path
			pane.SelectedIdx = 0
			pane.ScrollOffset = 0
			c.refreshPane(pane)
			c.setStatus("Navigated to: " + path)
		}
	}

	c.inputMode = ""
	c.inputBuffer = ""
	c.inputPrompt = ""
}

func (c *Commander) getActivePane() *Pane {
	if c.activePane == PaneLeft {
		return c.leftPane
	}
	return c.rightPane
}

func (c *Commander) getInactivePane() *Pane {
	if c.activePane == PaneLeft {
		return c.rightPane
	}
	return c.leftPane
}

func (c *Commander) moveSelection(delta int) {
	pane := c.getActivePane()
	if len(pane.Files) == 0 {
		return
	}

	pane.SelectedIdx += delta
	if pane.SelectedIdx < 0 {
		pane.SelectedIdx = 0
	}
	if pane.SelectedIdx >= len(pane.Files) {
		pane.SelectedIdx = len(pane.Files) - 1
	}

	// Adjust scroll offset
	if pane.SelectedIdx < pane.ScrollOffset {
		pane.ScrollOffset = pane.SelectedIdx
	}
	if pane.SelectedIdx >= pane.ScrollOffset+pane.Height-4 {
		pane.ScrollOffset = pane.SelectedIdx - pane.Height + 5
	}
}

func (c *Commander) enterDirectory() {
	pane := c.getActivePane()
	if len(pane.Files) == 0 {
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.IsDir {
		pane.CurrentPath = selected.Path
		pane.SelectedIdx = 0
		pane.ScrollOffset = 0
		c.refreshPane(pane)
		c.setStatus("Entered: " + selected.Name)
	} else {
		c.setStatus("Use Ctrl+E to edit file")
	}
}

func (c *Commander) goToParent() {
	pane := c.getActivePane()
	parent := filepath.Dir(pane.CurrentPath)
	if parent != pane.CurrentPath {
		pane.CurrentPath = parent
		pane.SelectedIdx = 0
		pane.ScrollOffset = 0
		c.refreshPane(pane)
		c.setStatus("Parent directory")
	}
}

func (c *Commander) startSearch() {
	c.searchMode = true
	c.searchQuery = ""
	c.setStatus("Search: ")
}

func (c *Commander) performSearch() {
	pane := c.getActivePane()
	query := strings.ToLower(c.searchQuery)

	if query == "" {
		c.setStatus("Search cancelled")
		c.searchQuery = ""
		return
	}

	c.setStatus("Searching...")
	c.draw()

	// Perform recursive search
	var results []SearchResult
	baseDir := pane.CurrentPath

	filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip directories we can't access
		}

		name := d.Name()
		if strings.Contains(strings.ToLower(name), query) {
			relPath, _ := filepath.Rel(baseDir, path)
			results = append(results, SearchResult{
				Name:    name,
				Path:    path,
				Dir:     filepath.Dir(path),
				IsDir:   d.IsDir(),
				RelPath: relPath,
			})
		}

		// Limit results to prevent UI slowdown
		if len(results) >= 500 {
			return filepath.SkipAll
		}
		return nil
	})

	if len(results) == 0 {
		c.setStatus("No matches found for: " + c.searchQuery)
		c.searchQuery = ""
		return
	}

	// Show search results
	c.searchResults = results
	c.searchResultIdx = 0
	c.searchResultScroll = 0
	c.searchBaseDir = baseDir
	c.searchResultsMode = true
	c.setStatus(fmt.Sprintf("Found %d matches. Enter:Go to folder, Esc:Cancel", len(results)))
	c.searchQuery = ""
}

func (c *Commander) handleSearchResultsKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		c.searchResultsMode = false
		c.searchResults = nil
		c.setStatus("Search cancelled")
		return false
	case tcell.KeyEnter:
		if len(c.searchResults) > 0 {
			result := c.searchResults[c.searchResultIdx]
			pane := c.getActivePane()
			pane.CurrentPath = result.Dir
			pane.SelectedIdx = 0
			pane.ScrollOffset = 0
			c.refreshPane(pane)

			// Try to select the found file
			for i, f := range pane.Files {
				if f.Name == result.Name {
					pane.SelectedIdx = i
					if pane.SelectedIdx >= pane.Height-4 {
						pane.ScrollOffset = pane.SelectedIdx - pane.Height + 5
					}
					break
				}
			}

			c.setStatus("Navigated to: " + result.Dir)
		}
		c.searchResultsMode = false
		c.searchResults = nil
		return false
	case tcell.KeyUp:
		if c.searchResultIdx > 0 {
			c.searchResultIdx--
		}
	case tcell.KeyDown:
		if c.searchResultIdx < len(c.searchResults)-1 {
			c.searchResultIdx++
		}
	case tcell.KeyPgUp:
		_, height := c.screen.Size()
		pageSize := height - 4
		c.searchResultIdx -= pageSize
		if c.searchResultIdx < 0 {
			c.searchResultIdx = 0
		}
	case tcell.KeyPgDn:
		_, height := c.screen.Size()
		pageSize := height - 4
		c.searchResultIdx += pageSize
		if c.searchResultIdx >= len(c.searchResults) {
			c.searchResultIdx = len(c.searchResults) - 1
		}
	case tcell.KeyHome:
		c.searchResultIdx = 0
	case tcell.KeyEnd:
		c.searchResultIdx = len(c.searchResults) - 1
	}

	// Adjust scroll
	_, height := c.screen.Size()
	visibleHeight := height - 4
	if c.searchResultIdx < c.searchResultScroll {
		c.searchResultScroll = c.searchResultIdx
	}
	if c.searchResultIdx >= c.searchResultScroll+visibleHeight {
		c.searchResultScroll = c.searchResultIdx - visibleHeight + 1
	}

	return false
}

func (c *Commander) startHashSelection() {
	pane := c.getActivePane()

	if len(pane.Files) == 0 {
		c.setStatus("No file selected")
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.setStatus("Cannot hash parent directory link")
		return
	}

	if selected.IsDir {
		c.setStatus("Cannot hash a directory")
		return
	}

	// Initialize hash algorithm list
	c.hashAlgorithms = []string{
		"MD5",
		"SHA-1",
		"SHA-256",
		"SHA-512",
		"SHA3-256",
		"SHA3-512",
		"BLAKE2b-256",
		"BLAKE2s-256",
		"BLAKE3",
		"RIPEMD-160",
	}
	c.hashSelectedIdx = 0
	c.hashFilePath = selected.Path
	c.hashSelectionMode = true
	c.setStatus("Select hash algorithm. Enter:Compute, Esc:Cancel")
}

func (c *Commander) handleHashSelectionKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		c.hashSelectionMode = false
		c.hashAlgorithms = nil
		c.hashFilePath = ""
		c.setStatus("Hash cancelled")
		return false
	case tcell.KeyEnter:
		if len(c.hashAlgorithms) > 0 {
			c.computeHash()
		}
		c.hashSelectionMode = false
		return false
	case tcell.KeyUp:
		if c.hashSelectedIdx > 0 {
			c.hashSelectedIdx--
		}
	case tcell.KeyDown:
		if c.hashSelectedIdx < len(c.hashAlgorithms)-1 {
			c.hashSelectedIdx++
		}
	case tcell.KeyHome:
		c.hashSelectedIdx = 0
	case tcell.KeyEnd:
		c.hashSelectedIdx = len(c.hashAlgorithms) - 1
	}
	return false
}

func (c *Commander) computeHash() {
	if c.hashFilePath == "" || len(c.hashAlgorithms) == 0 {
		c.setStatus("Error: No file or algorithm selected")
		return
	}

	algorithm := c.hashAlgorithms[c.hashSelectedIdx]
	c.setStatus("Computing " + algorithm + " hash...")
	if c.screen != nil {
		c.draw()
	}

	// Open file
	file, err := os.Open(c.hashFilePath)
	if err != nil {
		c.setStatus("Error opening file: " + err.Error())
		c.hashAlgorithms = nil
		c.hashFilePath = ""
		return
	}
	defer file.Close()

	// Get file info for progress indication
	fileInfo, err := file.Stat()
	if err != nil {
		c.setStatus("Error getting file info: " + err.Error())
		c.hashAlgorithms = nil
		c.hashFilePath = ""
		return
	}

	// Show file size in status for large files
	if fileInfo.Size() > 10*1024*1024 { // > 10MB
		c.setStatus(fmt.Sprintf("Computing %s hash for %s file...", algorithm, formatSize(fileInfo.Size())))
		if c.screen != nil {
			c.draw()
		}
	}

	var hashBytes []byte
	var hashErr error

	// Compute hash based on selected algorithm
	switch algorithm {
	case "MD5":
		hasher := md5.New()
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	case "SHA-1":
		hasher := sha1.New()
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	case "SHA-256":
		hasher := sha256.New()
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	case "SHA-512":
		hasher := sha512.New()
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	case "SHA3-256":
		hasher := sha3.New256()
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	case "SHA3-512":
		hasher := sha3.New512()
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	case "BLAKE2b-256":
		hasher, err := blake2b.New256(nil)
		if err != nil {
			c.setStatus("Error initializing BLAKE2b: " + err.Error())
			c.hashAlgorithms = nil
			c.hashFilePath = ""
			return
		}
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	case "BLAKE2s-256":
		hasher, err := blake2s.New256(nil)
		if err != nil {
			c.setStatus("Error initializing BLAKE2s: " + err.Error())
			c.hashAlgorithms = nil
			c.hashFilePath = ""
			return
		}
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	case "BLAKE3":
		hasher := blake3.New()
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	case "RIPEMD-160":
		hasher := ripemd160.New()
		_, hashErr = io.Copy(hasher, file)
		hashBytes = hasher.Sum(nil)
	default:
		c.setStatus("Error: Unknown algorithm")
		c.hashAlgorithms = nil
		c.hashFilePath = ""
		return
	}

	if hashErr != nil {
		c.setStatus("Error computing hash: " + hashErr.Error())
		c.hashAlgorithms = nil
		c.hashFilePath = ""
		return
	}

	// Convert to hex string (lowercase)
	c.hashResult = hex.EncodeToString(hashBytes)
	c.hashAlgorithm = algorithm
	c.hashResultFilePath = c.hashFilePath
	c.hashResultMode = true
	c.hashAlgorithms = nil
	c.hashFilePath = ""
	c.setStatus("Press any key to close | Hash: " + c.hashResult)
}

func (c *Commander) handleHashResultKey(ev *tcell.EventKey) bool {
	// Any key closes the hash result display
	c.hashResultMode = false
	c.hashResult = ""
	c.hashAlgorithm = ""
	c.hashResultFilePath = ""
	c.setStatus("")
	return false
}

func (c *Commander) copyFile() {
	pane := c.getActivePane()
	destPane := c.getInactivePane()

	if len(pane.Files) == 0 {
		c.setStatus("No file selected")
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.setStatus("Cannot copy parent directory link")
		return
	}

	destPath := filepath.Join(destPane.CurrentPath, selected.Name)

	err := copyFileOrDir(selected.Path, destPath)
	if err != nil {
		c.setStatus("Error copying: " + err.Error())
	} else {
		c.setStatus("Copied: " + selected.Name)
		c.refreshPane(destPane)
	}
}

func (c *Commander) moveFile() {
	pane := c.getActivePane()
	destPane := c.getInactivePane()

	if len(pane.Files) == 0 {
		c.setStatus("No file selected")
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.setStatus("Cannot move parent directory link")
		return
	}

	destPath := filepath.Join(destPane.CurrentPath, selected.Name)

	err := os.Rename(selected.Path, destPath)
	if err != nil {
		c.setStatus("Error moving: " + err.Error())
	} else {
		c.setStatus("Moved: " + selected.Name)
		c.refreshPane(pane)
		c.refreshPane(destPane)
	}
}

func (c *Commander) deleteFile() {
	pane := c.getActivePane()

	if len(pane.Files) == 0 {
		c.setStatus("No file selected")
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.setStatus("Cannot delete parent directory link")
		return
	}

	var err error
	if selected.IsDir {
		err = os.RemoveAll(selected.Path)
	} else {
		err = os.Remove(selected.Path)
	}

	if err != nil {
		c.setStatus("Error deleting: " + err.Error())
	} else {
		c.setStatus("Deleted: " + selected.Name)
		if pane.SelectedIdx > 0 {
			pane.SelectedIdx--
		}
		c.refreshPane(pane)
	}
}

func (c *Commander) renameFile() {
	pane := c.getActivePane()

	if len(pane.Files) == 0 {
		c.setStatus("No file selected")
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.setStatus("Cannot rename parent directory link")
		return
	}

	c.inputMode = "rename"
	c.inputBuffer = selected.Name
	c.inputPrompt = "Rename to: "
	c.setStatus(c.inputPrompt + c.inputBuffer)
}

func (c *Commander) editFile() {
	pane := c.getActivePane()

	if len(pane.Files) == 0 {
		c.setStatus("No file selected")
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.IsDir {
		c.setStatus("Cannot edit a directory")
		return
	}

	// Load file content
	content, err := os.ReadFile(selected.Path)
	if err != nil {
		c.setStatus("Error reading file: " + err.Error())
		return
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")
	// Remove trailing empty line if file ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		lines = []string{""}
	}

	c.editorMode = true
	c.editorLines = lines
	c.editorCursorX = 0
	c.editorCursorY = 0
	c.editorScrollY = 0
	c.editorScrollX = 0
	c.editorFilePath = selected.Path
	c.editorModified = false
	c.setStatus("Editing: " + selected.Name + " | Ctrl+S:Save Ctrl+Q:Quit")
}

func (c *Commander) handleEditorKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyCtrlQ, tcell.KeyEscape:
		if c.editorModified {
			c.setStatus("Unsaved changes! Press Ctrl+S to save or Ctrl+Q again to discard")
			c.editorModified = false // Allow second press to exit
			return false
		}
		c.exitEditor()
		return false
	case tcell.KeyCtrlS:
		c.saveEditorFile()
		return false
	case tcell.KeyUp:
		if c.editorCursorY > 0 {
			c.editorCursorY--
			if c.editorCursorX > len(c.editorLines[c.editorCursorY]) {
				c.editorCursorX = len(c.editorLines[c.editorCursorY])
			}
		}
	case tcell.KeyDown:
		if c.editorCursorY < len(c.editorLines)-1 {
			c.editorCursorY++
			if c.editorCursorX > len(c.editorLines[c.editorCursorY]) {
				c.editorCursorX = len(c.editorLines[c.editorCursorY])
			}
		}
	case tcell.KeyLeft:
		if c.editorCursorX > 0 {
			c.editorCursorX--
		} else if c.editorCursorY > 0 {
			c.editorCursorY--
			c.editorCursorX = len(c.editorLines[c.editorCursorY])
		}
	case tcell.KeyRight:
		if c.editorCursorX < len(c.editorLines[c.editorCursorY]) {
			c.editorCursorX++
		} else if c.editorCursorY < len(c.editorLines)-1 {
			c.editorCursorY++
			c.editorCursorX = 0
		}
	case tcell.KeyHome:
		c.editorCursorX = 0
	case tcell.KeyEnd:
		c.editorCursorX = len(c.editorLines[c.editorCursorY])
	case tcell.KeyPgUp:
		_, height := c.screen.Size()
		pageSize := height - 3
		c.editorCursorY -= pageSize
		if c.editorCursorY < 0 {
			c.editorCursorY = 0
		}
		if c.editorCursorX > len(c.editorLines[c.editorCursorY]) {
			c.editorCursorX = len(c.editorLines[c.editorCursorY])
		}
	case tcell.KeyPgDn:
		_, height := c.screen.Size()
		pageSize := height - 3
		c.editorCursorY += pageSize
		if c.editorCursorY >= len(c.editorLines) {
			c.editorCursorY = len(c.editorLines) - 1
		}
		if c.editorCursorX > len(c.editorLines[c.editorCursorY]) {
			c.editorCursorX = len(c.editorLines[c.editorCursorY])
		}
	case tcell.KeyEnter:
		// Split line at cursor
		line := c.editorLines[c.editorCursorY]
		leftPart := line[:c.editorCursorX]
		rightPart := line[c.editorCursorX:]
		c.editorLines[c.editorCursorY] = leftPart
		// Insert new line after current
		newLines := make([]string, len(c.editorLines)+1)
		copy(newLines, c.editorLines[:c.editorCursorY+1])
		newLines[c.editorCursorY+1] = rightPart
		copy(newLines[c.editorCursorY+2:], c.editorLines[c.editorCursorY+1:])
		c.editorLines = newLines
		c.editorCursorY++
		c.editorCursorX = 0
		c.editorModified = true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if c.editorCursorX > 0 {
			// Delete character before cursor
			line := c.editorLines[c.editorCursorY]
			c.editorLines[c.editorCursorY] = line[:c.editorCursorX-1] + line[c.editorCursorX:]
			c.editorCursorX--
			c.editorModified = true
		} else if c.editorCursorY > 0 {
			// Join with previous line
			prevLineLen := len(c.editorLines[c.editorCursorY-1])
			c.editorLines[c.editorCursorY-1] += c.editorLines[c.editorCursorY]
			// Remove current line
			c.editorLines = append(c.editorLines[:c.editorCursorY], c.editorLines[c.editorCursorY+1:]...)
			c.editorCursorY--
			c.editorCursorX = prevLineLen
			c.editorModified = true
		}
	case tcell.KeyDelete:
		line := c.editorLines[c.editorCursorY]
		if c.editorCursorX < len(line) {
			// Delete character at cursor
			c.editorLines[c.editorCursorY] = line[:c.editorCursorX] + line[c.editorCursorX+1:]
			c.editorModified = true
		} else if c.editorCursorY < len(c.editorLines)-1 {
			// Join with next line
			c.editorLines[c.editorCursorY] += c.editorLines[c.editorCursorY+1]
			c.editorLines = append(c.editorLines[:c.editorCursorY+1], c.editorLines[c.editorCursorY+2:]...)
			c.editorModified = true
		}
	case tcell.KeyTab:
		// Insert tab as spaces
		line := c.editorLines[c.editorCursorY]
		c.editorLines[c.editorCursorY] = line[:c.editorCursorX] + "    " + line[c.editorCursorX:]
		c.editorCursorX += 4
		c.editorModified = true
	case tcell.KeyRune:
		// Insert character
		line := c.editorLines[c.editorCursorY]
		c.editorLines[c.editorCursorY] = line[:c.editorCursorX] + string(ev.Rune()) + line[c.editorCursorX:]
		c.editorCursorX++
		c.editorModified = true
	}

	// Adjust scroll to keep cursor visible
	c.adjustEditorScroll()
	return false
}

func (c *Commander) adjustEditorScroll() {
	width, height := c.screen.Size()
	editorHeight := height - 2 // Leave room for header and status
	lineNumWidth := c.getLineNumWidth() + 1
	editorWidth := width - lineNumWidth

	// Vertical scrolling
	if c.editorCursorY < c.editorScrollY {
		c.editorScrollY = c.editorCursorY
	}
	if c.editorCursorY >= c.editorScrollY+editorHeight {
		c.editorScrollY = c.editorCursorY - editorHeight + 1
	}

	// Horizontal scrolling
	if c.editorCursorX < c.editorScrollX {
		c.editorScrollX = c.editorCursorX
	}
	if c.editorCursorX >= c.editorScrollX+editorWidth-1 {
		c.editorScrollX = c.editorCursorX - editorWidth + 2
	}
}

func (c *Commander) getLineNumWidth() int {
	// Calculate width needed for line numbers
	lineCount := len(c.editorLines)
	width := 1
	for lineCount >= 10 {
		lineCount /= 10
		width++
	}
	if width < 3 {
		width = 3
	}
	return width
}

func (c *Commander) saveEditorFile() {
	content := strings.Join(c.editorLines, "\n") + "\n"
	err := os.WriteFile(c.editorFilePath, []byte(content), 0644)
	if err != nil {
		c.setStatus("Error saving: " + err.Error())
	} else {
		c.editorModified = false
		c.setStatus("Saved: " + filepath.Base(c.editorFilePath))
	}
}

func (c *Commander) exitEditor() {
	c.editorMode = false
	c.editorLines = nil
	c.editorFilePath = ""
	c.setStatus("Editor closed")
	// Refresh pane in case file was modified
	c.refreshPane(c.getActivePane())
}

func (c *Commander) drawSearchResults() {
	c.screen.Clear()
	width, height := c.screen.Size()

	// Header style
	headerStyle := tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite).Bold(true)
	colHeaderStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite)
	selectedStyle := tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite)
	normalStyle := tcell.StyleDefault

	// Draw header
	title := fmt.Sprintf(" Search Results: %d matches in %s", len(c.searchResults), c.searchBaseDir)
	if len(title) > width-2 {
		title = title[:width-2]
	}
	c.drawText(0, 0, width, headerStyle, title)

	// Column widths
	typeColWidth := 6
	nameColWidth := 30
	if width < 80 {
		nameColWidth = 20
	}
	pathColWidth := width - typeColWidth - nameColWidth - 4

	// Draw column headers
	colHeader := fmt.Sprintf(" %-*s %-*s %-*s",
		typeColWidth, "Type",
		nameColWidth, "Name",
		pathColWidth, "Location")
	c.drawText(0, 1, width, colHeaderStyle, colHeader)

	// Draw results
	visibleHeight := height - 4
	visibleStart := c.searchResultScroll
	visibleEnd := c.searchResultScroll + visibleHeight
	if visibleEnd > len(c.searchResults) {
		visibleEnd = len(c.searchResults)
	}

	for i := visibleStart; i < visibleEnd; i++ {
		result := c.searchResults[i]
		y := i - c.searchResultScroll + 2

		style := normalStyle
		if i == c.searchResultIdx {
			style = selectedStyle
		}

		// Type column
		typeStr := "FILE"
		if result.IsDir {
			typeStr = "DIR"
		}

		// Name column (truncate if needed)
		name := result.Name
		if len(name) > nameColWidth {
			name = name[:nameColWidth-3] + "..."
		}

		// Path column (show relative path to parent dir)
		relDir := filepath.Dir(result.RelPath)
		if relDir == "." {
			relDir = "./"
		} else {
			relDir = "./" + relDir + "/"
		}
		if len(relDir) > pathColWidth {
			relDir = "..." + relDir[len(relDir)-pathColWidth+3:]
		}

		line := fmt.Sprintf(" %-*s %-*s %-*s",
			typeColWidth, typeStr,
			nameColWidth, name,
			pathColWidth, relDir)
		c.drawText(0, y, width, style, line)
	}

	// Draw status bar
	statusStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorBlack)
	statusLeft := c.statusMsg
	statusRight := fmt.Sprintf("%d/%d", c.searchResultIdx+1, len(c.searchResults))
	padding := width - len(statusLeft) - len(statusRight)
	if padding < 1 {
		padding = 1
	}
	statusText := statusLeft + strings.Repeat(" ", padding) + statusRight
	if len(statusText) > width {
		statusText = statusText[:width]
	}
	c.drawText(0, height-1, width, statusStyle, statusText)

	c.screen.Show()
}

func (c *Commander) drawHashSelection() {
	c.screen.Clear()
	width, height := c.screen.Size()

	// Header style
	headerStyle := tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite).Bold(true)
	selectedStyle := tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite)
	normalStyle := tcell.StyleDefault

	// Draw header
	fileName := filepath.Base(c.hashFilePath)
	title := fmt.Sprintf(" Select Hash Algorithm for: %s", fileName)
	if len(title) > width-2 {
		title = title[:width-2]
	}
	c.drawText(0, 0, width, headerStyle, title)

	// Draw algorithms list
	startY := 2
	for i, algo := range c.hashAlgorithms {
		y := startY + i
		if y >= height-2 { // Leave room for status bar
			break
		}

		style := normalStyle
		if i == c.hashSelectedIdx {
			style = selectedStyle
		}

		line := fmt.Sprintf("  %s", algo)
		c.drawText(0, y, width, style, line)
	}

	// Draw status bar
	statusStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorBlack)
	c.drawText(0, height-1, width, statusStyle, c.statusMsg)

	c.screen.Show()
}

func (c *Commander) drawHashResult() {
	c.screen.Clear()
	width, height := c.screen.Size()

	// Header style
	headerStyle := tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite).Bold(true)
	normalStyle := tcell.StyleDefault
	highlightStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)

	// Draw header
	title := fmt.Sprintf(" Hash Result - %s", c.hashAlgorithm)
	if len(title) > width-2 {
		title = title[:width-2]
	}
	c.drawText(0, 0, width, headerStyle, title)

	// Draw file path
	fileName := filepath.Base(c.hashResultFilePath)
	fileLabel := fmt.Sprintf("  File: %s", fileName)
	if len(fileLabel) > width {
		fileLabel = fileLabel[:width]
	}
	c.drawText(0, 2, width, normalStyle, fileLabel)

	// Draw hash result (wrapped if needed)
	hashLabel := "  Hash:"
	c.drawText(0, 4, width, normalStyle, hashLabel)
	
	// Draw hash value with wrapping for long hashes
	hashValue := c.hashResult
	currentY := 5
	currentX := 2
	maxLineWidth := width - 4

	for len(hashValue) > 0 {
		if currentY >= height-2 { // Leave room for status
			break
		}

		chunkSize := maxLineWidth
		if chunkSize > len(hashValue) {
			chunkSize = len(hashValue)
		}

		chunk := hashValue[:chunkSize]
		hashValue = hashValue[chunkSize:]

		c.drawText(currentX, currentY, len(chunk), highlightStyle, chunk)
		currentY++
	}

	// Draw status bar
	statusStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorBlack)
	c.drawText(0, height-1, width, statusStyle, c.statusMsg)

	c.screen.Show()
}

func (c *Commander) drawEditor() {
	c.screen.Clear()
	width, height := c.screen.Size()

	// Header style
	headerStyle := tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite).Bold(true)
	lineNumStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorDarkGray)
	textStyle := tcell.StyleDefault
	cursorStyle := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)

	// Draw header
	title := c.editorFilePath
	if c.editorModified {
		title += " [modified]"
	}
	if len(title) > width-2 {
		title = "..." + title[len(title)-width+5:]
	}
	c.drawText(0, 0, width, headerStyle, " "+title)

	// Calculate line number width
	lineNumWidth := c.getLineNumWidth()
	editorHeight := height - 2

	// Draw text area with line numbers
	for y := 0; y < editorHeight; y++ {
		lineIdx := c.editorScrollY + y
		screenY := y + 1

		if lineIdx < len(c.editorLines) {
			// Draw line number
			lineNumStr := fmt.Sprintf("%*d ", lineNumWidth, lineIdx+1)
			for i, ch := range lineNumStr {
				c.screen.SetContent(i, screenY, ch, nil, lineNumStyle)
			}

			// Draw line content
			line := c.editorLines[lineIdx]
			textStartX := lineNumWidth + 1
			for x := 0; x < width-textStartX; x++ {
				charIdx := c.editorScrollX + x
				var ch rune = ' '
				if charIdx < len(line) {
					ch = rune(line[charIdx])
				}

				// Highlight cursor position
				style := textStyle
				if lineIdx == c.editorCursorY && charIdx == c.editorCursorX {
					style = cursorStyle
				}
				c.screen.SetContent(textStartX+x, screenY, ch, nil, style)
			}
		} else {
			// Draw empty line with tilde
			lineNumStr := fmt.Sprintf("%*s ", lineNumWidth, "~")
			for i, ch := range lineNumStr {
				c.screen.SetContent(i, screenY, ch, nil, lineNumStyle)
			}
			for x := lineNumWidth + 1; x < width; x++ {
				c.screen.SetContent(x, screenY, ' ', nil, textStyle)
			}
		}
	}

	// Draw status bar
	c.drawEditorStatusBar(height - 1)
	c.screen.Show()
}

func (c *Commander) drawEditorStatusBar(y int) {
	width, _ := c.screen.Size()
	style := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorBlack)

	// Left side: status message
	statusLeft := c.statusMsg
	if statusLeft == "" {
		statusLeft = "Ctrl+S:Save Ctrl+Q:Quit"
	}

	// Right side: cursor position
	statusRight := fmt.Sprintf("Ln %d, Col %d", c.editorCursorY+1, c.editorCursorX+1)

	// Combine
	padding := width - len(statusLeft) - len(statusRight)
	if padding < 1 {
		padding = 1
	}
	statusText := statusLeft + strings.Repeat(" ", padding) + statusRight
	if len(statusText) > width {
		statusText = statusText[:width]
	}

	c.drawText(0, y, width, style, statusText)
}

func (c *Commander) createDirectory() {
	c.inputMode = "newdir"
	c.inputBuffer = ""
	c.inputPrompt = "New directory name: "
	c.setStatus(c.inputPrompt + c.inputBuffer)
}

func (c *Commander) gotoFolder() {
	pane := c.getActivePane()
	c.inputMode = "goto"
	c.inputBuffer = pane.CurrentPath
	c.inputPrompt = "Go to: "
	c.setStatus(c.inputPrompt + c.inputBuffer)
}

func (c *Commander) refreshPane(pane *Pane) error {
	entries, err := os.ReadDir(pane.CurrentPath)
	if err != nil {
		return err
	}

	pane.Files = make([]FileItem, 0, len(entries)+1)

	// Add parent directory link
	parent := filepath.Dir(pane.CurrentPath)
	if parent != pane.CurrentPath {
		pane.Files = append(pane.Files, FileItem{
			Name:  "..",
			IsDir: true,
			Path:  parent,
		})
	}

	// Add all entries
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		ext := ""
		if !entry.IsDir() {
			ext = strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
		}

		item := FileItem{
			Name:    entry.Name(),
			Ext:     ext,
			IsDir:   entry.IsDir(),
			Path:    filepath.Join(pane.CurrentPath, entry.Name()),
			ModTime: info.ModTime(),
		}
		if !entry.IsDir() {
			item.Size = info.Size()
		}
		pane.Files = append(pane.Files, item)
	}

	// Sort: directories first, then files, alphabetically
	sort.Slice(pane.Files, func(i, j int) bool {
		if pane.Files[i].Name == ".." {
			return true
		}
		if pane.Files[j].Name == ".." {
			return false
		}
		if pane.Files[i].IsDir != pane.Files[j].IsDir {
			return pane.Files[i].IsDir
		}
		return strings.ToLower(pane.Files[i].Name) < strings.ToLower(pane.Files[j].Name)
	})

	return nil
}

func (c *Commander) updateLayout() {
	width, height := c.screen.Size()

	paneWidth := (width - 1) / 2

	c.leftPane.Width = paneWidth
	c.leftPane.Height = height - 2

	c.rightPane.Width = width - paneWidth - 1
	c.rightPane.Height = height - 2
}

func (c *Commander) draw() {
	// Check if in editor mode
	if c.editorMode {
		c.drawEditor()
		return
	}

	// Check if in search results mode
	if c.searchResultsMode {
		c.drawSearchResults()
		return
	}

	// Check if in hash selection mode
	if c.hashSelectionMode {
		c.drawHashSelection()
		return
	}

	// Check if in hash result mode
	if c.hashResultMode {
		c.drawHashResult()
		return
	}

	c.screen.Clear()
	_, height := c.screen.Size()

	// Draw left pane
	c.drawPane(c.leftPane, 0, c.activePane == PaneLeft)

	// Draw divider
	dividerX := c.leftPane.Width
	for y := 0; y < height-1; y++ {
		c.screen.SetContent(dividerX, y, '│', nil, tcell.StyleDefault)
	}

	// Draw right pane
	c.drawPane(c.rightPane, dividerX+1, c.activePane == PaneRight)

	// Draw status bar
	c.drawStatusBar(height - 1)

	c.screen.Show()
}

func (c *Commander) drawPane(pane *Pane, offsetX int, active bool) {
	style := tcell.StyleDefault
	headerStyle := tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)
	if active {
		headerStyle = headerStyle.Background(tcell.ColorBlue).Bold(true)
	}

	// Draw path header
	pathDisplay := pane.CurrentPath
	if len(pathDisplay) > pane.Width-2 {
		pathDisplay = "..." + pathDisplay[len(pathDisplay)-pane.Width+5:]
	}
	c.drawText(offsetX, 0, pane.Width, headerStyle, " "+pathDisplay)

	// Column widths: Size(8) + Date(12) + Ext(6) + spacing(4) = 30, rest for name
	sizeColWidth := 8
	dateColWidth := 12
	extColWidth := 6
	fixedWidth := sizeColWidth + dateColWidth + extColWidth + 4 // 4 for spacing
	nameColWidth := pane.Width - fixedWidth
	if nameColWidth < 10 {
		nameColWidth = 10
	}

	// Draw column header
	colHeaderStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite)
	colHeader := fmt.Sprintf(" %-*s %-*s %-*s %*s",
		nameColWidth-1, "Name",
		extColWidth, "Ext",
		dateColWidth, "Modified",
		sizeColWidth, "Size")
	c.drawText(offsetX, 1, pane.Width, colHeaderStyle, colHeader)

	// Draw files
	visibleStart := pane.ScrollOffset
	visibleEnd := pane.ScrollOffset + pane.Height - 4 // -4 for path header, column header, and margins
	if visibleEnd > len(pane.Files) {
		visibleEnd = len(pane.Files)
	}

	for i := visibleStart; i < visibleEnd; i++ {
		file := pane.Files[i]
		y := i - pane.ScrollOffset + 2 // +2 to account for path header and column header

		itemStyle := style
		if i == pane.SelectedIdx {
			if active {
				itemStyle = tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite)
			} else {
				itemStyle = tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorWhite)
			}
		}

		// Format name
		displayName := file.Name
		if file.IsDir {
			displayName = "[" + displayName + "]"
		}
		if len(displayName) > nameColWidth-1 {
			displayName = displayName[:nameColWidth-4] + "..."
		}

		// Format extension
		ext := file.Ext
		if file.IsDir {
			ext = "<DIR>"
		}
		if len(ext) > extColWidth {
			ext = ext[:extColWidth]
		}

		// Format date
		dateStr := ""
		if file.Name != ".." {
			dateStr = file.ModTime.Format("Jan 02 15:04")
		}

		// Format size
		sizeStr := ""
		if !file.IsDir && file.Name != ".." {
			sizeStr = formatSize(file.Size)
		}

		line := fmt.Sprintf(" %-*s %-*s %-*s %*s",
			nameColWidth-1, displayName,
			extColWidth, ext,
			dateColWidth, dateStr,
			sizeColWidth, sizeStr)
		c.drawText(offsetX, y, pane.Width, itemStyle, line)
	}
}

func (c *Commander) drawText(x, y, width int, style tcell.Style, text string) {
	for i := 0; i < width; i++ {
		var ch rune
		if i < len(text) {
			ch = rune(text[i])
		} else {
			ch = ' '
		}
		c.screen.SetContent(x+i, y, ch, nil, style)
	}
}

func (c *Commander) drawStatusBar(y int) {
	width, _ := c.screen.Size()
	style := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorBlack)
	msgStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite).Bold(true)

	// Auto-reset status message after 10 seconds
	if c.statusMsg != "" && time.Since(c.statusMsgTime) > 10*time.Second {
		c.setStatus("")
	}

	shortcuts := "^C:Copy ^X:Move DEL:Del ^F:Find ^E:Edit ^G:Goto ^H:Hash ^N:New ^R:Rename Tab:Switch ESC:Quit"

	// Calculate available space for status message
	statusMsg := c.statusMsg
	separator := " | "

	// Build the status bar: shortcuts first, then status message
	if statusMsg != "" {
		availableForMsg := width - len(shortcuts) - len(separator)
		if availableForMsg > 0 {
			if len(statusMsg) > availableForMsg {
				statusMsg = statusMsg[:availableForMsg-3] + "..."
			}
			// Draw shortcuts
			c.drawText(0, y, len(shortcuts), style, shortcuts)
			// Draw separator
			c.drawText(len(shortcuts), y, len(separator), style, separator)
			// Draw status message with highlighted style
			c.drawText(len(shortcuts)+len(separator), y, width-len(shortcuts)-len(separator), msgStyle, statusMsg)
		} else {
			// Not enough room, just show shortcuts
			c.drawText(0, y, width, style, shortcuts)
		}
	} else {
		// No status message, just show shortcuts
		if len(shortcuts) > width {
			shortcuts = shortcuts[:width]
		}
		c.drawText(0, y, width, style, shortcuts)
	}
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func copyFileOrDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

func main() {
	cmd, err := NewCommander()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
		os.Exit(1)
	}

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running: %v\n", err)
		os.Exit(1)
	}
}
