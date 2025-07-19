# URL Navigator

A modern desktop application for managing website bookmarks and URL navigation, built with Go + Wails v2 + React + TypeScript.

## Features

- 🔖 **Bookmark Management**: Add, edit, delete and organize your favorite websites
- 📁 **Category System**: Organize bookmarks into custom categories with color coding
- 🔍 **Smart Search**: Quickly find bookmarks by title, URL, or tags
- 🚀 **Auto Update**: Automatic update functionality via GitHub Releases
- 🎨 **Modern UI**: Beautiful interface built with shadcn/ui components
- 💾 **Local Storage**: All data stored locally in JSON format

## Tech Stack

- **Backend**: Go 1.21+
- **Frontend**: React 18 + TypeScript + Vite
- **Desktop Framework**: Wails v2
- **UI Components**: shadcn/ui + Tailwind CSS
- **Auto Update**: MinIO selfupdate
- **Build Tools**: Yarn, Go modules

## Installation

### Prerequisites

- Go 1.21 or higher
- Node.js 18+ and Yarn
- Wails v2 CLI

### Development Setup

1. Clone the repository:
```bash
git clone https://github.com/YOUR_USERNAME/url-navigator.git
cd url-navigator
```

2. Install dependencies:
```bash
# Install Go dependencies
go mod download

# Install frontend dependencies
cd frontend
yarn install
cd ..
```

3. Run in development mode:
```bash
wails dev
```

### Building for Production

```bash
# Build for current platform
wails build

# Build with compression
wails build -compress

# Build for specific platform
wails build -platform windows/amd64
```

## Usage

### Managing Bookmarks

1. **Add Bookmark**: Click the "+" button to add a new bookmark
2. **Edit Bookmark**: Click the edit icon on any bookmark
3. **Delete Bookmark**: Click the trash icon to remove a bookmark
4. **Search**: Use the search bar to find specific bookmarks

### Category Management

1. Click "Manage Categories" to open category settings
2. Add new categories with custom names and colors
3. Edit or delete existing categories
4. Assign bookmarks to categories during creation/editing

### Auto Updates

The application automatically checks for updates on startup and can be manually triggered:

1. **Manual Check**: Click "Check for Updates" in the menu
2. **Test Updates**: Use development test buttons to simulate update scenarios
3. **Auto Install**: Updates are downloaded and installed automatically

## Configuration

Application data is stored in:
- **Windows**: `%APPDATA%\.urlnavigator\`
- **Data Files**: `urls.json`, `categories.json`

## Development

### Project Structure

```
url-navigator/
├── frontend/           # React frontend
│   ├── src/           # Source code
│   ├── dist/          # Build output
│   └── wailsjs/       # Wails bindings
├── build/             # Build artifacts
├── main.go            # Application entry point
├── app.go             # Core business logic
├── updater.go         # Update functionality
└── wails.json         # Wails configuration
```

### Building Releases

1. Update version in `wails.json`
2. Build for all platforms:
```bash
wails build -platform windows/amd64,darwin/amd64,linux/amd64
```
3. Create GitHub release with binaries
4. Tag the release for auto-update functionality

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Wails](https://wails.io/) - Go + Web frontend framework
- [shadcn/ui](https://ui.shadcn.com/) - React component library
- [MinIO selfupdate](https://github.com/minio/selfupdate) - Go self-update library