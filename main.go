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
	"os/exec"
	"path/filepath"
	"runtime"
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
	Name     string
	Ext      string
	IsDir    bool
	Size     int64
	ModTime  time.Time
	Path     string
	Selected bool
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

type DiffBlock struct {
	LeftStart  int
	LeftEnd    int
	RightStart int
	RightEnd   int
	Type       string // "add", "delete", "modify", "equal"
}

type Theme struct {
	Name                 string
	Background           tcell.Color
	Foreground           tcell.Color
	HeaderActive         tcell.Color
	HeaderInactive       tcell.Color
	HeaderText           tcell.Color
	SelectedActive       tcell.Color
	SelectedInactive     tcell.Color
	SelectedText         tcell.Color
	StatusBarBackground  tcell.Color
	StatusBarText        tcell.Color
	StatusMsgText        tcell.Color
	ColumnHeader         tcell.Color
	ColumnHeaderText     tcell.Color
	LineNumber           tcell.Color
	LineNumberBackground tcell.Color
	DiffAdd              tcell.Color
	DiffDelete           tcell.Color
	DiffModify           tcell.Color
	CompareLeftOnly      tcell.Color
	CompareRightOnly     tcell.Color
	CompareDifferent     tcell.Color
	CompareIdentical     tcell.Color
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
	// Archive selection state
	archiveSelectionMode bool
	archiveFormats       []string
	archiveSelectedIdx   int
	// Diff mode state
	diffMode          bool
	diffLeftLines     []string
	diffRightLines    []string
	diffLeftPath      string
	diffRightPath     string
	diffLeftModified  bool
	diffRightModified bool
	diffCurrentIdx    int // Current difference being viewed
	diffDifferences   []DiffBlock
	diffScrollY       int
	diffActiveSide    int // 0 for left, 1 for right
	diffEditMode      bool
	diffCursorX       int
	diffCursorY       int
	// Compare mode state
	compareMode    bool
	compareResults map[string]CompareStatus
	// Help mode state
	helpMode bool
	// Theme state
	currentTheme int
	themes       []Theme
}

type CompareStatus struct {
	Status    string // "left_only", "right_only", "different", "identical"
	LeftFile  *FileItem
	RightFile *FileItem
}

// getDefaultTheme returns the default Dark theme
func getDefaultTheme() Theme {
	return Theme{
		Name:                 "Dark",
		Background:           tcell.ColorBlack,
		Foreground:           tcell.ColorWhite,
		HeaderActive:         tcell.ColorBlue,
		HeaderInactive:       tcell.ColorDarkBlue,
		HeaderText:           tcell.ColorWhite,
		SelectedActive:       tcell.ColorDarkCyan,
		SelectedInactive:     tcell.ColorGray,
		SelectedText:         tcell.ColorWhite,
		StatusBarBackground:  tcell.ColorDarkGray,
		StatusBarText:        tcell.ColorWhite,
		StatusMsgText:        tcell.ColorWhite,
		ColumnHeader:         tcell.ColorDarkGray,
		ColumnHeaderText:     tcell.ColorWhite,
		LineNumber:           tcell.ColorYellow,
		LineNumberBackground: tcell.ColorDarkGray,
		DiffAdd:              tcell.ColorDarkGreen,
		DiffDelete:           tcell.ColorDarkRed,
		DiffModify:           tcell.ColorDarkGoldenrod,
		CompareLeftOnly:      tcell.ColorDarkCyan,
		CompareRightOnly:     tcell.ColorDarkCyan,
		CompareDifferent:     tcell.ColorYellow,
		CompareIdentical:     tcell.ColorDarkGreen,
	}
}

// initThemes creates the predefined color themes
func initThemes() []Theme {
	return []Theme{
		// Dark theme (default)
		getDefaultTheme(),
		// Light theme
		{
			Name:                 "Light",
			Background:           tcell.ColorWhite,
			Foreground:           tcell.ColorBlack,
			HeaderActive:         tcell.ColorBlue,
			HeaderInactive:       tcell.ColorSilver,
			HeaderText:           tcell.ColorBlack,
			SelectedActive:       tcell.ColorSkyblue,
			SelectedInactive:     tcell.ColorSilver,
			SelectedText:         tcell.ColorBlack,
			StatusBarBackground:  tcell.ColorSilver,
			StatusBarText:        tcell.ColorBlack,
			StatusMsgText:        tcell.ColorBlack,
			ColumnHeader:         tcell.ColorSilver,
			ColumnHeaderText:     tcell.ColorBlack,
			LineNumber:           tcell.ColorNavy,
			LineNumberBackground: tcell.ColorSilver,
			DiffAdd:              tcell.ColorLightGreen,
			DiffDelete:           tcell.ColorLightCoral,
			DiffModify:           tcell.ColorLightGoldenrodYellow,
			CompareLeftOnly:      tcell.ColorSkyblue,
			CompareRightOnly:     tcell.ColorSkyblue,
			CompareDifferent:     tcell.ColorGold,
			CompareIdentical:     tcell.ColorLightGreen,
		},
		// Solarized Dark
		{
			Name:                 "Solarized Dark",
			Background:           tcell.NewRGBColor(0, 43, 54),      // base03
			Foreground:           tcell.NewRGBColor(131, 148, 150),  // base0
			HeaderActive:         tcell.NewRGBColor(38, 139, 210),   // blue
			HeaderInactive:       tcell.NewRGBColor(88, 110, 117),   // base01
			HeaderText:           tcell.NewRGBColor(253, 246, 227),  // base3
			SelectedActive:       tcell.NewRGBColor(42, 161, 152),   // cyan
			SelectedInactive:     tcell.NewRGBColor(88, 110, 117),   // base01
			SelectedText:         tcell.NewRGBColor(253, 246, 227),  // base3
			StatusBarBackground:  tcell.NewRGBColor(7, 54, 66),      // base02
			StatusBarText:        tcell.NewRGBColor(101, 123, 131),  // base00
			StatusMsgText:        tcell.NewRGBColor(147, 161, 161),  // base1
			ColumnHeader:         tcell.NewRGBColor(7, 54, 66),      // base02
			ColumnHeaderText:     tcell.NewRGBColor(147, 161, 161),  // base1
			LineNumber:           tcell.NewRGBColor(181, 137, 0),    // yellow
			LineNumberBackground: tcell.NewRGBColor(7, 54, 66),      // base02
			DiffAdd:              tcell.NewRGBColor(133, 153, 0),    // green
			DiffDelete:           tcell.NewRGBColor(220, 50, 47),    // red
			DiffModify:           tcell.NewRGBColor(203, 75, 22),    // orange
			CompareLeftOnly:      tcell.NewRGBColor(42, 161, 152),   // cyan
			CompareRightOnly:     tcell.NewRGBColor(42, 161, 152),   // cyan
			CompareDifferent:     tcell.NewRGBColor(181, 137, 0),    // yellow
			CompareIdentical:     tcell.NewRGBColor(133, 153, 0),    // green
		},
		// Solarized Light
		{
			Name:                 "Solarized Light",
			Background:           tcell.NewRGBColor(253, 246, 227),  // base3
			Foreground:           tcell.NewRGBColor(101, 123, 131),  // base00
			HeaderActive:         tcell.NewRGBColor(38, 139, 210),   // blue
			HeaderInactive:       tcell.NewRGBColor(238, 232, 213),  // base2
			HeaderText:           tcell.NewRGBColor(0, 43, 54),      // base03
			SelectedActive:       tcell.NewRGBColor(42, 161, 152),   // cyan
			SelectedInactive:     tcell.NewRGBColor(238, 232, 213),  // base2
			SelectedText:         tcell.NewRGBColor(0, 43, 54),      // base03
			StatusBarBackground:  tcell.NewRGBColor(238, 232, 213),  // base2
			StatusBarText:        tcell.NewRGBColor(88, 110, 117),   // base01
			StatusMsgText:        tcell.NewRGBColor(88, 110, 117),   // base01
			ColumnHeader:         tcell.NewRGBColor(238, 232, 213),  // base2
			ColumnHeaderText:     tcell.NewRGBColor(88, 110, 117),   // base01
			LineNumber:           tcell.NewRGBColor(181, 137, 0),    // yellow
			LineNumberBackground: tcell.NewRGBColor(238, 232, 213),  // base2
			DiffAdd:              tcell.NewRGBColor(133, 153, 0),    // green
			DiffDelete:           tcell.NewRGBColor(220, 50, 47),    // red
			DiffModify:           tcell.NewRGBColor(203, 75, 22),    // orange
			CompareLeftOnly:      tcell.NewRGBColor(42, 161, 152),   // cyan
			CompareRightOnly:     tcell.NewRGBColor(42, 161, 152),   // cyan
			CompareDifferent:     tcell.NewRGBColor(181, 137, 0),    // yellow
			CompareIdentical:     tcell.NewRGBColor(133, 153, 0),    // green
		},
	}
}

