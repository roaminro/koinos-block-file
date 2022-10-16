package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	log "github.com/koinos/koinos-log-golang"
	koinosmq "github.com/koinos/koinos-mq-golang"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
	"github.com/koinos/koinos-proto-golang/koinos/rpc/block_store"
	"github.com/koinos/koinos-proto-golang/koinos/rpc/chain"
	util "github.com/koinos/koinos-util-golang"
	kjsonrpc "github.com/koinos/koinos-util-golang/rpc"
	"github.com/roaminro/koinos-block-file/internal/rpc"
	flag "github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
)

const (
	basedirOption          = "basedir"
	amqpUrlOption          = "amqp"
	rpcUrlOption           = "rpc"
	instanceIDOption       = "instance-id"
	logLevelOption         = "log-level"
	startBlockHeightOption = "start-block-height"
	blockFilePathOption    = "block-file-path"
	modeOption             = "mode"
	methodOption           = "method"
	nbBlocksPerCallOption  = "nb-blocks-per-call"
)

const (
	basedirDefault          = ".koinos"
	amqpUrlDefault          = "amqp://guest:guest@localhost:5672/"
	rpcUrlDefault           = "http://localhost:8080/"
	logLevelDefault         = "info"
	startBlockHeightDefault = 1
	nbBlocksPerCallDefault  = 10000
	blockFilePathDefault    = "blocks.dat"
	modeDefault             = "generate"
	methodDefault           = "json-rpc"
)

const (
	appName = "block_file"
	logDir  = "logs"

	defaultTimeout = time.Second * 60 * 10
)

func main() {

	var baseDir string

	baseDirPtr := flag.StringP(basedirOption, "d", basedirDefault, "Koinos base directory")
	mode := flag.StringP(modeOption, "m", modeDefault, "Mode (generate: generate a block file or process: process a block file)")
	method := flag.StringP(methodOption, "o", methodDefault, "Method used to fetch/submit blocks (json-rpc|amqp)")
	amqpUrl := flag.StringP(amqpUrlOption, "a", amqpUrlDefault, "AMQP server URL")
	rpcUrl := flag.StringP(rpcUrlOption, "r", rpcUrlDefault, "JSON RPC server URL")
	startBlockHeight := flag.Uint64P(startBlockHeightOption, "s", startBlockHeightDefault, "Start block height when generating/processing block file")
	nbBlocksPerCall := flag.Uint32P(nbBlocksPerCallOption, "f", nbBlocksPerCallDefault, "Number of blocks to fetch per call when generating block file")
	blockFilePath := flag.StringP(blockFilePathOption, "b", blockFilePathDefault, "Path to the block file")
	instanceID := flag.StringP(instanceIDOption, "i", util.GenerateBase58ID(5), "The instance ID to identify this service")
	logLevel := flag.StringP(logLevelOption, "v", logLevelDefault, "The log filtering level (debug, info, warn, error)")

	flag.Parse()

	// initialize base directory
	baseDir, err := util.InitBaseDir(*baseDirPtr)
	if err != nil {
		fmt.Printf("Could not initialize base directory '%v'\n", baseDir)
		os.Exit(1)
	}

	appID := fmt.Sprintf("%s.%s", appName, *instanceID)

	// Initialize logger
	logFilename := path.Join(util.GetAppDir(baseDir, appName), logDir, appName+".log")
	err = log.InitLogger(*logLevel, false, logFilename, appID)
	if err != nil {
		fmt.Printf("Invalid log-level: %s. Please choose one of: debug, info, warn, error", *logLevel)
		os.Exit(1)
	}

	log.Info("Starting with options:")
	log.Infof("basedir: %s", *baseDirPtr)
	log.Infof("mode: %s", *mode)
	log.Infof("method: %s", *method)
	log.Infof("block-file-path: %s", *blockFilePath)
	log.Infof("amqp: %s", *amqpUrl)
	log.Infof("rpc: %s", *rpcUrl)
	log.Infof("start-block-height: %d", *startBlockHeight)
	log.Infof("nb-blocks-per-call: %d", *nbBlocksPerCall)
	log.Infof("log-level: %s", *logLevel)
	log.Infof("instance-id: %s", *instanceID)
	log.Infof("app id: %s", appID)

	f, err := os.OpenFile(*blockFilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)

	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	defer f.Close()

	var jsonRpcClient *rpc.JsonRPC
	var amqpClient *rpc.KoinosRPC

	if *method == methodDefault {
		// method is set to "json-rpc"
		// init JSON RPC client
		rpcClient := kjsonrpc.NewKoinosRPCClient(*rpcUrl)
		jsonRpcClient = rpc.NewJsonRPC(rpcClient)
	} else {
		// method is set to "amqp"
		ctx, ctxCancel := context.WithCancel(context.Background())
		defer ctxCancel()

		rpcClient := koinosmq.NewClient(*amqpUrl, koinosmq.ExponentialBackoff)

		<-rpcClient.Start(ctx)

		amqpClient = rpc.NewKoinosRPC(rpcClient)

		log.Info("Attempting to connect to chain...")
		for {
			chainCtx, chainCancel := context.WithCancel(ctx)
			defer chainCancel()
			val, _ := amqpClient.IsConnectedToChain(chainCtx)
			if val {
				log.Info("Connected")
				break
			}
		}
	}

	log.Info("Started...")

	// generate the block file
	if *mode == modeDefault {
		go generateBlockFile(jsonRpcClient, amqpClient, *method, f, *startBlockHeight, *nbBlocksPerCall)
	} else {
		// process the block file
		go readBlockFile(jsonRpcClient, amqpClient, *method, *startBlockHeight, f)
	}

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Info("Shutting down node...")
}

