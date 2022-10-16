package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	koinosmq "github.com/koinos/koinos-mq-golang"
	"github.com/koinos/koinos-proto-golang/koinos/chain"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
	"github.com/koinos/koinos-proto-golang/koinos/rpc"
	"github.com/koinos/koinos-proto-golang/koinos/rpc/block_store"
	chainrpc "github.com/koinos/koinos-proto-golang/koinos/rpc/chain"
	"github.com/multiformats/go-multihash"
)

// RPC service constants
const (
	ChainRPC      = "chain"
	BlockStoreRPC = "block_store"
)

type chainError struct {
	Code int64 `json:"code"`
}

// KoinosRPC implements LocalRPC implementation by communicating with a local Koinos node via AMQP
type KoinosRPC struct {
	mq *koinosmq.Client
}

// NewKoinosRPC factory
func NewKoinosRPC(mq *koinosmq.Client) *KoinosRPC {
	rpc := new(KoinosRPC)
	rpc.mq = mq
	return rpc
}

// ApplyBlock rpc call
func (k *KoinosRPC) ApplyBlock(ctx context.Context, block *protocol.Block) (*chainrpc.SubmitBlockResponse, error) {
	args := &chainrpc.ChainRequest{
		Request: &chainrpc.ChainRequest_SubmitBlock{
			SubmitBlock: &chainrpc.SubmitBlockRequest{
				Block: block,
			},
		},
	}

	data, err := proto.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("error serializing ApplyBlock, %s", err)
	}

	var responseBytes []byte
	responseBytes, err = k.mq.RPC(ctx, "application/octet-stream", ChainRPC, data)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("rpc timeout ApplyBlock, %s", err)
		}
		return nil, fmt.Errorf("error local rpc ApplyBlock, %s", err)
	}

	responseVariant := &chainrpc.ChainResponse{}
	err = proto.Unmarshal(responseBytes, responseVariant)
	if err != nil {
		return nil, fmt.Errorf("error deserialization ApplyBlock, %s", err)
	}

	var response *chainrpc.SubmitBlockResponse

	switch t := responseVariant.Response.(type) {
	case *chainrpc.ChainResponse_SubmitBlock:
		response = t.SubmitBlock
	case *chainrpc.ChainResponse_Error:
		eData := chainError{}
		if jsonErr := json.Unmarshal([]byte(responseVariant.Response.(*chainrpc.ChainResponse_Error).Error.Data), &eData); jsonErr != nil {
			if eData.Code == int64(chain.ErrorCode_unknown_previous_block) {
				err = fmt.Errorf("unknown previous block")
				break
			} else if eData.Code == int64(chain.ErrorCode_pre_irreversibility_block) {
				err = fmt.Errorf("error block irreversibility")
				break
			}
		}
		err = fmt.Errorf("error local rpc ApplyBlock, chain rpc error, %s", string(t.Error.GetMessage()))
	default:
		err = fmt.Errorf("error local rpc ApplyBlock, unexpected chain rpc response")
	}

	return response, err
}

// // GetBlocksByHeight rpc call
func (k *KoinosRPC) GetBlocksByHeight(ctx context.Context, blockID multihash.Multihash, height uint64, numBlocks uint32) (*block_store.GetBlocksByHeightResponse, error) {
	args := &block_store.BlockStoreRequest{
		Request: &block_store.BlockStoreRequest_GetBlocksByHeight{
			GetBlocksByHeight: &block_store.GetBlocksByHeightRequest{
				HeadBlockId:         blockID,
				AncestorStartHeight: height,
				NumBlocks:           numBlocks,
				ReturnBlock:         true,
				ReturnReceipt:       false,
			},
		},
	}

	data, err := proto.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("serialization error GetBlocksByHeight, %s", err)
	}

	var responseBytes []byte
	responseBytes, err = k.mq.RPC(ctx, "application/octet-stream", BlockStoreRPC, data)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("local rpc timeout GetBlocksByHeight, %s", err)
		}
		return nil, fmt.Errorf("local rpc error GetBlocksByHeight, %s", err)
	}

	responseVariant := &block_store.BlockStoreResponse{}
	err = proto.Unmarshal(responseBytes, responseVariant)
	if err != nil {
		return nil, fmt.Errorf("deserialization error GetBlocksByHeight, %s", err)
	}

	var response *block_store.GetBlocksByHeightResponse

	switch t := responseVariant.Response.(type) {
	case *block_store.BlockStoreResponse_GetBlocksByHeight:
		response = t.GetBlocksByHeight
	case *block_store.BlockStoreResponse_Error:
		err = fmt.Errorf("local rpc error GetBlocksByHeight, block_store rpc error, %s", string(t.Error.GetMessage()))
	default:
		err = fmt.Errorf("local rpc error GetBlocksByHeight, unexpected block_store rpc response")
	}

	return response, err
}

// IsConnectedToChain returns if the AMQP connection can currently communicate
// with the chain microservice.
func (k *KoinosRPC) IsConnectedToChain(ctx context.Context) (bool, error) {
	args := &chainrpc.ChainRequest{
		Request: &chainrpc.ChainRequest_Reserved{
			Reserved: &rpc.ReservedRpc{},
		},
	}

	data, err := proto.Marshal(args)
	if err != nil {
		return false, fmt.Errorf("error serializattion IsConnectedToChain, %s", err)
	}

	var responseBytes []byte
	responseBytes, err = k.mq.RPC(ctx, "application/octet-stream", ChainRPC, data)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return false, fmt.Errorf("error local rpc timeout IsConnectedToChain, %s", err)
		}
		return false, fmt.Errorf("error local rpc IsConnectedToChain, %s", err)
	}

	responseVariant := &chainrpc.ChainResponse{}
	err = proto.Unmarshal(responseBytes, responseVariant)
	if err != nil {
		return false, fmt.Errorf("error deserialization IsConnectedToChain, %s", err)
	}

	return true, nil
}

// GetHeadInfo rpc call
func (k *KoinosRPC) GetHeadInfo(ctx context.Context) (*chainrpc.GetHeadInfoResponse, error) {
	args := &chainrpc.ChainRequest{
		Request: &chainrpc.ChainRequest_GetHeadInfo{
			GetHeadInfo: &chainrpc.GetHeadInfoRequest{},
		},
	}

	data, err := proto.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("serialization error GetForkHeads, %s", err)
	}

	var responseBytes []byte
	responseBytes, err = k.mq.RPC(ctx, "application/octet-stream", ChainRPC, data)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("local rpc timeout GetForkHeads, %s", err)
		}
		return nil, fmt.Errorf("local rpc error GetForkHeads, %s", err)
	}

	responseVariant := &chainrpc.ChainResponse{}
	err = proto.Unmarshal(responseBytes, responseVariant)
	if err != nil {
		return nil, fmt.Errorf("deserialization error, %s", err)
	}

	var response *chainrpc.GetHeadInfoResponse

	switch t := responseVariant.Response.(type) {
	case *chainrpc.ChainResponse_GetHeadInfo:
		response = t.GetHeadInfo
	case *chainrpc.ChainResponse_Error:
		err = fmt.Errorf("local rpc error GetForkHeads, chain rpc error, %s", string(t.Error.GetMessage()))
	default:
		err = fmt.Errorf("local rpc error GetForkHeads, unexpected chain rpc response")
	}

	return response, err
}
