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

Available binaries:
- Linux
  - `pvcusage_Linux_x86_64` (64-bit)
  - `pvcusage_Linux_i386` (32-bit)
  - `pvcusage_Linux_arm64` (ARM 64-bit)
- macOS
  - `pvcusage_Darwin_x86_64` (Intel)
  - `pvcusage_Darwin_arm64` (Apple Silicon)
- Windows
  - `pvcusage_Windows_x86_64.exe` (64-bit)
  - `pvcusage_Windows_i386.exe` (32-bit)

Each release includes SHA256 checksums in `checksums.txt` for verification.

### Installing the binary

To install the latest version, run the following command according to your architecture:

#### For Linux (amd64)

```bash
wget https://github.com/joseEnrique/pvcusage/releases/download/v1.0.13/pvcusage_linux_amd64
sudo mv pvcusage_linux_amd64 /usr/local/bin/pvcusage
chmod +x /usr/local/bin/pvcusage
```

#### For Linux (arm64)

```bash
wget https://github.com/joseEnrique/pvcusage/releases/download/v1.0.13/pvcusage_linux_arm64
sudo mv pvcusage_linux_arm64 /usr/local/bin/pvcusage
chmod +x /usr/local/bin/pvcusage
```

#### For macOS (Intel)

```bash
wget https://github.com/joseEnrique/pvcusage/releases/download/v1.0.13/pvcusage_darwin_amd64
sudo mv pvcusage_darwin_amd64 /usr/local/bin/pvcusage
chmod +x /usr/local/bin/pvcusage
```

#### For macOS (Apple Silicon)

```bash
wget https://github.com/joseEnrique/pvcusage/releases/download/v1.0.13/pvcusage_darwin_arm64
sudo mv pvcusage_darwin_arm64 /usr/local/bin/pvcusage
chmod +x /usr/local/bin/pvcusage
```


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

### PVC Performance Monitoring

You can monitor the performance of a specific PVC that is being used by a pod. This feature creates a sidecar container that mounts the PVC and measures its performance metrics in real-time.

To monitor the performance of a specific PVC:
```bash
pvcusage -pvc my-pvc -namespace my-namespace -perf
```

This will display real-time metrics including:
- IOPS (Input/Output Operations Per Second)
- Throughput (MB/s)
- Latency (ms)
- Disk utilization (%)
- Storage capacity and usage

Press Ctrl+C to stop monitoring and clean up resources.

## Flags

- `-watch`: Enable watch mode (refresh every s seconds)
- `-s`: Interval in seconds for watch mode (default: 5)
- `-filter`: Filter PVCs by usage percentage (e.g., `>50`, `<=80`, `=90`)
- `-top`: Show only top N PVCs by usage percentage
- `-pvc`: Name of a specific PVC to analyze
- `-namespace`: Namespace of the PVC to analyze (required with -pvc)
- `-perf`: Enable performance monitoring for the specified PVC

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