# Terminal Commander - Quick Start Guide

## Installation

```bash
# Clone the repository
git clone https://github.com/CyberSecDef/TerminalCommander.git
cd TerminalCommander

# Build
make build

# Run
./terminalcommander
```

## First Launch

When you first start Terminal Commander, you'll see a dual-pane interface:

```
┌─────────────────────────────────────────┬─────────────────────────────────────────┐
│ /home/user/documents                    │ /home/user/documents                    │
├─────────────────────────────────────────┼─────────────────────────────────────────┤
│ [..]                                    │ [..]                                    │
│ [folder1]                               │ [folder1]                               │
│ [folder2]                               │ [folder2]                               │
│ file1.txt                         123B  │ file1.txt                         123B  │
│ file2.pdf                        1.2MB  │ file2.pdf                        1.2MB  │
│                                         │                                         │
└─────────────────────────────────────────┴─────────────────────────────────────────┘
c/C:Copy m/M:Move Del:Delete s/S:Search e/E:Edit n/N:NewDir ?:Help Tab:Switch ESC:Quit
```

The **left pane** is active (brighter colors), indicated by the blue header.

## Quick Tutorial

### 1. Navigate Files
```
Press ↓ (Down Arrow) → Move to next file
Press ↑ (Up Arrow)   → Move to previous file
Press Enter          → Enter the selected directory
Press Backspace      → Go to parent directory
```

### 2. Switch Panes
```
Press Tab            → Switch to the other pane
```

The active pane will have a brighter blue header.

### 3. Copy a File
```
1. Select file in left pane (use arrow keys)
2. Press Tab to switch to right pane
3. Navigate to destination directory (use Enter to go into folders)
4. Press Tab to go back to left pane
5. Press c/C (copy)
```

Status bar will show: `Copied: filename.txt`

### 4. Move a File
```
1. Select file to move
2. Navigate to destination in other pane
3. Press m/M (move)
```

File will be moved (deleted from source, copied to destination).

### 5. Delete a File
```
1. Select file
2. Press Delete
```

**Warning**: Deletion is permanent! Directories are deleted recursively.

### 6. Rename a File
```
1. Select file
2. Press r/R (rename)
3. Edit the name in the prompt
4. Press Enter to confirm or ESC to cancel
```

Status bar shows: `Rename to: newname.txt_` (cursor blinks at end)

### 7. Create a Directory
```
1. Press n/N (new directory)
2. Type directory name
3. Press Enter to create or ESC to cancel
```

New directory appears in the current pane.

### 8. Create a Blank File
```
1. Press b/B (blank file)
2. Type file name
3. Press Enter to create or ESC to cancel
```

Empty file appears in the current pane.

### 9. Select Multiple Files
```
1. Navigate to a file with arrow keys
2. Press Spacebar to toggle selection (shows [*] marker)
3. Move to another file and press Spacebar again
4. Repeat for all files you want to select
```

Selected files will show `[*]` before their name and remain selected while you navigate.

### 10. Create an Archive
```
1. Select files with Spacebar (or just highlight one file)
2. Press a/A (archive)
3. Choose archive format with arrow keys (.zip, .tar.gz, etc.)
4. Press Enter
```

Archive will be created in the current directory:
- Single file: Uses the file's name (e.g., `report.zip`)
- Multiple files: Uses timestamp (e.g., `archive_20240115_143022.zip`)

After archiving, selections are cleared automatically.

### 11. Search for a File
```
1. Press s/S (search)
2. Type part of filename
3. Press Enter
```

First matching file will be highlighted.

### 12. Edit a File
```
1. Select a text file
2. Press e/E (edit)
```

External editor opens:
- Linux/macOS: Uses $EDITOR (or nano/vi)
- Windows: Opens notepad

After saving and closing editor, you return to Terminal Commander.

### 13. Get Help
```
1. Press ? (help)
```

A comprehensive help pane shows all available keyboard shortcuts and functions. Press any key to close.
```
Press ESC or Ctrl+Q → Exit Terminal Commander
```

## Common Tasks

### Organizing Photos
```
Left Pane: /Downloads    Right Pane: /Photos/2024
1. Navigate Downloads → find photo
2. Tab to switch → navigate to Photos/2024
3. Tab back → press c/C to copy
4. Repeat for all photos
```

### Backup Important Files
```
Left Pane: /Documents    Right Pane: /Backup
1. Select file in Documents
2. Press c/C to copy to Backup
```

### Clean Up Downloads
```
Left Pane: /Downloads    Right Pane: /Organized
1. Press m/M to move files worth keeping to Organized
2. Press Delete to delete files you don't need
```

### Archive Project Files
```
Left Pane: /project/src
1. Press Spacebar on each important file to select them
2. Press a/A to create archive
3. Select .tar.gz format with arrow keys
4. Press Enter → Creates timestamped archive
5. Use m/M to move archive to backup location
```

### Batch Operations on Multiple Files
```
Left Pane: /photos    Right Pane: /vacation_album
1. Select multiple photos with Spacebar (shows [*] marker)
2. Press c/C to copy all selected files to right pane
   OR
   Press a/A to create a single archive of all photos
```

## Keyboard Reference Card

| Action | Key |
|--------|-----|
| Move Up | ↑ |
| Move Down | ↓ |
| Enter Directory | Enter |
| Parent Directory | Backspace |
| Select/Deselect Item | Spacebar |
| Switch Pane | Tab |
| Copy | c/C |
| Move | m/M |
| Delete | Delete |
| Create Archive | a/A |
| Rename | r/R |
| Edit | e/E |
| New Directory | n/N |
| New Blank File | b/B |
| Search | s/S |
| Go to Folder | g/G |
| Hash File | h/H |
| Diff Mode | f/F |
| Compare Mode | y/Y |
| Help | ? |
| Quit | ESC or Ctrl+Q |

## Tips

1. **Use Tab frequently** - Switch between panes to set up your source and destination
2. **Parent directory (..)** - Always shown at top, use Enter on it to go up
3. **File sizes** - Shown on right side in human-readable format (B, KB, MB, GB)
4. **Directories in brackets** - Easy to distinguish `[dirname]` from files
5. **Selected items** - Marked with `[*]` prefix, selections persist while navigating
6. **Status bar** - Watch for confirmation messages after operations
7. **Search is fast** - Use s/S instead of scrolling through long lists
8. **Delete is permanent** - Be careful with Delete, there's no undo
9. **Archive formats** - Only formats with available tools on your system will be shown
10. **Multi-select operations** - Works with copy, move, delete, and archive creation
11. **Help is available** - Press ? anytime to see all keyboard shortcuts

## Troubleshooting

### "Permission denied" errors
- Make sure you have read/write permissions for the directories
- On Linux, you may need `sudo` for system directories

### "No archive tools available" message
- Install archive tools based on your system:
  - Ubuntu/Debian: `sudo apt install zip tar p7zip-full`
  - macOS: `brew install p7zip` (tar and zip are pre-installed)
  - Windows: Install 7-Zip or use built-in tar

### Editor doesn't open
- Set EDITOR environment variable: `export EDITOR=nano`
- Or use the default (nano/vi on Linux, notepad on Windows)

### Display issues
- Make sure terminal supports color
- Try resizing terminal window
- Use a modern terminal emulator (Windows Terminal, iTerm2, etc.)

## More Information

- See **FEATURES.md** for detailed feature descriptions with examples
- See **DEVELOPMENT.md** for technical implementation details
- See **README.md** for installation and build instructions