func NewCommander() (*Commander, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}

	// Initialize themes
	themes := initThemes()

	// Set default theme (Dark theme)
	screen.SetStyle(tcell.StyleDefault.
		Foreground(themes[0].Foreground).
		Background(themes[0].Background))
	screen.Clear()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	cmd := &Commander{
		screen:       screen,
		activePane:   PaneLeft,
		currentTheme: 0,
		themes:       themes,
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

// getTheme returns the current theme
func (c *Commander) getTheme() *Theme {
	// Safety check: ensure themes slice is not empty
	if len(c.themes) == 0 {
		// Return default theme if no themes are loaded
		theme := getDefaultTheme()
		return &theme
	}
	
	if c.currentTheme >= 0 && c.currentTheme < len(c.themes) {
		return &c.themes[c.currentTheme]
	}
	// Fallback to first theme if index is invalid
	return &c.themes[0]
}

// cycleTheme switches to the next theme in the list
func (c *Commander) cycleTheme() {
	c.currentTheme++
	if c.currentTheme >= len(c.themes) {
		c.currentTheme = 0
	}

	theme := c.getTheme()
	
	// Update screen default style
	c.screen.SetStyle(tcell.StyleDefault.
		Foreground(theme.Foreground).
		Background(theme.Background))
	c.screen.Clear()
	
	c.setStatus(fmt.Sprintf("Theme: %s", theme.Name))
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
	if c.diffMode {
		return c.handleDiffInput(ev)
	}

	if c.editorMode {
		return c.handleEditorKey(ev)
	}

	if c.searchResultsMode {
		return c.handleSearchResultsKey(ev)
	}

	if c.hashSelectionMode {
		return c.handleHashSelectionKey(ev)
	}

	if c.archiveSelectionMode {
		return c.handleArchiveSelectionKey(ev)
	}

	if c.hashResultMode {
		return c.handleHashResultKey(ev)
	}

	if c.helpMode {
		c.helpMode = false
		return false
	}

	if c.inputMode != "" {
		return c.handleInputKey(ev)
	}

	if c.searchMode {
		return c.handleSearchKey(ev)
	}

	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlQ:
		// If in compare mode, exit it
		if c.compareMode {
			c.exitCompareMode()
			return false
		}
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
		if !c.compareMode {
			c.enterDirectory()
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if !c.compareMode {
			c.goToParent()
		}
	case tcell.KeyRune:
		// Handle spacebar for selection toggle
		if ev.Rune() == ' ' {
			c.toggleSelection()
			return false
		}
		// Handle comparison mode sync operations
		if c.compareMode {
			switch ev.Rune() {
			case '>':
				c.syncLeftToRight()
				return false
			case '<':
				c.syncRightToLeft()
				return false
			case '=':
				c.syncBothWays()
				return false
			}
		}
		// Handle 'h' or 'H' for integrity hash
		if ev.Rune() == 'h' || ev.Rune() == 'H' {
			c.startHashSelection()
			return false
		}
		// Handle 'a' or 'A' for archive
		if ev.Rune() == 'a' || ev.Rune() == 'A' {
			c.startArchiveSelection()
			return false
		}

		// Handle 'b' or 'B' for blank file
		if ev.Rune() == 'b' || ev.Rune() == 'B' {
			c.createBlankFile()
			return false
		}

		// Handle 'c' or 'C' for copy
		if ev.Rune() == 'c' || ev.Rune() == 'C' {
			c.copyFile()
		}

		// Handle 'm' or 'M' for move
		if ev.Rune() == 'm' || ev.Rune() == 'M' {
			c.moveFile()
		}

		// Handle 'r' or 'R' for rename
		if ev.Rune() == 'r' || ev.Rune() == 'R' {
			c.renameFile()
		}

		// Handle 'n' or 'N' for new directory
		if ev.Rune() == 'n' || ev.Rune() == 'N' {
			c.createDirectory()
		}

		// Handle 'e' or 'E' for edit
		if ev.Rune() == 'e' || ev.Rune() == 'E' {
			c.editFile()
		}

		// Handle 'g' or 'G' for goto
		if ev.Rune() == 'g' || ev.Rune() == 'G' {
			c.gotoFolder()
		}

		// Handle 's' or 'S' for find
		if ev.Rune() == 's' || ev.Rune() == 'S' {
			c.startSearch()
		}

		// Handle 'y' or 'Y' for find
		if ev.Rune() == 'y' || ev.Rune() == 'Y' {
			// Toggle compare mode
			if c.compareMode {
				c.exitCompareMode()
			} else {
				c.enterCompareMode()
			}
		}

		// Handle 'f' or 'F' for find
		if ev.Rune() == 'f' || ev.Rune() == 'F' {
			c.enterDiffMode()
		}

		// Handle '?' for help
		if ev.Rune() == '?' {
			c.helpMode = true
			return false
		}

		// Handle 't' or 'T' for theme cycling
		if ev.Rune() == 't' || ev.Rune() == 'T' {
			c.cycleTheme()
			return false
		}
	case tcell.KeyDelete:
		c.deleteFile()

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

	case "newfile":
		if len(c.inputBuffer) == 0 {
			c.setStatus("File name cannot be empty")
			c.inputMode = ""
			c.inputBuffer = ""
			return
		}

		newPath := filepath.Join(pane.CurrentPath, c.inputBuffer)
		err := os.WriteFile(newPath, []byte{}, 0644)
		if err != nil {
			c.setStatus("Error creating file: " + err.Error())
		} else {
			c.setStatus("Created file: " + c.inputBuffer)
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

func (c *Commander) toggleSelection() {
	pane := c.getActivePane()
	if len(pane.Files) == 0 {
		return
	}

	selected := &pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.setStatus("Cannot select parent directory link")
		return
	}

	selected.Selected = !selected.Selected
	if selected.Selected {
		c.setStatus("Selected: " + selected.Name)
	} else {
		c.setStatus("Deselected: " + selected.Name)
	}

	// Move to next item for convenience
	if pane.SelectedIdx < len(pane.Files)-1 {
		c.moveSelection(1)
	}
}

func (c *Commander) startArchiveSelection() {
	pane := c.getActivePane()

	// Check if there are any selected files or a current file to archive
	hasSelection := false
	for _, f := range pane.Files {
		if f.Selected && f.Name != ".." {
			hasSelection = true
			break
		}
	}

	if !hasSelection && len(pane.Files) == 0 {
		c.setStatus("No files to archive")
		return
	}

	if !hasSelection && len(pane.Files) > 0 {
		selected := pane.Files[pane.SelectedIdx]
		if selected.Name == ".." {
			c.setStatus("Cannot archive parent directory link")
			return
		}
	}

	// Detect available archive formats
	c.archiveFormats = c.getAvailableArchiveFormats()
	if len(c.archiveFormats) == 0 {
		c.setStatus("No archive tools available (install zip, tar, 7z, etc.)")
		return
	}

	c.archiveSelectedIdx = 0
	c.archiveSelectionMode = true
	c.setStatus("Select archive format. Enter:Create, Esc:Cancel")
}

func (c *Commander) handleArchiveSelectionKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		c.archiveSelectionMode = false
		c.archiveFormats = nil
		c.setStatus("Archive cancelled")
		return false
	case tcell.KeyEnter:
		if len(c.archiveFormats) > 0 {
			c.createArchive()
		}
		c.archiveSelectionMode = false
		return false
	case tcell.KeyUp:
		if c.archiveSelectedIdx > 0 {
			c.archiveSelectedIdx--
		}
	case tcell.KeyDown:
		if c.archiveSelectedIdx < len(c.archiveFormats)-1 {
			c.archiveSelectedIdx++
		}
	case tcell.KeyHome:
		c.archiveSelectedIdx = 0
	case tcell.KeyEnd:
		c.archiveSelectedIdx = len(c.archiveFormats) - 1
	}
	return false
}

func (c *Commander) getAvailableArchiveFormats() []string {
	formats := []string{}
	zipAdded := false

	// Check for zip command (cross-platform, including third-party Windows installations)
	if _, err := exec.LookPath("zip"); err == nil {
		formats = append(formats, ".zip")
		zipAdded = true
	}

	// On Windows, check for additional zip creation tools
	if runtime.GOOS == "windows" {
		// Check for tar.exe (built-in on Windows 10+)
		if !zipAdded {
			if _, err := exec.LookPath("tar.exe"); err == nil {
				formats = append(formats, ".zip")
				zipAdded = true
			}
		}

		// Check for PowerShell (fallback option)
		if !zipAdded {
			if _, err := exec.LookPath("powershell.exe"); err == nil {
				formats = append(formats, ".zip")
				zipAdded = true
			}
		}
	}

	// Check for 7z (try both 7z and 7za)
	if _, err := exec.LookPath("7z"); err == nil {
		formats = append(formats, ".7z")
	} else if _, err := exec.LookPath("7za"); err == nil {
		formats = append(formats, ".7z")
	}

	// Check for tar
	if _, err := exec.LookPath("tar"); err == nil {
		formats = append(formats, ".tar", ".tar.gz", ".tar.bz2", ".tar.xz")
	}

	return formats
}

func (c *Commander) createArchive() {
	if len(c.archiveFormats) == 0 {
		c.setStatus("Error: No archive format selected")
		return
	}

	format := c.archiveFormats[c.archiveSelectedIdx]
	pane := c.getActivePane()

	// Collect files to archive
	var filesToArchive []FileItem
	for _, f := range pane.Files {
		if f.Selected && f.Name != ".." {
			filesToArchive = append(filesToArchive, f)
		}
	}

	// If nothing selected, use current file
	if len(filesToArchive) == 0 && len(pane.Files) > 0 {
		selected := pane.Files[pane.SelectedIdx]
		if selected.Name != ".." {
			filesToArchive = append(filesToArchive, selected)
		}
	}

	if len(filesToArchive) == 0 {
		c.setStatus("Error: No files to archive")
		c.archiveFormats = nil
		return
	}

	// Generate archive name
	archiveName := c.generateArchiveName(filesToArchive, format)
	archivePath := filepath.Join(pane.CurrentPath, archiveName)

	c.setStatus(fmt.Sprintf("Creating %s archive...", format))
	if c.screen != nil {
		c.draw()
	}

	// Create archive based on format
	var err error
	switch format {
	case ".zip":
		err = c.createZipArchive(archivePath, filesToArchive)
	case ".7z":
		err = c.create7zArchive(archivePath, filesToArchive)
	case ".tar":
		err = c.createTarArchive(archivePath, filesToArchive, "")
	case ".tar.gz":
		err = c.createTarArchive(archivePath, filesToArchive, "gzip")
	case ".tar.bz2":
		err = c.createTarArchive(archivePath, filesToArchive, "bzip2")
	case ".tar.xz":
		err = c.createTarArchive(archivePath, filesToArchive, "xz")
	default:
		err = fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		c.setStatus("Error creating archive: " + err.Error())
	} else {
		c.setStatus("Archive created: " + archiveName)
		// Clear selections
		for i := range pane.Files {
			pane.Files[i].Selected = false
		}
		// Refresh pane to show new archive
		c.refreshPane(pane)
	}

	c.archiveFormats = nil
}

func (c *Commander) generateArchiveName(files []FileItem, format string) string {
	if len(files) == 1 {
		// Single file/folder: use its name
		name := files[0].Name
		// Remove existing extension if it's a file
		if !files[0].IsDir {
			ext := filepath.Ext(name)
			if ext != "" {
				name = name[:len(name)-len(ext)]
			}
		}
		return name + format
	}

	// Multiple files: use timestamp
	now := time.Now()
	return fmt.Sprintf("archive_%s%s", now.Format("20060102_150405"), format)
}

func (c *Commander) createZipArchive(archivePath string, files []FileItem) error {
	pane := c.getActivePane()
	var lastErr error
	var attemptedMethods []string

	// Method 1: Try zip command (cross-platform, including third-party Windows installations)
	if _, err := exec.LookPath("zip"); err == nil {
		attemptedMethods = append(attemptedMethods, "zip command")
		// Build command: zip -r archive.zip file1 file2 ...
		args := []string{"-r", archivePath}
		for _, f := range files {
			args = append(args, f.Name)
		}

		cmd := exec.Command("zip", args...)
		cmd.Dir = pane.CurrentPath
		output, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("zip command failed: %v, output: %s", err, string(output))
	}

	// On Windows, try additional methods
	if runtime.GOOS == "windows" {
		// Method 2: Try tar.exe with --format=zip (built-in on Windows 10+)
		if _, err := exec.LookPath("tar.exe"); err == nil {
			attemptedMethods = append(attemptedMethods, "tar.exe")
			// Build command: tar.exe --format=zip -cf archive.zip file1 file2 ...
			args := []string{"--format=zip", "-cf", archivePath}
			for _, f := range files {
				args = append(args, f.Name)
			}

			cmd := exec.Command("tar.exe", args...)
			cmd.Dir = pane.CurrentPath
			output, err := cmd.CombinedOutput()
			if err == nil {
				return nil
			}
			lastErr = fmt.Errorf("tar.exe failed: %v, output: %s", err, string(output))
		}

		// Method 3: Try PowerShell Compress-Archive
		if _, err := exec.LookPath("powershell.exe"); err == nil {
			attemptedMethods = append(attemptedMethods, "PowerShell Compress-Archive")
			// Build file list for PowerShell
			var pathList []string
			for _, f := range files {
				// Escape single quotes in file names
				escapedName := strings.ReplaceAll(f.Name, "'", "''")
				pathList = append(pathList, fmt.Sprintf("'%s'", escapedName))
			}
			paths := strings.Join(pathList, ",")

			// Escape single quotes in archive path
			escapedArchive := strings.ReplaceAll(archivePath, "'", "''")

			// Build PowerShell command
			psCmd := fmt.Sprintf("Compress-Archive -Path %s -DestinationPath '%s' -Force", paths, escapedArchive)
			cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd)
			cmd.Dir = pane.CurrentPath
			output, err := cmd.CombinedOutput()
			if err == nil {
				return nil
			}
			lastErr = fmt.Errorf("PowerShell Compress-Archive failed: %v, output: %s", err, string(output))
		}
	}

	// If all methods failed, return comprehensive error
	if len(attemptedMethods) > 0 {
		return fmt.Errorf("all zip creation methods failed (tried: %s): %v", strings.Join(attemptedMethods, ", "), lastErr)
	}

	return fmt.Errorf("no zip creation tools available on this system")
}

