# Getting Started

## System Requirements

- OS: macOS, Linux, or Windows
- Arch: amd64 or arm64
- No extra dependencies required (SQLite is embedded)

## Option 1: Download Pre-built Binary (Recommended)

1. Go to the [Releases](https://codeberg.org/wsh233/golog/releases) page
2. Download the archive matching your OS and architecture
3. Extract to get `golog` (or `golog.exe` on Windows)
4. Grant execution permission (Linux/macOS):

   ```bash
   chmod +x golog
   ```

## Option 2: Build from Source

### Prerequisites

- [Go](https://go.dev/dl/) 1.25 or later
- Make (optional, for cross-compilation)

### Build Steps

```bash
# Clone the repository
git clone https://github.com/giserlab/golog.git
cd golog

# Build directly
go build -o golog main.go

# Or use Makefile for cross-compilation (CGO_ENABLED=0)
make build
```

The build output is located in the `bin/` directory.

## Run

### Basic Startup

```bash
./golog
```

The default listening port is `5201`. Open your browser and visit:

```
http://localhost:5201
```

### First Launch

If there is no `config.json` in the directory, the app will automatically redirect to the initialization wizard (`/wizard`). Follow the prompts to complete:

1. Set site name and description
2. Create an admin account
3. Configure basic options

### Command Line Options

```bash
# Specify port
./golog --port 8080

# Enable TLS
./golog --tls-crt server.crt --tls-key server.key

# Reset user password
./golog reset-password user@example.com

# Database migration
./golog db:migrate        # Migrate to latest version
./golog db:migrate 5      # Migrate to specific version
```

### Common Options

| Option      | Short | Description          | Default |
| ----------- | ----- | -------------------- | ------- |
| `--port`    | `-p`  | Listening port       | `5201`  |
| `--tls-crt` | -     | TLS certificate path | -       |
| `--tls-key` | -     | TLS private key path | -       |

## Directory Structure

The following directories and files are automatically created on first run:

```
golog/
├── config.json          # Site configuration file
├── db.sqlite            # SQLite database
├── data/
│   └── uploads/
│       ├── covers/      # Post cover images
│       └── images/      # Inline post images
└── ...
```

## Next Steps

- Visit the admin panel to write your first post
- Switch themes and personalize in settings
- Read [Features](./features) to learn about all capabilities
