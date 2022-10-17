# Block file

A small utility tool to generate and process block files. A block file contains blocks serialized with protobuf.

## Usage:

(when processing a block file you will need to stop your p2p service)

```ssh
koinos-block-file [FLAGS]
```

Flags available:
- `--mode -m`: Mode (`"generate"`: generate a block file or "process": process a block file) (default: `"generate"`)
- `--method -o`: Method used to fetch/submit blocks (`"json-rpc"|"amqp"`) (default: `"json-rpc"`)
- `--block-file-path -b`: Path to the block file (default: `"blocks.dat"`)
- `--amqp -a`: AMQP server URL (default: `"amqp://guest:guest@localhost:5672/"`)
- `--rpc -r`: JSON RPC server URL (default: `"http://localhost:8080/"`)
- `--start-block-height -s`: Start block height when generating/processing block file (default: `1`)
- `--nb-blocks-per-call -f`: Number of blocks to fetch per call when generating block file (default: `10000`)
- `--basedir -d`: Koinos base directory (default: `".koinos"`)
- `--instance-id -i`: The instance ID to identify this service (default: `random`)
- `--log-level -v`: The log filtering level (debug, info, warn, error) (default: `"info"`)

## Usage Examples:

Generate a block file from the genesis block using the https://api.koinos.io JSON RPC:
```ssh
koinos-block-file --rpc https://api.koinos.io --nb-blocks-per-call 1000
```

Process a block file using the local AMQP service:
```ssh
koinos-block-file --mode process --method amqp
```

## Build:
Build for Linux 64 bits:
```ssh
go get ./...
mkdir -p build/linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/linux/koinos-block-file cmd/koinos-block-file/main.go
```

Build for MacOS:
```ssh
go get ./...
mkdir -p build/macos
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o build/macos/koinos-block-file cmd/koinos-block-file/main.go
```

Build for Windows 64 bits:
```ssh
go get ./...
mkdir -p build/windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o build/windows/koinos-block-file.exe cmd/koinos-block-file/main.go
```