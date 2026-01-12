# TerminalCommander

A cross-platform (Windows and Linux) dual-pane console-based file explorer inspired by Total Commander, written in Go.

## Features

- **Dual-Pane Interface**: Navigate two directories simultaneously
- **Keyboard Navigation**: Arrow keys for file selection, TAB to switch between panes
- **File Operations**:
  - Copy files/directories (F5 or Ctrl+C)
  - Move files/directories (F6 or Ctrl+X)
  - Delete files/directories (F8 or Ctrl+D)
  - Rename files (Ctrl+R)
  - Edit files (Ctrl+E)
  - Create directories (Ctrl+N)
- **Search**: Ctrl+F to search for files in current directory
- **Visual Indicators**: 
  - Directories shown in brackets [dirname]
  - File sizes displayed
  - Active pane highlighted
  - Current path shown at top of each pane

## Installation

### Prerequisites
- Go 1.20 or later

### Build from Source

Clone the repository:
```bash
git clone https://github.com/CyberSecDef/TerminalCommander.git
cd TerminalCommander
```

#### Quick Build
```bash
make build
```

Or manually:
```bash
go build -o terminalcommander main.go
```

#### Build for All Platforms
```bash
make all
```

This creates:
- `terminalcommander-linux` - Linux binary
- `terminalcommander.exe` - Windows binary
- `terminalcommander-mac` - macOS binary

#### Platform-Specific Builds

For Linux:
```bash
make linux
```

For Windows:
```bash
make windows
```

For macOS:
```bash
make darwin
```

## Usage

Run the application:

```bash
./terminalcommander
```

On Windows:
```cmd
terminalcommander.exe
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ↑/↓ | Move selection up/down |
| Enter | Enter directory / Select file |
| Backspace | Go to parent directory |
| Tab | Switch between left and right pane |
| F5 / Ctrl+C | Copy selected file/directory to other pane |
| F6 / Ctrl+X | Move selected file/directory to other pane |
| F8 / Ctrl+D / Delete | Delete selected file/directory |
| Ctrl+R | Rename file/directory |
| Ctrl+E | Edit file with default editor |
| Ctrl+F | Search for file in current directory |
| Ctrl+N | Create new directory |
| Ctrl+Q / ESC | Quit application |

### Environment Variables

- `EDITOR`: Specify your preferred text editor for Ctrl+E (defaults to nano/vi on Linux, notepad on Windows)

## Cross-Platform Compatibility

TerminalCommander uses the `tcell` library which provides excellent cross-platform terminal handling for:
- Linux
- macOS  
- Windows (via Windows Console or Windows Terminal)

## Development

### Project Structure

```
TerminalCommander/
├── main.go           # Main application code
├── go.mod            # Go module definition
├── go.sum            # Go module checksums
└── README.md         # This file
```

### Building for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o terminalcommander-linux main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o terminalcommander.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o terminalcommander-mac main.go
```

## License

MIT License - feel free to use and modify as needed.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
