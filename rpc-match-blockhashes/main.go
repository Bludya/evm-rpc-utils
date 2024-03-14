package main

import (
	"context"
	"fmt"

	"github.com/ledgerwatch/log/v3"

	"github.com/bludya/evm-rpc-utils/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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
		panic(fmt.Sprintf("rpcClientA.BlockNumber: %s", err))
	}
	highestBlockB, err := rpcClientB.BlockNumber(ctx)
	if err != nil {
		panic(fmt.Sprintf("rpcClientB.BlockNumber: %s", err))
	}
	highestBlockNumber := highestBlockA
	if highestBlockB < highestBlockA {
		highestBlockNumber = highestBlockB
	}

	log.Warn("Starting blockhash mismatch check", "highestBlockA", highestBlockA, "highestBlockB", highestBlockB, "working highestBlockNumber", highestBlockNumber)

	lowestBlockNumber := uint64(0)
	checkBlockNumber := highestBlockNumber

	var blockA, blockB *types.Block

	for {
		log.Warn("Checking for block", "blockNumber", checkBlockNumber)
		// get blocks
		blockA, blockB, err = utils.GetBlocks(ctx, rpcClientA, rpcClientB, checkBlockNumber)
		if err != nil {
			log.Error(fmt.Sprintf("blockNum: %d, error getBlocks: %s", checkBlockNumber, err))
			return
		}
		// if they match, go higher
		if blockA.Hash() == blockB.Hash() {
			lowestBlockNumber = checkBlockNumber + 1
			log.Warn("Blockhash match")
		} else {
			highestBlockNumber = checkBlockNumber
			log.Warn("Blockhash MISMATCH")
		}

		checkBlockNumber = (lowestBlockNumber + highestBlockNumber) / 2
		if lowestBlockNumber >= highestBlockNumber {
			break
		}
	}

	// get blocks
	blockA, blockB, err = utils.GetBlocks(ctx, rpcClientA, rpcClientB, checkBlockNumber)
	if err != nil {
		log.Error(fmt.Sprintf("blockNum: %d, error getBlocks: %s", checkBlockNumber, err))
		return
	}

	if blockA.Hash() != blockB.Hash() {
		utils.CompareBlocks(ctx, false, blockA, blockB, rpcClientA, rpcClientB)
	}

	log.Warn("Check finished!")
}