func readLine(r *bufio.Reader) ([]byte, error) {
	var (
		isPrefix = true
		err      error
		line, ln []byte
	)

	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}

	return ln, err
}

func readBlockFile(jsonRpcClient *rpc.JsonRPC, amqpClient *rpc.KoinosRPC, method string, startHeight uint64, f *os.File) {
	reader := bufio.NewReader(f)
	var (
		line []byte
		err  error
	)

	for {
		line, err = readLine(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Error(err.Error())
		}

		data, err := base64.StdEncoding.DecodeString(string(line))
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}

		block := &protocol.Block{}

		err = proto.Unmarshal(data, block)

		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}

		if block.Header.Height >= startHeight {
			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
			defer cancel()

			// use json-rpc
			if method == methodDefault {
				_, err = jsonRpcClient.ApplyBlock(ctx, block)
			} else {
				// use amqp
				_, err = amqpClient.ApplyBlock(ctx, block)
			}

			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}

			log.Infof("Applied Block %d", block.Header.Height)
		} else {
			log.Infof("Skipped Block %d", block.Header.Height)
		}
	}

	log.Info("Block file successfully processed")
}

func generateBlockFile(jsonRpcClient *rpc.JsonRPC, amqpClient *rpc.KoinosRPC, method string, f *os.File, startHeight uint64, numBlocksPerCall uint32) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	var headInfo *chain.GetHeadInfoResponse
	var err error

	// json-rpc
	if method == methodDefault {
		headInfo, err = jsonRpcClient.GetHeadInfo(ctx)
	} else {
		// amqp
		headInfo, err = amqpClient.GetHeadInfo(ctx)
	}

	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	log.Infof("Last Irreversible Block %d", headInfo.LastIrreversibleBlock)

	var i uint64 = startHeight

	for i < headInfo.LastIrreversibleBlock {
		log.Infof("Fetching Blocks %d-%d", i, i+uint64(numBlocksPerCall))

		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		var blocks *block_store.GetBlocksByHeightResponse

		// json-rpc
		if method == methodDefault {
			blocks, err = jsonRpcClient.GetBlocksByHeight(ctx, headInfo.HeadTopology.Id, i, numBlocksPerCall)
		} else {
			// amqp
			blocks, err = amqpClient.GetBlocksByHeight(ctx, headInfo.HeadTopology.Id, i, numBlocksPerCall)
		}

		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}

		for _, blockItem := range blocks.BlockItems {
			data, err := proto.Marshal(blockItem.Block)

			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}

			b64 := base64.StdEncoding.EncodeToString(data)

			_, err = f.WriteString(b64 + "\n")

			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
		}

		log.Infof("Saved Blocks %d-%d", blocks.BlockItems[0].BlockHeight, blocks.BlockItems[len(blocks.BlockItems)-1].BlockHeight)

		i += uint64(numBlocksPerCall)
	}

	log.Info("Block file successfully generated")
}
