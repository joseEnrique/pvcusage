# PVC Usage Monitor

A command-line tool to monitor Persistent Volume Claim (PVC) usage in Kubernetes clusters.

## Features

- Real-time monitoring of PVC usage across all nodes
- Watch mode with configurable refresh interval
- Filter PVCs by usage percentage (e.g., `>50`, `<=80`, `=90`)
- Show only top N PVCs by usage percentage
- Human-readable output with proper formatting
- Graceful termination with SIGINT/SIGTERM handling

## Installation

### Using Go install (recommended)
```bash
go install github.com/joseEnrique/pvcusage@latest
```

### Using pre-built binaries
Download the latest release for your platform from the [releases page](https://github.com/joseEnrique/pvcusage/releases).

Supported platforms:
- Linux (amd64)

## Usage

Basic usage:
```bash
pvcusage
```

Watch mode with 5-second interval:
```bash
pvcusage -watch -s 5
```

Filter PVCs with usage > 80%:
```bash
pvcusage -filter ">80"
```

Show top 10 PVCs by usage:
```bash
pvcusage -top 10
```

Combine options:
```bash
pvcusage -watch -s 10 -filter ">50" -top 5
```

## Flags

- `-watch`: Enable watch mode (refresh every s seconds)
- `-s`: Interval in seconds for watch mode (default: 5)
- `-filter`: Filter PVCs by usage percentage (e.g., `>50`, `<=80`, `=90`)
- `-top`: Show only top N PVCs by usage percentage

## Project Structure

```
pvcusage/
├── main.go                    # Main entry point
├── internal/                  # Internal packages
│   ├── display/              # Display utilities
│   │   ├── humanize.go      # Human-readable formatting
│   │   └── table.go         # Table display
│   └── k8s/                 # Kubernetes operations
│       ├── client.go        # Kubernetes client
│       ├── types.go         # Type definitions
│       └── usage.go         # PVC usage operations
├── .github/                  # GitHub configuration
│   └── workflows/           # GitHub Actions workflows
│       └── release.yml      # Release workflow
├── .goreleaser.yml          # GoReleaser configuration
├── go.mod                   # Go module definition
└── README.md               # Project documentation
```

## Development

1. Clone the repository:
```bash
git clone https://github.com/joseEnrique/pvcusage.git
cd pvcusage
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the project:
```bash
go build
```

## Release Process

The project uses GitHub Actions for automated releases:

1. Create and push a tag:
```bash
git tag v1.0.0
git push origin v1.0.0
```

2. A new release will be created automatically with the binary attached.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see LICENSE file for details 