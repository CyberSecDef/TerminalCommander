package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
)

const (
	PaneLeft = iota
	PaneRight
)

type FileItem struct {
	Name  string
	IsDir bool
	Size  int64
	Path  string
}

type Pane struct {
	CurrentPath  string
	Files        []FileItem
	SelectedIdx  int
	ScrollOffset int
	Width        int
	Height       int
}

type Commander struct {
	screen      tcell.Screen
	leftPane    *Pane
	rightPane   *Pane
	activePane  int
	statusMsg   string
	searchMode  bool
	searchQuery string
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
	}

	return false
}

func (c *Commander) handleSearchKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		c.searchMode = false
		c.searchQuery = ""
		c.statusMsg = ""
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
	c.statusMsg = "Search: " + c.searchQuery
	return false
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
	if pane.SelectedIdx >= pane.ScrollOffset+pane.Height-3 {
		pane.ScrollOffset = pane.SelectedIdx - pane.Height + 4
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
		c.statusMsg = "Entered: " + selected.Name
	} else {
		c.statusMsg = "Use Ctrl+E to edit file"
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
		c.statusMsg = "Parent directory"
	}
}

func (c *Commander) startSearch() {
	c.searchMode = true
	c.searchQuery = ""
	c.statusMsg = "Search: "
}

func (c *Commander) performSearch() {
	pane := c.getActivePane()
	query := strings.ToLower(c.searchQuery)
	
	for i, file := range pane.Files {
		if strings.Contains(strings.ToLower(file.Name), query) {
			pane.SelectedIdx = i
			pane.ScrollOffset = 0
			if pane.SelectedIdx >= pane.Height-3 {
				pane.ScrollOffset = pane.SelectedIdx - pane.Height + 4
			}
			c.statusMsg = "Found: " + file.Name
			c.searchQuery = ""
			return
		}
	}
	c.statusMsg = "Not found: " + c.searchQuery
	c.searchQuery = ""
}

func (c *Commander) copyFile() {
	pane := c.getActivePane()
	destPane := c.getInactivePane()
	
	if len(pane.Files) == 0 {
		c.statusMsg = "No file selected"
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.statusMsg = "Cannot copy parent directory link"
		return
	}

	destPath := filepath.Join(destPane.CurrentPath, selected.Name)
	
	err := copyFileOrDir(selected.Path, destPath)
	if err != nil {
		c.statusMsg = "Error copying: " + err.Error()
	} else {
		c.statusMsg = "Copied: " + selected.Name
		c.refreshPane(destPane)
	}
}

func (c *Commander) moveFile() {
	pane := c.getActivePane()
	destPane := c.getInactivePane()
	
	if len(pane.Files) == 0 {
		c.statusMsg = "No file selected"
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.statusMsg = "Cannot move parent directory link"
		return
	}

	destPath := filepath.Join(destPane.CurrentPath, selected.Name)
	
	err := os.Rename(selected.Path, destPath)
	if err != nil {
		c.statusMsg = "Error moving: " + err.Error()
	} else {
		c.statusMsg = "Moved: " + selected.Name
		c.refreshPane(pane)
		c.refreshPane(destPane)
	}
}

func (c *Commander) deleteFile() {
	pane := c.getActivePane()
	
	if len(pane.Files) == 0 {
		c.statusMsg = "No file selected"
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.statusMsg = "Cannot delete parent directory link"
		return
	}

	var err error
	if selected.IsDir {
		err = os.RemoveAll(selected.Path)
	} else {
		err = os.Remove(selected.Path)
	}

	if err != nil {
		c.statusMsg = "Error deleting: " + err.Error()
	} else {
		c.statusMsg = "Deleted: " + selected.Name
		if pane.SelectedIdx > 0 {
			pane.SelectedIdx--
		}
		c.refreshPane(pane)
	}
}

func (c *Commander) renameFile() {
	pane := c.getActivePane()
	
	if len(pane.Files) == 0 {
		c.statusMsg = "No file selected"
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.Name == ".." {
		c.statusMsg = "Cannot rename parent directory link"
		return
	}

	c.statusMsg = "Rename not yet implemented in TUI mode - use Ctrl+R externally"
}

func (c *Commander) editFile() {
	pane := c.getActivePane()
	
	if len(pane.Files) == 0 {
		c.statusMsg = "No file selected"
		return
	}

	selected := pane.Files[pane.SelectedIdx]
	if selected.IsDir {
		c.statusMsg = "Cannot edit a directory"
		return
	}

	// Suspend screen to launch external editor
	c.screen.Fini()
	
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if _, err := os.Stat("/usr/bin/nano"); err == nil {
			editor = "nano"
		} else if _, err := os.Stat("/usr/bin/vi"); err == nil {
			editor = "vi"
		} else {
			editor = "notepad" // Windows fallback
		}
	}

	cmd := fmt.Sprintf("%s %s", editor, selected.Path)
	c.statusMsg = "Launching editor: " + cmd
	
	// Re-initialize screen
	c.screen.Init()
	c.screen.Clear()
}

func (c *Commander) createDirectory() {
	c.statusMsg = "Create directory not yet implemented in TUI mode"
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

		item := FileItem{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Path:  filepath.Join(pane.CurrentPath, entry.Name()),
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
	
	// Draw files
	visibleStart := pane.ScrollOffset
	visibleEnd := pane.ScrollOffset + pane.Height - 3
	if visibleEnd > len(pane.Files) {
		visibleEnd = len(pane.Files)
	}

	for i := visibleStart; i < visibleEnd; i++ {
		file := pane.Files[i]
		y := i - pane.ScrollOffset + 1
		
		itemStyle := style
		if i == pane.SelectedIdx {
			if active {
				itemStyle = tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite)
			} else {
				itemStyle = tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorWhite)
			}
		}
		
		display := file.Name
		if file.IsDir {
			display = "[" + display + "]"
		}
		
		if len(display) > pane.Width-10 {
			display = display[:pane.Width-13] + "..."
		}
		
		sizeStr := ""
		if !file.IsDir && file.Name != ".." {
			sizeStr = formatSize(file.Size)
		}
		
		c.drawText(offsetX, y, pane.Width, itemStyle, fmt.Sprintf(" %-*s %8s", pane.Width-10, display, sizeStr))
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
	style := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite)
	
	statusText := c.statusMsg
	if statusText == "" {
		statusText = "F5:Copy F6:Move F8:Del Ctrl+F:Search Ctrl+E:Edit Ctrl+N:NewDir Tab:Switch ESC:Quit"
	}
	
	if len(statusText) > width {
		statusText = statusText[:width]
	}
	
	c.drawText(0, y, width, style, statusText)
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