func (c *Commander) create7zArchive(archivePath string, files []FileItem) error {
	// Build command: 7z a archive.7z file1 file2 ...
	args := []string{"a", archivePath}
	for _, f := range files {
		args = append(args, f.Name)
	}

	// Change to the directory containing the files
	pane := c.getActivePane()

	// Try different 7z command names
	cmdNames := []string{"7z", "7za"}
	var lastErr error

	for _, cmdName := range cmdNames {
		cmd := exec.Command(cmdName, args...)
		cmd.Dir = pane.CurrentPath
		output, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("%s failed: %v, output: %s", cmdName, err, string(output))
	}

	return lastErr
}

func (c *Commander) createTarArchive(archivePath string, files []FileItem, compression string) error {
	// Build command: tar -cf archive.tar file1 file2 ...
	// or: tar -czf archive.tar.gz file1 file2 ...
	args := []string{}

	switch compression {
	case "gzip":
		args = append(args, "-czf")
	case "bzip2":
		args = append(args, "-cjf")
	case "xz":
		args = append(args, "-cJf")
	default:
		args = append(args, "-cf")
	}

	args = append(args, archivePath)
	for _, f := range files {
		args = append(args, f.Name)
	}

	// Change to the directory containing the files
	pane := c.getActivePane()

	// Execute tar command
	cmd := exec.Command("tar", args...)
	cmd.Dir = pane.CurrentPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar failed: %v, output: %s", err, string(output))
	}

	return nil
}

func (c *Commander) copyFile() {
	pane := c.getActivePane()
	destPane := c.getInactivePane()

	if len(pane.Files) == 0 {
		c.setStatus("No file selected")
		return
	}

	// Collect files to copy
	var filesToCopy []FileItem
	for _, f := range pane.Files {
		if f.Selected && f.Name != ".." {
			filesToCopy = append(filesToCopy, f)
		}
	}

	// If nothing selected, use current file
	if len(filesToCopy) == 0 {
		selected := pane.Files[pane.SelectedIdx]
		if selected.Name == ".." {
			c.setStatus("Cannot copy parent directory link")
			return
		}
		filesToCopy = append(filesToCopy, selected)
	}

	// Copy all selected files
	copiedCount := 0
	var lastErr error
	for _, file := range filesToCopy {
		destPath := filepath.Join(destPane.CurrentPath, file.Name)
		err := copyFileOrDir(file.Path, destPath)
		if err != nil {
			lastErr = err
		} else {
			copiedCount++
		}
	}

	// Update status and refresh
	if lastErr != nil {
		c.setStatus(fmt.Sprintf("Copied %d file(s), last error: %s", copiedCount, lastErr.Error()))
	} else {
		if copiedCount == 1 {
			c.setStatus("Copied: " + filesToCopy[0].Name)
		} else {
			c.setStatus(fmt.Sprintf("Copied %d file(s)", copiedCount))
		}
	}

	// Clear selections after copy
	for i := range pane.Files {
		pane.Files[i].Selected = false
	}

	c.refreshPane(destPane)
}

func (c *Commander) moveFile() {
	pane := c.getActivePane()
	destPane := c.getInactivePane()

	if len(pane.Files) == 0 {
		c.setStatus("No file selected")
		return
	}

	// Collect files to move
	var filesToMove []FileItem
	for _, f := range pane.Files {
		if f.Selected && f.Name != ".." {
			filesToMove = append(filesToMove, f)
		}
	}

	// If nothing selected, use current file
	if len(filesToMove) == 0 {
		selected := pane.Files[pane.SelectedIdx]
		if selected.Name == ".." {
			c.setStatus("Cannot move parent directory link")
			return
		}
		filesToMove = append(filesToMove, selected)
	}

	// Move all selected files
	movedCount := 0
	var lastErr error
	for _, file := range filesToMove {
		destPath := filepath.Join(destPane.CurrentPath, file.Name)
		err := os.Rename(file.Path, destPath)
		if err != nil {
			lastErr = err
		} else {
			movedCount++
		}
	}

	// Update status and refresh
	if lastErr != nil {
		c.setStatus(fmt.Sprintf("Moved %d file(s), last error: %s", movedCount, lastErr.Error()))
	} else {
		if movedCount == 1 {
			c.setStatus("Moved: " + filesToMove[0].Name)
		} else {
			c.setStatus(fmt.Sprintf("Moved %d file(s)", movedCount))
		}
	}

	// Clear selections after move
	for i := range pane.Files {
		pane.Files[i].Selected = false
	}

	c.refreshPane(pane)
	c.refreshPane(destPane)
}

