# Terminal Commander - Feature Demonstration

## Visual Layout

```
┌─────────────────────────────────────────┬─────────────────────────────────────────┐
│ /home/user/documents                    │ /home/user/downloads                    │
├─────────────────────────────────────────┼─────────────────────────────────────────┤
│ [..]                                    │ [..]                                    │
│ [projects]                              │ [images]                                │
│ [notes]                                 │ [videos]                                │
│ report.pdf                      1.2MB   │ movie.mp4                      450MB   │
│ budget.xlsx                      45KB   │ photo.jpg                      2.3MB   │
│ letter.docx                      23KB   │ song.mp3                       4.5MB   │
│                                         │                                         │
└─────────────────────────────────────────┴─────────────────────────────────────────┘
F5:Copy F6:Move F8:Del Ctrl+F:Search Ctrl+E:Edit Ctrl+N:NewDir Tab:Switch ESC:Quit
```

## Key Features Demonstrated

### 1. Dual-Pane Navigation
- **Left Pane**: Shows `/home/user/documents`
- **Right Pane**: Shows `/home/user/downloads`
- **TAB key**: Switches between panes
- **Active pane**: Highlighted with different background color

### 2. File/Directory Display
- **Directories**: Shown in brackets, e.g., `[projects]`
- **Files**: Shown with size information
- **Parent directory**: Always shown as `[..]` at the top
- **Sorting**: Directories first, then files, alphabetically

### 3. Navigation Controls
- **Up/Down arrows**: Move selection
- **Enter**: Enter directory or select file
- **Backspace**: Go to parent directory

### 4. File Operations

#### Copy (F5 or Ctrl+C)
```
1. Select file in left pane
2. Press F5
3. File is copied to the right pane's directory
Status bar shows: "Copied: filename.txt"
```

#### Move (F6 or Ctrl+X)
```
1. Select file in left pane
2. Press F6
3. File is moved to the right pane's directory
Status bar shows: "Moved: filename.txt"
```

#### Delete (F8 or Ctrl+D)
```
1. Select file
2. Press F8
3. File is deleted (directories are recursively deleted)
Status bar shows: "Deleted: filename.txt"
```

#### Rename (Ctrl+R)
```
Left Pane: /home/user/documents          │ Right Pane: /home/user/downloads
───────────────────────────────────────────────────────────────────────────
report.pdf <-- selected                  │
───────────────────────────────────────────────────────────────────────────
Rename to: report_final.pdf_            │ (input mode active)
```

1. Select file
2. Press Ctrl+R
3. Current name appears in input field
4. Type new name
5. Press Enter to confirm or ESC to cancel

#### Create Directory (Ctrl+N)
```
Left Pane: /home/user/documents          │ Right Pane: /home/user/downloads
───────────────────────────────────────────────────────────────────────────
[projects]                               │
[notes]                                  │
───────────────────────────────────────────────────────────────────────────
New directory name: archive_            │ (input mode active)
```

1. Press Ctrl+N
2. Enter directory name
3. Press Enter to create or ESC to cancel

### 5. Multi-File Selection (Spacebar)
```
Left Pane: /home/user/documents          │ Right Pane: /home/user/downloads
───────────────────────────────────────────────────────────────────────────
[projects]                               │
[*] report.pdf <-- selected              │
[*] budget.xlsx <-- selected             │
presentation.pptx <-- highlighted        │
───────────────────────────────────────────────────────────────────────────
Selected: presentation.pptx               │ (selection mode active)
```

1. Navigate to a file with arrow keys
2. Press Spacebar to toggle selection (shows `[*]` marker)
3. Move to next file and press Spacebar again
4. Continue selecting multiple files
5. Selected items remain marked while navigating

**Note**: Operations (copy, move, delete, archive) will work on all selected items if any are selected, otherwise on the currently highlighted item.

### 6. Archive Compression (Ctrl+A)

#### Single File Archive
```
Left Pane: /home/user/documents          │ Right Pane: /home/user/downloads
───────────────────────────────────────────────────────────────────────────
report.pdf <-- highlighted               │
budget.xlsx                              │
───────────────────────────────────────────────────────────────────────────
Press Ctrl+A → Archive format selection appears
```

Archive Format Selection:
```
┌──────────────────────────────────────────────────────────────┐
│ Select Archive Format for: report.pdf                        │
├──────────────────────────────────────────────────────────────┤
│  .zip       <-- selected                                      │
│  .tar                                                         │
│  .tar.gz                                                      │
│  .tar.bz2                                                     │
│  .tar.xz                                                      │
└──────────────────────────────────────────────────────────────┘
Select archive format. Enter:Create, Esc:Cancel
```

Press Enter → Creates `report.zip` in the current directory

