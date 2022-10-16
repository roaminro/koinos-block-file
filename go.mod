module github.com/roaminro/koinos-block-file

go 1.16

require (
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/btcsuite/btcd v0.22.1 // indirect
	github.com/klauspost/cpuid/v2 v2.0.12 // indirect
	github.com/koinos/koinos-log-golang v0.0.0-20220316225301-aaeabad5b543
	github.com/koinos/koinos-mq-golang v0.0.0-20220923190404-3c5aa9b8945a
	github.com/koinos/koinos-proto-golang v0.4.1-0.20220906183809-4e07dbd482f6
	github.com/koinos/koinos-util-golang v0.0.0-20220831225923-5ba6e0d4e7b9
	github.com/multiformats/go-multihash v0.1.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4 // indirect
	golang.org/x/sys v0.0.0-20220422013727-9388b58f7150 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	google.golang.org/protobuf v1.28.1
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	lukechampine.com/blake3 v1.1.7 // indirect
)

replace google.golang.org/protobuf => github.com/koinos/protobuf-go v1.27.2-0.20211026185306-2456c83214fe
