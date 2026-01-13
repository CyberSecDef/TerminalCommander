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

### Example 6: File Comparison and Merging with Diff Mode
```
Left: /home/user/project/version1    Right: /home/user/project/version2
1. Navigate to version1 directory (left pane)
2. Select config.yaml file
3. Navigate to version2 directory (right pane, Tab to switch)
4. Select config.yaml file
5. Press F3 to enter diff mode

Diff View:
┌─────────────────────────────────────────────────────────────────────────────┐
│ Left: config.yaml                       │ Right: config.yaml                │
├─────────────────────────────────────────────────────────────────────────────┤
│    1 server:                            │    1 server:                      │
│    2   port: 8080                       │    2   port: 8080                 │
│    3   host: localhost                  │    3   host: 0.0.0.0              │ (yellow)
│    4 database:                          │    4 database:                    │
│    5   enabled: false                   │    5   enabled: true              │ (yellow)
│    6                                    │    6   connection: postgres://... │ (green)
└─────────────────────────────────────────────────────────────────────────────┘
F3/ESC:Exit n:Next p:Prev >:Copy→ <:Copy← e:Edit Ctrl+S:Save | 2 differences

6. Press 'n' to jump to next difference (line 3: host change)
7. Press '>' to copy "host: 0.0.0.0" from right to left
8. Press 'n' again to jump to next difference (line 5-6: database config)
9. Press '>' to copy database changes from right to left
10. Press Ctrl+S to save changes to left file
11. Press F3 to exit diff mode

Result: version1/config.yaml now has the updated configuration from version2
```

**Diff Mode Features:**
- **Color Coding:**
  - Red background: Lines deleted (only in left)
  - Green background: Lines added (only in right)
  - Yellow background: Lines modified (different in both)
- **Navigation:**
  - Arrow keys for manual scrolling
  - 'n' to jump to next difference
  - 'p' to jump to previous difference
- **Merging:**
  - '>' copies current difference left → right
  - '<' copies current difference right → left
- **Editing:**
  - Press 'e' to enter edit mode
  - Make manual changes with arrow keys and typing
  - ESC to exit edit mode
- **Saving:**
  - Ctrl+S saves both modified files
  - Modified files show [modified] indicator in header

### Example 7: Quick Code Review with Diff
```
Scenario: Review changes made to a source file
Left: /home/user/backup/main.go     Right: /home/user/project/main.go

1. Navigate to backup folder (left)
2. Navigate to current project folder (right)
3. Select main.go in both panes
4. Press F3 to see what changed
5. Review each difference with 'n' key
6. Decide whether to keep, discard, or merge changes
7. Use '<' to revert unwanted changes (copy from backup to current)
8. Use '>' to update backup with good changes
9. Save with Ctrl+S
```

### Example 8: Folder Synchronization with Comparison Mode
```
Scenario: Synchronize two project directories
Left: /home/user/project/dev     Right: /home/user/project/backup

1. Navigate to dev directory (left pane)
2. Navigate to backup directory (right pane, Tab to switch)
3. Press Ctrl+Y to enter comparison mode

Comparison View:
┌──────────────────────────────────────────┬──────────────────────────────────────────┐
│ /home/user/project/dev                   │ /home/user/project/backup                 │
├──────────────────────────────────────────┼──────────────────────────────────────────┤
│ [L] new_feature.go         (cyan)        │                                           │
│ [D] config.yaml            (yellow)      │ [D] config.yaml            (yellow)      │
│ [=] main.go                (green)       │ [=] main.go                (green)       │
│                                          │ [R] old_backup.log         (cyan)        │
└──────────────────────────────────────────┴──────────────────────────────────────────┘
Compare: 4 files | Left only: 1 | Right only: 1 | Different: 1 | Identical: 1

Understanding the indicators:
• [L] new_feature.go - Exists only in left (dev) - cyan color
• [R] old_backup.log - Exists only in right (backup) - cyan color  
• [D] config.yaml - Exists in both but different size/time - yellow color
• [=] main.go - Identical in both folders - green color

Synchronization options:

Option 1: Sync specific file left→right
4. Select new_feature.go (marked [L])
5. Press '>' to copy it to backup
6. Status: "Synced 1 file(s) left→right"
7. Comparison automatically updates

Option 2: Sync specific file right→left  
4. Press Tab to switch to right pane
5. Select old_backup.log (marked [R])
6. Press '<' to copy it to dev
7. Status: "Synced 1 file(s) right→left"

Option 3: Full bidirectional sync
4. Press '=' to sync everything
5. This will:
   - Copy new_feature.go from dev to backup
   - Copy old_backup.log from backup to dev
   - For config.yaml: copy the newer version based on modification time
6. Status: "Synced both ways: 1 left→right, 1 right→left, 1 newer copied"

After sync:
7. Press Ctrl+Y to exit comparison mode
8. Both folders now synchronized

Real-world use cases:
• Synchronize development and backup directories
• Compare local and network drive folders
• Identify missing files between two locations
• Keep two project versions in sync
```

### Example 9: Selective File Synchronization
```
Scenario: Sync only certain files between folders
Left: /home/user/docs/current     Right: /home/user/docs/archive

1. Navigate to current directory (left)
2. Navigate to archive directory (right)
3. Press Ctrl+Y to enter comparison mode
4. Select multiple files to sync using Spacebar:
   - Press Spacebar on file1.txt [L] to select it
   - Press Spacebar on file2.txt [L] to select it
   - Press Spacebar on file3.txt [D] to select it
5. Press '>' to sync only selected files left→right
6. Status: "Synced 3 file(s) left→right"
7. Unselected files remain untouched
8. Press Ctrl+Y to exit

This allows fine-grained control over which files to synchronize.
```

6. **File Comparison**: Use F3 to quickly compare and merge changes between files
7. **Version Control**: Keep backups in one pane and working files in another, use diff to review changes
8. **Folder Synchronization**: Use Ctrl+Y to compare and sync entire folders efficiently
