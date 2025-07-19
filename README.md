# Quickr - URL Shortener

A modern URL shortener built with Go, featuring a clean UI with HTMX for dynamic interactions.

## Features

- **URL Shortening**: Create short, memorable aliases for long URLs
- **Inline Editing**: Edit URLs and aliases directly in the list with HTMX
- **Real-time Search**: Search through links with debounced input
- **Statistics**: Track clicks and view usage statistics
- **Hot Links**: View trending links over different time periods
- **Dark Mode**: Built-in dark mode support

## Tech Stack

- **Backend**: Go with Gin framework
- **Database**: SQLite with GORM
- **Frontend**: HTMX + Tailwind CSS
- **Deployment**: Single binary with embedded assets

## Quick Start

### Using Docker (Recommended)

```bash
# Build and start the application
docker compose up -d

# View logs
docker compose logs -f

# Stop the application
docker compose down
```

The application will be available at http://localhost:8080

### Manual Installation

1. Build the binary:
```bash
go build -o quickr
```

2. Run the application:
```bash
./quickr
```

## Project Structure

```
quickr/
├── data/           # Database directory (created on first run)
├── handlers/       # HTTP handlers
├── models/         # Database models
├── static/         # Static assets
│   └── js/        # JavaScript files
├── templates/      # HTML templates
├── main.go        # Application entry point
└── resources.go   # Embedded resources
```

## Embedded Resources

The application uses Go's embed feature to include all necessary assets in a single binary:

- **Templates**: All HTML templates are embedded
- **Static Files**: JavaScript and other static assets are embedded
- **Database**: SQLite database is stored externally in a data directory

### How Embedding Works

1. Resources are defined in `resources.go`:
```go
//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/js/*.js
var staticFS embed.FS
```

2. The application loads these resources at runtime:
- Templates are parsed from `templateFS`
- Static files are served from `staticFS`
- Database is stored in `./data/quickr.db`

## Docker Support

### Configuration

- Multi-stage build for minimal image size
- Non-root user for security
- Volume for database persistence
- Automatic restart on failure

### Database Management

The SQLite database is stored in a Docker volume for persistence:

```bash
# Backup database
docker compose exec quickr sqlite3 /app/data/quickr.db ".backup '/app/data/backup.db'"

# Restore database
docker compose exec quickr sqlite3 /app/data/quickr.db ".restore '/app/data/backup.db'"
```

### Environment Variables

None required. The application uses sensible defaults:
- Port: 8080
- Database: /app/data/quickr.db

## Development

### Prerequisites

- Go 1.21 or higher
- SQLite3

### Local Development

1. Clone the repository:
```bash
git clone <repository-url>
cd quickr
```

2. Install dependencies:
```bash
go mod download
```

3. Run the application:
```bash
go run main.go
```

### Building

For production deployment:

```bash
# Build with embedded resources
go build -o quickr

# Test the build
./build.sh
```

## API Endpoints

- `GET /`: Homepage with link management
- `GET /hot`: Trending links view
- `GET /stats`: Usage statistics
- `GET /go/:alias`: Link redirection

API endpoints:
- `GET /api/links`: List all links
- `POST /api/links`: Create new link
- `PUT /api/links/:id`: Update link
- `DELETE /api/links/:id`: Delete link
- `GET /api/search`: Search links

## Security Considerations

- SQLite database is stored in a dedicated directory
- Docker container runs as non-root user
- No sensitive environment variables required
- Input validation for URLs and aliases

## License

MIT License