#### Multiple Files Archive
```
Left Pane: /home/user/documents          │ Right Pane: /home/user/downloads
───────────────────────────────────────────────────────────────────────────
[*] report.pdf <-- selected              │
[*] budget.xlsx <-- selected             │
[*] presentation.pptx <-- selected       │
───────────────────────────────────────────────────────────────────────────
Selected: presentation.pptx               │
```

Press Ctrl+A → Archive format selection appears:
```
┌──────────────────────────────────────────────────────────────┐
│ Select Archive Format (3 file(s) selected)                   │
├──────────────────────────────────────────────────────────────┤
│  .zip                                                         │
│  .tar.gz    <-- selected                                      │
│  .tar.bz2                                                     │
└──────────────────────────────────────────────────────────────┘
```

Press Enter → Creates `archive_20240115_143022.tar.gz` with all selected files

**Available formats** (detected automatically):
- `.zip` - ZIP compression (requires `zip` command)
- `.7z` - 7-Zip compression (requires `7z` or `7za` command)
- `.tar` - TAR archive without compression
- `.tar.gz` - TAR with gzip compression
- `.tar.bz2` - TAR with bzip2 compression
- `.tar.xz` - TAR with xz compression

### 7. Search (Ctrl+F)
```
Left Pane: /home/user/documents          │ Right Pane: /home/user/downloads
───────────────────────────────────────────────────────────────────────────
[projects]                               │
report.pdf <-- found and selected        │
budget.xlsx                              │
───────────────────────────────────────────────────────────────────────────
Search: report_                          │ (search mode active)
```

1. Press Ctrl+F
2. Type search term
3. Press Enter to find first match
4. Status shows "Found: filename" or "Not found"

### 8. File Editing (Ctrl+E)
```
1. Select a text file
2. Press Ctrl+E
3. External editor opens (respects $EDITOR environment variable)
   - Linux: nano, vi, or $EDITOR
   - Windows: notepad
4. After editing, return to Terminal Commander
```

## Status Bar

The status bar at the bottom shows:
- **Default**: Available keyboard shortcuts
- **Operations**: Confirmation messages (e.g., "Copied: file.txt")
- **Errors**: Error messages (e.g., "Error copying: permission denied")
- **Input mode**: Current prompt and input buffer

## Practical Usage Examples

### Example 1: Organizing Photos
```
Left: /home/user/downloads      Right: /home/user/Pictures/2024
1. Navigate to downloads (left)
2. Navigate to Pictures/2024 (right, TAB to switch, arrows to navigate)
3. Select photo in left pane
4. Press F5 to copy to right pane
5. Repeat for all photos
```

### Example 2: Code Project Management
```
Left: /home/user/project/src    Right: /home/user/project/backup
1. Select file in src
2. Press Ctrl+E to edit
3. After changes, press F5 to copy to backup
4. Use Ctrl+N to create new directory for organization
```

### Example 3: File Cleanup
```
Left: /home/user/temp           Right: /home/user/archive
1. Review files in temp (left pane)
2. Press F6 to move important files to archive
3. Press F8 to delete unwanted files
4. Use Ctrl+F to search for specific files
```

### Example 4: Creating Project Backups with Archives
```
Left: /home/user/project/src    Right: /home/user/project
1. Navigate to src directory (left pane)
2. Press Spacebar on each important file to select them
   [*] main.go
   [*] utils.go
   [*] config.json
3. Press Ctrl+A to start archive creation
4. Select .tar.gz format
5. Press Enter → Creates archive_20240115_143022.tar.gz in src directory
6. Press Tab to switch to right pane
7. Navigate back to parent (Backspace)
8. Press F6 to move the archive to project directory
```

### Example 5: Batch Archive Multiple Folders
```
Left: /home/user/documents
1. Navigate to documents directory
2. Select multiple folders:
   Press Spacebar on [photos]
   Press Spacebar on [videos]
   Press Spacebar on [documents]
3. Press Ctrl+A
4. Select .zip format
5. Press Enter → Creates archive_20240115_143022.zip containing all three folders
```

## Cross-Platform Notes

### Linux/macOS
- Uses `$EDITOR` environment variable for file editing
- Defaults to nano or vi if not set
- Full Unicode support for file names
- Archive tools: Install via package manager
  - Ubuntu/Debian: `sudo apt install zip tar p7zip-full`
  - macOS: `brew install p7zip` (tar and zip are pre-installed)

### Windows
- Works in Windows Terminal or Command Prompt
- Uses notepad as default editor
- Handles Windows-style paths (C:\Users\...)
- Archive tools: 
  - Windows 10+ has tar built-in
  - Install 7-Zip for .7z and .zip support

## Tips and Tricks

1. **Quick Navigation**: Use `.` in path to refresh current directory
2. **Safe Operations**: Operations cannot be undone - be careful with F8 (delete)
3. **Large Directories**: Scroll through with arrow keys, use Ctrl+F to find files
4. **Dual Monitor Workflow**: Keep two related directories open simultaneously
5. **Batch Operations**: No need to leave the app - all operations are in-app