func (c *Commander) deleteFile() {
	pane := c.getActivePane()

	if len(pane.Files) == 0 {
		c.setStatus("No file selected")
		return
	}

	// Collect files to delete
	var filesToDelete []FileItem
	for _, f := range pane.Files {
		if f.Selected && f.Name != ".." {
			filesToDelete = append(filesToDelete, f)
		}
	}

	// If nothing selected, use current file
	if len(filesToDelete) == 0 {
		selected := pane.Files[pane.SelectedIdx]
		if selected.Name == ".." {
			c.setStatus("Cannot delete parent directory link")
			return
		}
		filesToDelete = append(filesToDelete, selected)
	}

	// Delete all selected files
	deletedCount := 0
	var lastErr error
	for _, file := range filesToDelete {
		var err error
		if file.IsDir {
			err = os.RemoveAll(file.Path)
		} else {
			err = os.Remove(file.Path)
		}
		if err != nil {
			lastErr = err
		} else {
			deletedCount++
		}
	}

	// Update status
	if lastErr != nil {
		c.setStatus(fmt.Sprintf("Deleted %d file(s), last error: %s", deletedCount, lastErr.Error()))
	} else {
		if deletedCount == 1 {
			c.setStatus("Deleted: " + filesToDelete[0].Name)
		} else {
			c.setStatus(fmt.Sprintf("Deleted %d file(s)", deletedCount))
		}
	}

	// Move cursor up if needed
	if pane.SelectedIdx > 0 && pane.SelectedIdx >= len(pane.Files)-deletedCount {
		pane.SelectedIdx--
	}

	c.refreshPane(pane)
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
	theme := c.getTheme()

	// Header style
	headerStyle := tcell.StyleDefault.Background(theme.HeaderActive).Foreground(theme.HeaderText).Bold(true)
	colHeaderStyle := tcell.StyleDefault.Background(theme.ColumnHeader).Foreground(theme.ColumnHeaderText)
	selectedStyle := tcell.StyleDefault.Background(theme.SelectedActive).Foreground(theme.SelectedText)
	normalStyle := tcell.StyleDefault.Foreground(theme.Foreground).Background(theme.Background)

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
	statusStyle := tcell.StyleDefault.Background(theme.StatusBarBackground).Foreground(theme.StatusBarText)
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
	theme := c.getTheme()

	// Header style
	headerStyle := tcell.StyleDefault.Background(theme.HeaderActive).Foreground(theme.HeaderText).Bold(true)
	selectedStyle := tcell.StyleDefault.Background(theme.SelectedActive).Foreground(theme.SelectedText)
	normalStyle := tcell.StyleDefault.Foreground(theme.Foreground).Background(theme.Background)

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
	statusStyle := tcell.StyleDefault.Background(theme.StatusBarBackground).Foreground(theme.StatusBarText)
	c.drawText(0, height-1, width, statusStyle, c.statusMsg)

	c.screen.Show()
}

func (c *Commander) drawArchiveSelection() {
	c.screen.Clear()
	width, height := c.screen.Size()
	theme := c.getTheme()

	// Header style
	headerStyle := tcell.StyleDefault.Background(theme.HeaderActive).Foreground(theme.HeaderText).Bold(true)
	selectedStyle := tcell.StyleDefault.Background(theme.SelectedActive).Foreground(theme.SelectedText)
	normalStyle := tcell.StyleDefault.Foreground(theme.Foreground).Background(theme.Background)

	// Count selected files
	pane := c.getActivePane()
	selectedCount := 0
	for _, f := range pane.Files {
		if f.Selected && f.Name != ".." {
			selectedCount++
		}
	}

	// Draw header
	title := " Select Archive Format"
	if selectedCount > 0 {
		title = fmt.Sprintf(" Select Archive Format (%d file(s) selected)", selectedCount)
	} else {
		currentFile := ""
		if len(pane.Files) > 0 && pane.SelectedIdx < len(pane.Files) {
			currentFile = pane.Files[pane.SelectedIdx].Name
		}
		title = fmt.Sprintf(" Select Archive Format for: %s", currentFile)
	}
	if len(title) > width-2 {
		title = title[:width-2]
	}
	c.drawText(0, 0, width, headerStyle, title)

	// Draw formats list
	startY := 2
	for i, format := range c.archiveFormats {
		y := startY + i
		if y >= height-2 { // Leave room for status bar
			break
		}

		style := normalStyle
		if i == c.archiveSelectedIdx {
			style = selectedStyle
		}

		line := fmt.Sprintf("  %s", format)
		c.drawText(0, y, width, style, line)
	}

	// Draw status bar
	statusStyle := tcell.StyleDefault.Background(theme.StatusBarBackground).Foreground(theme.StatusBarText)
	c.drawText(0, height-1, width, statusStyle, c.statusMsg)

	c.screen.Show()
}

func (c *Commander) drawHashResult() {
	c.screen.Clear()
	width, height := c.screen.Size()
	theme := c.getTheme()

	// Header style
	headerStyle := tcell.StyleDefault.Background(theme.HeaderActive).Foreground(theme.HeaderText).Bold(true)
	normalStyle := tcell.StyleDefault.Foreground(theme.Foreground).Background(theme.Background)
	highlightStyle := tcell.StyleDefault.Foreground(theme.LineNumber).Bold(true)

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
	statusStyle := tcell.StyleDefault.Background(theme.StatusBarBackground).Foreground(theme.StatusBarText)
	c.drawText(0, height-1, width, statusStyle, c.statusMsg)

	c.screen.Show()
}

func (c *Commander) drawHelp() {
	c.screen.Clear()
	width, height := c.screen.Size()
	theme := c.getTheme()

	// Header style
	headerStyle := tcell.StyleDefault.Background(theme.HeaderActive).Foreground(theme.HeaderText).Bold(true)
	normalStyle := tcell.StyleDefault.Foreground(theme.Foreground).Background(theme.Background)

	// Draw header
	title := " Terminal Commander - Help"
	c.drawText(0, 0, width, headerStyle, title)

	// Help content
	helpLines := []string{
		"",
		" Navigation:",
		"  Arrow Keys         Navigate files/directories",
		"  Tab                Switch between panes",
		"  Enter              Enter directory",
		"  Backspace          Go to parent directory",
		"",
		" File Operations:",
		"  r/R                Rename file/directory",
		"  e/E                Edit file",
		"  c/C                Copy file/directory",
		"  m/M                Move file/directory",
		"  Delete             Delete file/directory",
		"  b/B                Create blank file",
		"",
		" Directory Operations:",
		"  n/N                Create new directory",
		"  g/G                Go to folder",
		"",
		" Selection & Archive:",
		"  Space              Toggle selection",
		"  a/A                Archive selected files",
		"  Ctrl+A             Archive selection mode",
		"",
		" Search & Compare:",
		"  s/S                Search files",
		"  f/F                Diff mode",
		"  y/Y                Toggle compare mode",
		"",
		" Hash & Integrity:",
		"  h/H                Integrity hash selection",
		"",
		" Display:",
		"  t/T                Cycle color themes",
		"",
		" Other:",
		"  ?                  Show this help",
		"  Ctrl+Q             Quit",
		"",
		" Compare Mode:",
		"  >                  Sync left to right",
		"  <                  Sync right to left",
		"  =                  Sync both ways",
		"",
		" Input Mode:",
		"  Enter              Confirm",
		"  Escape             Cancel",
	}

	y := 2
	for _, line := range helpLines {
		if y >= height-2 {
			break
		}
		c.drawText(0, y, width, normalStyle, line)
		y++
	}

	// Draw status bar
	statusStyle := tcell.StyleDefault.Background(theme.StatusBarBackground).Foreground(theme.StatusBarText)
	statusMsg := "Press any key to close help"
	c.drawText(0, height-1, width, statusStyle, statusMsg)

	c.screen.Show()
}

