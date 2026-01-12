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

### 5. Search (Ctrl+F)
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

### 6. File Editing (Ctrl+E)
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

## Cross-Platform Notes

### Linux/macOS
- Uses `$EDITOR` environment variable for file editing
- Defaults to nano or vi if not set
- Full Unicode support for file names

### Windows
- Works in Windows Terminal or Command Prompt
- Uses notepad as default editor
- Handles Windows-style paths (C:\Users\...)

## Tips and Tricks

1. **Quick Navigation**: Use `.` in path to refresh current directory
2. **Safe Operations**: Operations cannot be undone - be careful with F8 (delete)
3. **Large Directories**: Scroll through with arrow keys, use Ctrl+F to find files
4. **Dual Monitor Workflow**: Keep two related directories open simultaneously
5. **Batch Operations**: No need to leave the app - all operations are in-app
