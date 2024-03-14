package main

import (
	"context"
	"fmt"

	"github.com/bludya/evm-rpc-utils/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ledgerwatch/log/v3"
)

// compare block hashes and binary search the first block where they mismatch
// then print the block number and the field differences
func main() {
	ctx := context.Background()
	rpcConfig, err := utils.GetConf()
	if err != nil {
		panic(fmt.Sprintf("RPGCOnfig: %s", err))
	}

	rpcClientA, err := ethclient.Dial(rpcConfig.Url1)
	if err != nil {
		panic(fmt.Sprintf("ethclient.Dial: %s", err))
	}
	rpcClientB, err := ethclient.Dial(rpcConfig.Url2)
	if err != nil {
		panic(fmt.Sprintf("ethclient.Dial: %s", err))
	}

	// highest block number
	highestBlockA, err := rpcClientA.BlockNumber(ctx)
	if err != nil {
		panic(fmt.Sprintf("rpcClientRemote.BlockNumber: %s", err))
	}
	highestBlockB, err := rpcClientB.BlockNumber(ctx)
	if err != nil {
		panic(fmt.Sprintf("rpcClientLocal.BlockNumber: %s", err))
	}
	highestBlockNumber := highestBlockB
	if highestBlockA < highestBlockB {
		highestBlockNumber = highestBlockA
	}

	log.Warn("Starting block traces mismatch check", "highestBlockRemote", highestBlockA, "highestBlockLocal", highestBlockB, "working highestBlockNumber", highestBlockNumber)

	lowestBlockNumber := uint64(0)
	checkBlockNumber := highestBlockNumber
	step := uint64(100)

	logEachRange := uint64(10000)

	var blockRemote, blockLocal *types.Block
	for i := lowestBlockNumber; i < checkBlockNumber; i += step {
		if i%logEachRange == 0 {
			log.Warn("Checking block", "blockNumber", i)
		}
		// get blocks
		blockRemote, blockLocal, err = utils.GetBlocks(ctx, rpcClientA, rpcClientB, i)
		if err != nil {
			log.Error(fmt.Sprintf("blockNum: %d, error getBlockTraces: %s", i, err))
			return
		}

		if match := utils.CompareBlocks(ctx, false, blockRemote, blockLocal, rpcClientA, rpcClientB); !match {
			log.Error("Mismatch found", "blockNum", i)
		}
	}
	log.Warn("Check finished!")
}