func (c *Commander) drawEditor() {
	c.screen.Clear()
	width, height := c.screen.Size()
	theme := c.getTheme()

	// Header style
	headerStyle := tcell.StyleDefault.Background(theme.HeaderActive).Foreground(theme.HeaderText).Bold(true)
	lineNumStyle := tcell.StyleDefault.Foreground(theme.LineNumber).Background(theme.LineNumberBackground)
	textStyle := tcell.StyleDefault.Foreground(theme.Foreground).Background(theme.Background)
	cursorStyle := tcell.StyleDefault.Background(theme.SelectedActive).Foreground(theme.SelectedText)

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
	theme := c.getTheme()
	style := tcell.StyleDefault.Background(theme.StatusBarBackground).Foreground(theme.StatusBarText)

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

func (c *Commander) createBlankFile() {
	c.inputMode = "newfile"
	c.inputBuffer = ""
	c.inputPrompt = "New file name: "
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
			Name:     "..",
			IsDir:    true,
			Path:     parent,
			Selected: false,
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
			Name:     entry.Name(),
			Ext:      ext,
			IsDir:    entry.IsDir(),
			Path:     filepath.Join(pane.CurrentPath, entry.Name()),
			ModTime:  info.ModTime(),
			Selected: false,
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
	// Check if in diff mode
	if c.diffMode {
		c.drawDiff()
		return
	}

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

	// Check if in archive selection mode
	if c.archiveSelectionMode {
		c.drawArchiveSelection()
		return
	}

	// Check if in hash result mode
	if c.hashResultMode {
		c.drawHashResult()
		return
	}

	// Check if in help mode
	if c.helpMode {
		c.drawHelp()
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
	theme := c.getTheme()
	style := tcell.StyleDefault.Foreground(theme.Foreground).Background(theme.Background)
	
	headerStyle := tcell.StyleDefault.Background(theme.HeaderInactive).Foreground(theme.HeaderText)
	if active {
		headerStyle = tcell.StyleDefault.Background(theme.HeaderActive).Foreground(theme.HeaderText).Bold(true)
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
	colHeaderStyle := tcell.StyleDefault.Background(theme.ColumnHeader).Foreground(theme.ColumnHeaderText)
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
				itemStyle = tcell.StyleDefault.Background(theme.SelectedActive).Foreground(theme.SelectedText)
			} else {
				itemStyle = tcell.StyleDefault.Background(theme.SelectedInactive).Foreground(theme.SelectedText)
			}
		}

		// Add comparison indicator if in compare mode
		compareIndicator := ""
		compareColor := tcell.ColorDefault
		if c.compareMode && file.Name != ".." {
			if status, exists := c.compareResults[file.Name]; exists {
				switch status.Status {
				case "left_only":
					compareIndicator = "[L] "
					compareColor = theme.CompareLeftOnly
				case "right_only":
					compareIndicator = "[R] "
					compareColor = theme.CompareRightOnly
				case "different":
					compareIndicator = "[D] "
					compareColor = theme.CompareDifferent
				case "identical":
					compareIndicator = "[=] "
					compareColor = theme.CompareIdentical
				}
				// Override item style with comparison color if not selected
				if i != pane.SelectedIdx {
					itemStyle = tcell.StyleDefault.Foreground(compareColor).Background(theme.Background)
				}
			}
		}

		// Format name
		displayName := file.Name
		if file.IsDir {
			displayName = "[" + displayName + "]"
		}
		// Add selection marker
		if file.Selected {
			displayName = "[*] " + displayName
		}
		// Add comparison indicator
		if compareIndicator != "" {
			displayName = compareIndicator + displayName
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
	theme := c.getTheme()
	style := tcell.StyleDefault.Background(theme.StatusBarBackground).Foreground(theme.StatusBarText)
	msgStyle := tcell.StyleDefault.Background(theme.StatusBarBackground).Foreground(theme.StatusMsgText).Bold(true)

	// Auto-reset status message after 10 seconds
	if c.statusMsg != "" && time.Since(c.statusMsgTime) > 10*time.Second {
		c.setStatus("")
	}

	shortcuts := "SPC:Select A:Archive C:Copy M:Move DEL:Del S:Search E:Edit G:Goto H:Hash N:New_Dir B:New_File R:Rename Y:Diff_Dir F:Diff_File T:Theme Tab:Switch ESC:Quit"

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

// enterDiffMode validates and enters diff mode
func (c *Commander) enterDiffMode() {
	// Check both panes have files selected
	if len(c.leftPane.Files) == 0 || len(c.rightPane.Files) == 0 {
		c.setStatus("Both panes must have a file selected")
		return
	}

	leftFile := c.leftPane.Files[c.leftPane.SelectedIdx]
	rightFile := c.rightPane.Files[c.rightPane.SelectedIdx]

	// Check both are files (not directories)
	if leftFile.IsDir || rightFile.IsDir {
		c.setStatus("Both selections must be files, not directories")
		return
	}

	// Check neither is parent directory link
	if leftFile.Name == ".." || rightFile.Name == ".." {
		c.setStatus("Cannot diff parent directory link")
		return
	}

	// Read left file
	leftContent, err := os.ReadFile(leftFile.Path)
	if err != nil {
		c.setStatus("Error reading left file: " + err.Error())
		return
	}

	// Read right file
	rightContent, err := os.ReadFile(rightFile.Path)
	if err != nil {
		c.setStatus("Error reading right file: " + err.Error())
		return
	}

	// Check if files are text files (basic check)
	if !isTextFile(leftContent) || !isTextFile(rightContent) {
		c.setStatus("Both files must be readable text files")
		return
	}

	// Split into lines
	c.diffLeftLines = strings.Split(string(leftContent), "\n")
	c.diffRightLines = strings.Split(string(rightContent), "\n")

	// Remove trailing empty line if file ends with newline
	if len(c.diffLeftLines) > 0 && c.diffLeftLines[len(c.diffLeftLines)-1] == "" {
		c.diffLeftLines = c.diffLeftLines[:len(c.diffLeftLines)-1]
	}
	if len(c.diffRightLines) > 0 && c.diffRightLines[len(c.diffRightLines)-1] == "" {
		c.diffRightLines = c.diffRightLines[:len(c.diffRightLines)-1]
	}

	// Ensure at least one line
	if len(c.diffLeftLines) == 0 {
		c.diffLeftLines = []string{""}
	}
	if len(c.diffRightLines) == 0 {
		c.diffRightLines = []string{""}
	}

	c.diffLeftPath = leftFile.Path
	c.diffRightPath = rightFile.Path
	c.diffLeftModified = false
	c.diffRightModified = false
	c.diffCurrentIdx = 0
	c.diffScrollY = 0
	c.diffActiveSide = 0
	c.diffEditMode = false
	c.diffCursorX = 0
	c.diffCursorY = 0

	// Calculate differences
	c.calculateDiff()

	c.diffMode = true
	c.setStatus("Diff mode: f/F/ESC:Exit n:Next p:Prev >:Copy→ <:Copy← e:Edit Ctrl+S:Save")
}

// isTextFile checks if content appears to be text
func isTextFile(content []byte) bool {
	// Check for null bytes (binary file indicator)
	for i := 0; i < len(content) && i < 8192; i++ {
		if content[i] == 0 {
			return false
		}
	}
	return true
}

// calculateDiff computes differences between left and right files
func (c *Commander) calculateDiff() {
	c.diffDifferences = []DiffBlock{}

	leftLen := len(c.diffLeftLines)
	rightLen := len(c.diffRightLines)

	// Simple line-by-line comparison algorithm
	// This is a basic implementation; Myers diff would be more sophisticated
	leftIdx := 0
	rightIdx := 0

	for leftIdx < leftLen || rightIdx < rightLen {
		// Check if lines match
		if leftIdx < leftLen && rightIdx < rightLen && c.diffLeftLines[leftIdx] == c.diffRightLines[rightIdx] {
			// Equal block
			equalStart := leftIdx
			for leftIdx < leftLen && rightIdx < rightLen && c.diffLeftLines[leftIdx] == c.diffRightLines[rightIdx] {
				leftIdx++
				rightIdx++
			}
			c.diffDifferences = append(c.diffDifferences, DiffBlock{
				LeftStart:  equalStart,
				LeftEnd:    leftIdx - 1,
				RightStart: equalStart,
				RightEnd:   rightIdx - 1,
				Type:       "equal",
			})
		} else {
			// Different block - find the extent
			diffLeftStart := leftIdx
			diffRightStart := rightIdx

			// Advance through differences until we find a match or reach end
			foundMatch := false
			for !foundMatch && (leftIdx < leftLen || rightIdx < rightLen) {
				// Look ahead to find matching lines
				if leftIdx < leftLen && rightIdx < rightLen {
					// Check if current lines match
					if c.diffLeftLines[leftIdx] == c.diffRightLines[rightIdx] {
						foundMatch = true
						break
					}

					// Look ahead a few lines to find sync point
					matchFound := false
					for lookAhead := 1; lookAhead <= 3 && !matchFound; lookAhead++ {
						if leftIdx+lookAhead < leftLen && c.diffLeftLines[leftIdx+lookAhead] == c.diffRightLines[rightIdx] {
							// Found match, advance left
							leftIdx++
							matchFound = true
							break
						}
						if rightIdx+lookAhead < rightLen && c.diffLeftLines[leftIdx] == c.diffRightLines[rightIdx+lookAhead] {
							// Found match, advance right
							rightIdx++
							matchFound = true
							break
						}
					}

					if !matchFound {
						// No match found nearby, advance both
						leftIdx++
						rightIdx++
					}
				} else if leftIdx < leftLen {
					leftIdx++
				} else {
					rightIdx++
				}
			}

			// Determine type of difference
			diffType := "modify"
			if diffLeftStart >= leftLen {
				diffType = "add" // Lines only in right
			} else if diffRightStart >= rightLen {
				diffType = "delete" // Lines only in left
			} else if leftIdx-diffLeftStart == 0 {
				diffType = "add"
			} else if rightIdx-diffRightStart == 0 {
				diffType = "delete"
			}

			if diffLeftStart < leftIdx || diffRightStart < rightIdx {
				c.diffDifferences = append(c.diffDifferences, DiffBlock{
					LeftStart:  diffLeftStart,
					LeftEnd:    leftIdx - 1,
					RightStart: diffRightStart,
					RightEnd:   rightIdx - 1,
					Type:       diffType,
				})
			}
		}
	}

	// If no differences found, add one equal block for the whole file
	if len(c.diffDifferences) == 0 {
		c.diffDifferences = append(c.diffDifferences, DiffBlock{
			LeftStart:  0,
			LeftEnd:    leftLen - 1,
			RightStart: 0,
			RightEnd:   rightLen - 1,
			Type:       "equal",
		})
	}
}

// drawDiff renders the diff view
func (c *Commander) drawDiff() {
	c.screen.Clear()
	width, height := c.screen.Size()
	theme := c.getTheme()

	// Styles
	headerStyle := tcell.StyleDefault.Background(theme.HeaderActive).Foreground(theme.HeaderText).Bold(true)
	normalStyle := tcell.StyleDefault.Foreground(theme.Foreground).Background(theme.Background)
	deleteStyle := tcell.StyleDefault.Background(theme.DiffDelete).Foreground(theme.SelectedText)
	addStyle := tcell.StyleDefault.Background(theme.DiffAdd).Foreground(theme.SelectedText)
	modifyStyle := tcell.StyleDefault.Background(theme.DiffModify).Foreground(theme.SelectedText)
	lineNumStyle := tcell.StyleDefault.Foreground(theme.LineNumber).Background(theme.LineNumberBackground)

	// Calculate pane widths
	halfWidth := (width - 1) / 2
	lineNumWidth := 5

	// Draw headers
	leftHeader := " Left: " + filepath.Base(c.diffLeftPath)
	if c.diffLeftModified {
		leftHeader += " [modified]"
	}
	if len(leftHeader) > halfWidth {
		leftHeader = leftHeader[:halfWidth-3] + "..."
	}
	c.drawText(0, 0, halfWidth, headerStyle, leftHeader)

	rightHeader := " Right: " + filepath.Base(c.diffRightPath)
	if c.diffRightModified {
		rightHeader += " [modified]"
	}
	if len(rightHeader) > halfWidth {
		rightHeader = rightHeader[:halfWidth-3] + "..."
	}
	c.drawText(halfWidth+1, 0, halfWidth, headerStyle, rightHeader)

	// Draw separator
	for y := 0; y < height-1; y++ {
		c.screen.SetContent(halfWidth, y, '│', nil, normalStyle)
	}

	// Draw file contents
	visibleHeight := height - 2 // Leave room for header and status
	maxLines := len(c.diffLeftLines)
	if len(c.diffRightLines) > maxLines {
		maxLines = len(c.diffRightLines)
	}

	for y := 0; y < visibleHeight; y++ {
		lineIdx := c.diffScrollY + y
		screenY := y + 1

		if lineIdx >= maxLines {
			break
		}

		// Determine line style based on differences
		leftStyle := normalStyle
		rightStyle := normalStyle

		for _, diff := range c.diffDifferences {
			if lineIdx >= diff.LeftStart && lineIdx <= diff.LeftEnd {
				if diff.Type == "delete" {
					leftStyle = deleteStyle
				} else if diff.Type == "modify" {
					leftStyle = modifyStyle
				}
			}
			if lineIdx >= diff.RightStart && lineIdx <= diff.RightEnd {
				if diff.Type == "add" {
					rightStyle = addStyle
				} else if diff.Type == "modify" {
					rightStyle = modifyStyle
				}
			}
		}

		// Draw left side
		leftLineNum := ""
		leftContent := ""
		if lineIdx < len(c.diffLeftLines) {
			leftLineNum = fmt.Sprintf("%4d ", lineIdx+1)
			leftContent = c.diffLeftLines[lineIdx]
		}

		// Draw left line number
		for i, ch := range leftLineNum {
			c.screen.SetContent(i, screenY, ch, nil, lineNumStyle)
		}

		// Draw left content
		maxContentWidth := halfWidth - lineNumWidth
		for x := 0; x < maxContentWidth; x++ {
			var ch rune = ' '
			if x < len(leftContent) {
				ch = rune(leftContent[x])
			}
			c.screen.SetContent(lineNumWidth+x, screenY, ch, nil, leftStyle)
		}

		// Draw right side
		rightLineNum := ""
		rightContent := ""
		if lineIdx < len(c.diffRightLines) {
			rightLineNum = fmt.Sprintf("%4d ", lineIdx+1)
			rightContent = c.diffRightLines[lineIdx]
		}

		// Draw right line number
		for i, ch := range rightLineNum {
			c.screen.SetContent(halfWidth+1+i, screenY, ch, nil, lineNumStyle)
		}

		// Draw right content
		for x := 0; x < maxContentWidth; x++ {
			var ch rune = ' '
			if x < len(rightContent) {
				ch = rune(rightContent[x])
			}
			c.screen.SetContent(halfWidth+1+lineNumWidth+x, screenY, ch, nil, rightStyle)
		}
	}

	// Draw status bar
	statusStyle := tcell.StyleDefault.Background(theme.StatusBarBackground).Foreground(theme.StatusBarText)
	statusText := c.statusMsg
	if statusText == "" {
		diffCount := 0
		for _, d := range c.diffDifferences {
			if d.Type != "equal" {
				diffCount++
			}
		}
		statusText = fmt.Sprintf("f/F/ESC:Exit n:Next p:Prev >:Copy→ <:Copy← e:Edit Ctrl+S:Save | %d differences", diffCount)
	}
	if len(statusText) > width {
		statusText = statusText[:width]
	}
	c.drawText(0, height-1, width, statusStyle, statusText)

	c.screen.Show()
}

// handleDiffInput handles keyboard input in diff mode
func (c *Commander) handleDiffInput(ev *tcell.EventKey) bool {
	// Handle edit mode within diff
	if c.diffEditMode {
		return c.handleDiffEditKey(ev)
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		return c.exitDiffMode()
	case tcell.KeyCtrlQ:
		return c.exitDiffMode()
	case tcell.KeyUp:
		if c.diffScrollY > 0 {
			c.diffScrollY--
		}
	case tcell.KeyDown:
		maxLines := len(c.diffLeftLines)
		if len(c.diffRightLines) > maxLines {
			maxLines = len(c.diffRightLines)
		}
		if c.diffScrollY < maxLines-1 {
			c.diffScrollY++
		}
	case tcell.KeyPgUp:
		_, height := c.screen.Size()
		pageSize := height - 2
		c.diffScrollY -= pageSize
		if c.diffScrollY < 0 {
			c.diffScrollY = 0
		}
	case tcell.KeyPgDn:
		_, height := c.screen.Size()
		pageSize := height - 2
		maxLines := len(c.diffLeftLines)
		if len(c.diffRightLines) > maxLines {
			maxLines = len(c.diffRightLines)
		}
		c.diffScrollY += pageSize
		if c.diffScrollY >= maxLines {
			c.diffScrollY = maxLines - 1
		}
		if c.diffScrollY < 0 {
			c.diffScrollY = 0
		}
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'n', 'N':
			c.jumpToNextDiff()
		case 'p', 'P':
			c.jumpToPrevDiff()
		case '>':
			c.copyDiffLeftToRight()
		case '<':
			c.copyDiffRightToLeft()
		case 'e', 'E':
			c.enterDiffEditMode()
		}
	case tcell.KeyCtrlS:
		c.saveDiffFiles()
	}

	return false
}

// handleDiffEditKey handles keyboard input in diff edit mode
func (c *Commander) handleDiffEditKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		c.diffEditMode = false
		c.calculateDiff()
		c.setStatus("Edit mode exited")
		return false
	case tcell.KeyUp:
		if c.diffCursorY > 0 {
			c.diffCursorY--
			lines := c.diffLeftLines
			if c.diffActiveSide == 1 {
				lines = c.diffRightLines
			}
			if c.diffCursorX > len(lines[c.diffCursorY]) {
				c.diffCursorX = len(lines[c.diffCursorY])
			}
		}
	case tcell.KeyDown:
		lines := c.diffLeftLines
		if c.diffActiveSide == 1 {
			lines = c.diffRightLines
		}
		if c.diffCursorY < len(lines)-1 {
			c.diffCursorY++
			if c.diffCursorX > len(lines[c.diffCursorY]) {
				c.diffCursorX = len(lines[c.diffCursorY])
			}
		}
	case tcell.KeyLeft:
		if c.diffCursorX > 0 {
			c.diffCursorX--
		}
	case tcell.KeyRight:
		lines := c.diffLeftLines
		if c.diffActiveSide == 1 {
			lines = c.diffRightLines
		}
		if c.diffCursorX < len(lines[c.diffCursorY]) {
			c.diffCursorX++
		}
	case tcell.KeyHome:
		c.diffCursorX = 0
	case tcell.KeyEnd:
		lines := c.diffLeftLines
		if c.diffActiveSide == 1 {
			lines = c.diffRightLines
		}
		c.diffCursorX = len(lines[c.diffCursorY])
	case tcell.KeyEnter:
		// Insert new line
		lines := &c.diffLeftLines
		if c.diffActiveSide == 1 {
			lines = &c.diffRightLines
		}
		line := (*lines)[c.diffCursorY]
		leftPart := line[:c.diffCursorX]
		rightPart := line[c.diffCursorX:]
		(*lines)[c.diffCursorY] = leftPart
		newLines := make([]string, len(*lines)+1)
		copy(newLines, (*lines)[:c.diffCursorY+1])
		newLines[c.diffCursorY+1] = rightPart
		copy(newLines[c.diffCursorY+2:], (*lines)[c.diffCursorY+1:])
		*lines = newLines
		c.diffCursorY++
		c.diffCursorX = 0
		if c.diffActiveSide == 0 {
			c.diffLeftModified = true
		} else {
			c.diffRightModified = true
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		lines := &c.diffLeftLines
		if c.diffActiveSide == 1 {
			lines = &c.diffRightLines
		}
		if c.diffCursorX > 0 {
			line := (*lines)[c.diffCursorY]
			(*lines)[c.diffCursorY] = line[:c.diffCursorX-1] + line[c.diffCursorX:]
			c.diffCursorX--
			if c.diffActiveSide == 0 {
				c.diffLeftModified = true
			} else {
				c.diffRightModified = true
			}
		} else if c.diffCursorY > 0 {
			prevLineLen := len((*lines)[c.diffCursorY-1])
			(*lines)[c.diffCursorY-1] += (*lines)[c.diffCursorY]
			*lines = append((*lines)[:c.diffCursorY], (*lines)[c.diffCursorY+1:]...)
			c.diffCursorY--
			c.diffCursorX = prevLineLen
			if c.diffActiveSide == 0 {
				c.diffLeftModified = true
			} else {
				c.diffRightModified = true
			}
		}
	case tcell.KeyDelete:
		lines := &c.diffLeftLines
		if c.diffActiveSide == 1 {
			lines = &c.diffRightLines
		}
		line := (*lines)[c.diffCursorY]
		if c.diffCursorX < len(line) {
			(*lines)[c.diffCursorY] = line[:c.diffCursorX] + line[c.diffCursorX+1:]
			if c.diffActiveSide == 0 {
				c.diffLeftModified = true
			} else {
				c.diffRightModified = true
			}
		} else if c.diffCursorY < len(*lines)-1 {
			(*lines)[c.diffCursorY] += (*lines)[c.diffCursorY+1]
			*lines = append((*lines)[:c.diffCursorY+1], (*lines)[c.diffCursorY+2:]...)
			if c.diffActiveSide == 0 {
				c.diffLeftModified = true
			} else {
				c.diffRightModified = true
			}
		}
	case tcell.KeyRune:
		lines := &c.diffLeftLines
		if c.diffActiveSide == 1 {
			lines = &c.diffRightLines
		}
		line := (*lines)[c.diffCursorY]
		(*lines)[c.diffCursorY] = line[:c.diffCursorX] + string(ev.Rune()) + line[c.diffCursorX:]
		c.diffCursorX++
		if c.diffActiveSide == 0 {
			c.diffLeftModified = true
		} else {
			c.diffRightModified = true
		}
	}

	return false
}

// jumpToNextDiff jumps to the next difference
func (c *Commander) jumpToNextDiff() {
	if len(c.diffDifferences) == 0 {
		return
	}

	// Find next non-equal diff
	for i := c.diffCurrentIdx + 1; i < len(c.diffDifferences); i++ {
		if c.diffDifferences[i].Type != "equal" {
			c.diffCurrentIdx = i
			c.diffScrollY = c.diffDifferences[i].LeftStart
			c.setStatus(fmt.Sprintf("Difference %d/%d", i+1, len(c.diffDifferences)))
			return
		}
	}

	// Wrap around to first diff
	for i := 0; i <= c.diffCurrentIdx; i++ {
		if c.diffDifferences[i].Type != "equal" {
			c.diffCurrentIdx = i
			c.diffScrollY = c.diffDifferences[i].LeftStart
			c.setStatus(fmt.Sprintf("Difference %d/%d (wrapped)", i+1, len(c.diffDifferences)))
			return
		}
	}

	c.setStatus("No differences found")
}

// jumpToPrevDiff jumps to the previous difference
func (c *Commander) jumpToPrevDiff() {
	if len(c.diffDifferences) == 0 {
		return
	}

	// Find previous non-equal diff
	for i := c.diffCurrentIdx - 1; i >= 0; i-- {
		if c.diffDifferences[i].Type != "equal" {
			c.diffCurrentIdx = i
			c.diffScrollY = c.diffDifferences[i].LeftStart
			c.setStatus(fmt.Sprintf("Difference %d/%d", i+1, len(c.diffDifferences)))
			return
		}
	}

	// Wrap around to last diff
	for i := len(c.diffDifferences) - 1; i >= c.diffCurrentIdx; i-- {
		if c.diffDifferences[i].Type != "equal" {
			c.diffCurrentIdx = i
			c.diffScrollY = c.diffDifferences[i].LeftStart
			c.setStatus(fmt.Sprintf("Difference %d/%d (wrapped)", i+1, len(c.diffDifferences)))
			return
		}
	}

	c.setStatus("No differences found")
}

// copyDiffLeftToRight copies current difference from left to right
func (c *Commander) copyDiffLeftToRight() {
	if c.diffCurrentIdx < 0 || c.diffCurrentIdx >= len(c.diffDifferences) {
		c.setStatus("No difference selected")
		return
	}

	diff := c.diffDifferences[c.diffCurrentIdx]
	if diff.Type == "equal" {
		c.setStatus("No difference at current position")
		return
	}

	// Extract lines from left
	var leftLines []string
	if diff.LeftStart <= diff.LeftEnd && diff.LeftEnd < len(c.diffLeftLines) {
		leftLines = make([]string, diff.LeftEnd-diff.LeftStart+1)
		copy(leftLines, c.diffLeftLines[diff.LeftStart:diff.LeftEnd+1])
	}

	// Replace in right
	newRight := []string{}
	newRight = append(newRight, c.diffRightLines[:diff.RightStart]...)
	newRight = append(newRight, leftLines...)
	if diff.RightEnd+1 < len(c.diffRightLines) {
		newRight = append(newRight, c.diffRightLines[diff.RightEnd+1:]...)
	}

	c.diffRightLines = newRight
	c.diffRightModified = true
	c.calculateDiff()
	c.setStatus("Copied left → right")
}

// copyDiffRightToLeft copies current difference from right to left
func (c *Commander) copyDiffRightToLeft() {
	if c.diffCurrentIdx < 0 || c.diffCurrentIdx >= len(c.diffDifferences) {
		c.setStatus("No difference selected")
		return
	}

	diff := c.diffDifferences[c.diffCurrentIdx]
	if diff.Type == "equal" {
		c.setStatus("No difference at current position")
		return
	}

	// Extract lines from right
	var rightLines []string
	if diff.RightStart <= diff.RightEnd && diff.RightEnd < len(c.diffRightLines) {
		rightLines = make([]string, diff.RightEnd-diff.RightStart+1)
		copy(rightLines, c.diffRightLines[diff.RightStart:diff.RightEnd+1])
	}

	// Replace in left
	newLeft := []string{}
	newLeft = append(newLeft, c.diffLeftLines[:diff.LeftStart]...)
	newLeft = append(newLeft, rightLines...)
	if diff.LeftEnd+1 < len(c.diffLeftLines) {
		newLeft = append(newLeft, c.diffLeftLines[diff.LeftEnd+1:]...)
	}

	c.diffLeftLines = newLeft
	c.diffLeftModified = true
	c.calculateDiff()
	c.setStatus("Copied right → left")
}

// enterDiffEditMode enters edit mode for the active side
func (c *Commander) enterDiffEditMode() {
	c.diffEditMode = true
	c.diffCursorX = 0
	c.diffCursorY = c.diffScrollY
	if c.diffCursorY < 0 {
		c.diffCursorY = 0
	}
	lines := c.diffLeftLines
	if c.diffActiveSide == 1 {
		lines = c.diffRightLines
	}
	if c.diffCursorY >= len(lines) {
		c.diffCursorY = len(lines) - 1
	}
	c.setStatus("Edit mode: ESC to exit, changes auto-saved")
}

// saveDiffFiles saves modified files
func (c *Commander) saveDiffFiles() {
	savedCount := 0

	if c.diffLeftModified {
		content := strings.Join(c.diffLeftLines, "\n") + "\n"
		err := os.WriteFile(c.diffLeftPath, []byte(content), 0644)
		if err != nil {
			c.setStatus("Error saving left file: " + err.Error())
			return
		}
		c.diffLeftModified = false
		savedCount++
	}

	if c.diffRightModified {
		content := strings.Join(c.diffRightLines, "\n") + "\n"
		err := os.WriteFile(c.diffRightPath, []byte(content), 0644)
		if err != nil {
			c.setStatus("Error saving right file: " + err.Error())
			return
		}
		c.diffRightModified = false
		savedCount++
	}

	if savedCount == 0 {
		c.setStatus("No changes to save")
	} else if savedCount == 1 {
		c.setStatus("Saved 1 file")
	} else {
		c.setStatus("Saved both files")
	}
}

// exitDiffMode exits diff mode with unsaved changes warning
func (c *Commander) exitDiffMode() bool {
	if c.diffLeftModified || c.diffRightModified {
		// Simple warning - in a real implementation, would use a dialog
		c.setStatus("Unsaved changes! Press Ctrl+S to save, ESC again to discard")
		if !c.diffLeftModified && !c.diffRightModified {
			// Second press - actually exit
			c.diffMode = false
			c.diffLeftLines = nil
			c.diffRightLines = nil
			c.diffDifferences = nil
			c.setStatus("Diff mode exited")
			c.refreshPane(c.leftPane)
			c.refreshPane(c.rightPane)
			return false
		}
		// First press with unsaved changes - clear flags so second press exits
		c.diffLeftModified = false
		c.diffRightModified = false
		return false
	}

	// No unsaved changes, exit immediately
	c.diffMode = false
	c.diffLeftLines = nil
	c.diffRightLines = nil
	c.diffDifferences = nil
	c.setStatus("Diff mode exited")
	c.refreshPane(c.leftPane)
	c.refreshPane(c.rightPane)
	return false
}

// enterCompareMode initializes folder comparison mode
func (c *Commander) enterCompareMode() {
	// Initialize compare results map
	c.compareResults = make(map[string]CompareStatus)

	// Get files from both panes (excluding "..")
	leftFiles := make(map[string]*FileItem)
	for i := range c.leftPane.Files {
		if c.leftPane.Files[i].Name != ".." {
			leftFiles[c.leftPane.Files[i].Name] = &c.leftPane.Files[i]
		}
	}

	rightFiles := make(map[string]*FileItem)
	for i := range c.rightPane.Files {
		if c.rightPane.Files[i].Name != ".." {
			rightFiles[c.rightPane.Files[i].Name] = &c.rightPane.Files[i]
		}
	}

	// Compare files
	leftOnly := 0
	rightOnly := 0
	different := 0
	identical := 0

	// Check files in left pane
	for name, leftFile := range leftFiles {
		if rightFile, exists := rightFiles[name]; exists {
			// File exists in both panes
			if leftFile.IsDir && rightFile.IsDir {
				// Both are directories - consider identical by name only
				c.compareResults[name] = CompareStatus{
					Status:    "identical",
					LeftFile:  leftFile,
					RightFile: rightFile,
				}
				identical++
			} else if !leftFile.IsDir && !rightFile.IsDir {
				// Both are files - compare by size and modification time
				if leftFile.Size == rightFile.Size && leftFile.ModTime.Equal(rightFile.ModTime) {
					c.compareResults[name] = CompareStatus{
						Status:    "identical",
						LeftFile:  leftFile,
						RightFile: rightFile,
					}
					identical++
				} else {
					c.compareResults[name] = CompareStatus{
						Status:    "different",
						LeftFile:  leftFile,
						RightFile: rightFile,
					}
					different++
				}
			} else {
				// One is file, one is directory - different
				c.compareResults[name] = CompareStatus{
					Status:    "different",
					LeftFile:  leftFile,
					RightFile: rightFile,
				}
				different++
			}
		} else {
			// File exists only in left pane
			c.compareResults[name] = CompareStatus{
				Status:   "left_only",
				LeftFile: leftFile,
			}
			leftOnly++
		}
	}

	// Check files in right pane that don't exist in left
	for name, rightFile := range rightFiles {
		if _, exists := leftFiles[name]; !exists {
			c.compareResults[name] = CompareStatus{
				Status:    "right_only",
				RightFile: rightFile,
			}
			rightOnly++
		}
	}

	// Set compare mode flag
	c.compareMode = true

	// Display statistics
	totalFiles := len(c.compareResults)
	c.setStatus(fmt.Sprintf("Compare: %d files | Left only: %d | Right only: %d | Different: %d | Identical: %d",
		totalFiles, leftOnly, rightOnly, different, identical))
}

// exitCompareMode cleans up and exits comparison mode
func (c *Commander) exitCompareMode() {
	c.compareMode = false
	c.compareResults = nil
	c.setStatus("Compare mode exited")
	c.refreshPane(c.leftPane)
	c.refreshPane(c.rightPane)
}

// syncLeftToRight copies selected file(s) from left to right pane
func (c *Commander) syncLeftToRight() {
	if !c.compareMode {
		c.setStatus("Not in compare mode")
		return
	}

	// Collect files to sync
	var filesToSync []FileItem
	for i := range c.leftPane.Files {
		file := &c.leftPane.Files[i]
		if file.Name == ".." {
			continue
		}
		if file.Selected {
			// Check if file can be synced
			if status, exists := c.compareResults[file.Name]; exists {
				if status.Status == "left_only" || status.Status == "different" {
					filesToSync = append(filesToSync, *file)
				}
			}
		}
	}

	// If nothing selected, use current file
	if len(filesToSync) == 0 && c.activePane == PaneLeft && len(c.leftPane.Files) > 0 {
		file := c.leftPane.Files[c.leftPane.SelectedIdx]
		if file.Name != ".." {
			if status, exists := c.compareResults[file.Name]; exists {
				if status.Status == "left_only" || status.Status == "different" {
					filesToSync = append(filesToSync, file)
				}
			}
		}
	}

	if len(filesToSync) == 0 {
		c.setStatus("No files to sync (select left_only or different files)")
		return
	}

	// Copy files
	copiedCount := 0
	var lastErr error
	for _, file := range filesToSync {
		destPath := filepath.Join(c.rightPane.CurrentPath, file.Name)
		err := copyFileOrDir(file.Path, destPath)
		if err != nil {
			lastErr = err
		} else {
			copiedCount++
		}
	}

	// Update status
	if lastErr != nil {
		c.setStatus(fmt.Sprintf("Synced %d file(s) left→right, last error: %s", copiedCount, lastErr.Error()))
	} else {
		c.setStatus(fmt.Sprintf("Synced %d file(s) left→right", copiedCount))
	}

	// Clear selections
	for i := range c.leftPane.Files {
		c.leftPane.Files[i].Selected = false
	}

	// Refresh and re-compare
	c.refreshPane(c.rightPane)
	c.enterCompareMode()
}

// syncRightToLeft copies selected file(s) from right to left pane
func (c *Commander) syncRightToLeft() {
	if !c.compareMode {
		c.setStatus("Not in compare mode")
		return
	}

	// Collect files to sync
	var filesToSync []FileItem
	for i := range c.rightPane.Files {
		file := &c.rightPane.Files[i]
		if file.Name == ".." {
			continue
		}
		if file.Selected {
			// Check if file can be synced
			if status, exists := c.compareResults[file.Name]; exists {
				if status.Status == "right_only" || status.Status == "different" {
					filesToSync = append(filesToSync, *file)
				}
			}
		}
	}

	// If nothing selected, use current file
	if len(filesToSync) == 0 && c.activePane == PaneRight && len(c.rightPane.Files) > 0 {
		file := c.rightPane.Files[c.rightPane.SelectedIdx]
		if file.Name != ".." {
			if status, exists := c.compareResults[file.Name]; exists {
				if status.Status == "right_only" || status.Status == "different" {
					filesToSync = append(filesToSync, file)
				}
			}
		}
	}

	if len(filesToSync) == 0 {
		c.setStatus("No files to sync (select right_only or different files)")
		return
	}

	// Copy files
	copiedCount := 0
	var lastErr error
	for _, file := range filesToSync {
		destPath := filepath.Join(c.leftPane.CurrentPath, file.Name)
		err := copyFileOrDir(file.Path, destPath)
		if err != nil {
			lastErr = err
		} else {
			copiedCount++
		}
	}

	// Update status
	if lastErr != nil {
		c.setStatus(fmt.Sprintf("Synced %d file(s) right→left, last error: %s", copiedCount, lastErr.Error()))
	} else {
		c.setStatus(fmt.Sprintf("Synced %d file(s) right→left", copiedCount))
	}

	// Clear selections
	for i := range c.rightPane.Files {
		c.rightPane.Files[i].Selected = false
	}

	// Refresh and re-compare
	c.refreshPane(c.leftPane)
	c.enterCompareMode()
}

// syncBothWays synchronizes bidirectionally
func (c *Commander) syncBothWays() {
	if !c.compareMode {
		c.setStatus("Not in compare mode")
		return
	}

	leftCopied := 0
	rightCopied := 0
	newerCopied := 0
	var lastErr error

	// Process all files in compare results
	for name, status := range c.compareResults {
		switch status.Status {
		case "left_only":
			// Copy from left to right
			destPath := filepath.Join(c.rightPane.CurrentPath, name)
			err := copyFileOrDir(status.LeftFile.Path, destPath)
			if err != nil {
				lastErr = err
			} else {
				leftCopied++
			}
		case "right_only":
			// Copy from right to left
			destPath := filepath.Join(c.leftPane.CurrentPath, name)
			err := copyFileOrDir(status.RightFile.Path, destPath)
			if err != nil {
				lastErr = err
			} else {
				rightCopied++
			}
		case "different":
			// Copy newer file to the other side
			if !status.LeftFile.IsDir && !status.RightFile.IsDir {
				if status.LeftFile.ModTime.After(status.RightFile.ModTime) {
					// Left is newer, copy to right
					destPath := filepath.Join(c.rightPane.CurrentPath, name)
					err := copyFileOrDir(status.LeftFile.Path, destPath)
					if err != nil {
						lastErr = err
					} else {
						newerCopied++
					}
				} else if status.RightFile.ModTime.After(status.LeftFile.ModTime) {
					// Right is newer, copy to left
					destPath := filepath.Join(c.leftPane.CurrentPath, name)
					err := copyFileOrDir(status.RightFile.Path, destPath)
					if err != nil {
						lastErr = err
					} else {
						newerCopied++
					}
				}
			}
		}
	}

	// Update status
	if lastErr != nil {
		c.setStatus(fmt.Sprintf("Synced both ways: %d left→right, %d right→left, %d newer copied | Error: %s",
			leftCopied, rightCopied, newerCopied, lastErr.Error()))
	} else {
		c.setStatus(fmt.Sprintf("Synced both ways: %d left→right, %d right→left, %d newer copied",
			leftCopied, rightCopied, newerCopied))
	}

	// Refresh both panes and re-compare
	c.refreshPane(c.leftPane)
	c.refreshPane(c.rightPane)
	c.enterCompareMode()
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
