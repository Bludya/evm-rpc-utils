package main

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ledgerwatch/log/v3"

	"github.com/bludya/evm-rpc-utils/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// compare block hashes and binary search the first block where they mismatch
// then print the block number and the field differences
func main() {
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
	highestBlockA, err := rpcClientA.BlockNumber(context.Background())
	if err != nil {
		panic(fmt.Sprintf("rpcClientA.BlockNumber: %s", err))
	}
	highestBlockB, err := rpcClientB.BlockNumber(context.Background())
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
		blockA, blockB, err = getBlocks(*rpcClientA, *rpcClientB, checkBlockNumber)
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
	blockA, blockB, err = getBlocks(*rpcClientA, *rpcClientB, checkBlockNumber)
	if err != nil {
		log.Error(fmt.Sprintf("blockNum: %d, error getBlocks: %s", checkBlockNumber, err))
		return
	}

	if blockA.Hash() != blockB.Hash() {
		log.Warn("Blockhash mismatch", "blockNum", checkBlockNumber, "blockA.Hash", blockA.Hash().Hex(), "blockB.Hash", blockB.Hash().Hex())

		// check all fields
		if blockA.ParentHash() != blockB.ParentHash() {
			log.Warn("ParentHash", "A", blockA.ParentHash().Hex(), "B", blockB.ParentHash().Hex())
		}
		if blockA.UncleHash() != blockB.UncleHash() {
			log.Warn("UnclesHash", "A", blockA.UncleHash().Hex(), "B", blockB.UncleHash().Hex())
		}
		if blockA.Root() != blockB.Root() {
			log.Warn("Root", "A", blockA.Root().Hex(), "B", blockB.Root().Hex())
		}
		if blockA.TxHash() != blockB.TxHash() {
			log.Warn("TxHash", "A", blockA.TxHash().Hex(), "B", blockB.TxHash().Hex())
		}

		txHashesA := make([]common.Hash, len(blockA.Transactions()))
		for i, tx := range blockA.Transactions() {
			txHashesA[i] = tx.Hash()
		}
		txHashesB := make([]common.Hash, len(blockB.Transactions()))
		for i, tx := range blockB.Transactions() {
			txHashesB[i] = tx.Hash()
		}

		if len(txHashesA) != len(txHashesB) {
			log.Warn("Transactions amount mismatch", "A", len(txHashesA), "B", len(txHashesB))

			log.Warn("RPAc transactions", "txs", txHashesA)
			log.Warn("B transactions", "txs", txHashesB)
		} else {
			for i, txA := range txHashesB {
				txB := txHashesB[i]
				if txA != txB {
					log.Warn("TxHash", txA.Hex(), txB.Hex())
				}
			}
		}

		if blockA.ReceiptHash() != blockB.ReceiptHash() {
			log.Warn("ReceiptHash mismatch. Checking receipts", "A", blockA.ReceiptHash().Hex(), "B", blockB.ReceiptHash().Hex())
			for y, tx := range txHashesA {
				receiptA, receiptB, err := getReceipt(*rpcClientA, *rpcClientB, tx)
				if err != nil {
					log.Error(fmt.Sprintf("getReceipt: %s", err))
					return
				}
				log.Info("-------------------------------------------------")
				log.Info("Checking Receipts for tx.", "TxHash", tx.Hex())

				if receiptB.Status != receiptA.Status {
					log.Warn("ReceiptStatus", "A", receiptA.Status, "B", receiptB.Status)
				}
				if receiptB.CumulativeGasUsed != receiptA.CumulativeGasUsed {
					log.Warn("CumulativeGasUsed", "A", receiptA.CumulativeGasUsed, "B", receiptB.CumulativeGasUsed)
				}
				if !reflect.DeepEqual(receiptB.PostState, receiptA.PostState) {
					log.Warn("PostState", "A", common.BytesToHash(receiptA.PostState), "B", common.BytesToHash(receiptB.PostState))
				}
				if receiptB.ContractAddress != receiptA.ContractAddress {
					log.Warn("ContractAddress", "A", receiptA.ContractAddress, "B", receiptB.ContractAddress)
				}
				if receiptB.GasUsed != receiptA.GasUsed {
					log.Warn("GasUsed", "A", receiptA.GasUsed, "B", receiptB.GasUsed)
				}
				if receiptB.Bloom != receiptA.Bloom {
					log.Warn("LogsBloom", "A", receiptA.Bloom, "B", receiptB.Bloom)
				}

				if len(receiptA.Logs) != len(receiptB.Logs) {
					log.Warn("Receipt log amount mismatch", "receipt index", y, "A log amount", len(receiptA.Logs), "B log amount", len(receiptB.Logs))

					logIndexesA := make([]uint, len(receiptA.Logs))
					for i, log := range receiptA.Logs {
						logIndexesA[i] = log.Index
					}
					logIndexesB := make([]uint, len(receiptB.Logs))
					for i, log := range receiptB.Logs {
						logIndexesB[i] = log.Index
					}

					log.Warn("A log indexes", "indexes", logIndexesA)
					log.Warn("B log indexes", "indexes", logIndexesB)
				}

				// still check the available logs
				// there should be a mismatch on the first index they differ
				smallerLogLength := len(receiptB.Logs)
				if len(receiptA.Logs) < len(receiptB.Logs) {
					smallerLogLength = len(receiptA.Logs)
				}
				for i := 0; i < smallerLogLength; i++ {

					log.Warn("-------------------------------------------------")
					log.Warn("	Checking Logs for receipt.", "index", i)
					logB := receiptB.Logs[i]
					logA := receiptA.Logs[i]

					if logA.Address != logB.Address {
						log.Warn("Log Address", "index", i, "A", logA.Address, "B", logB.Address)
					}
					if !reflect.DeepEqual(logA.Data, logB.Data) {
						log.Warn("Log Data", "index", i, "A", logA.Data, "B", logB.Data)
					}

					for j, topicA := range logA.Topics {
						topicB := logB.Topics[j]
						if topicA != topicB {
							log.Warn("Log Topic", "Log index", i, "Topic index", j, "A", topicA, "B", topicB)
						}
					}
				}
				log.Warn("Finished tx check")
				log.Warn("-------------------------------------------------")
			}
		}
		if blockA.Bloom() != blockB.Bloom() {
			log.Warn("Bloom", "A", blockA.Bloom(), "B", blockB.Bloom())
		}
		if blockA.Difficulty().Cmp(blockB.Difficulty()) != 0 {
			log.Warn("Difficulty", "A", blockA.Difficulty().Uint64(), "B", blockB.Difficulty().Uint64())
		}
		if blockA.NumberU64() != blockB.NumberU64() {
			log.Warn("NumberU64", "A", blockA.NumberU64(), "B", blockB.NumberU64())
		}
		if blockA.GasLimit() != blockB.GasLimit() {
			log.Warn("GasLimit", "A", blockA.GasLimit(), "B", blockB.GasLimit())
		}
		if blockA.GasUsed() != blockB.GasUsed() {
			log.Warn("GasUsed", "A", blockA.GasUsed(), "B", blockB.GasUsed())
		}
		if blockA.Time() != blockB.Time() {
			log.Warn("Time", "A", blockA.Time(), "B", blockB.Time())
		}
		if blockA.MixDigest() != blockB.MixDigest() {
			log.Warn("MixDigest", "A", blockA.MixDigest().Hex(), "B", blockB.MixDigest().Hex())
		}
		if blockA.Nonce() != blockB.Nonce() {
			log.Warn("Nonce", "A", blockA.Nonce(), "B", blockB.Nonce())
		}
		if blockA.BaseFee() != blockB.BaseFee() {
			log.Warn("BaseFee", "A", blockA.BaseFee(), "B", blockB.BaseFee())
		}

		for i, txA := range blockA.Transactions() {
			txB := blockB.Transactions()[i]
			if txA.Hash() != txB.Hash() {
				log.Warn("TxHash", txA.Hash().Hex(), txB.Hash().Hex())
			}
		}
	}

	log.Warn("Check finished!")
}

func getReceipt(clientA, clientB ethclient.Client, txHash common.Hash) (*types.Receipt, *types.Receipt, error) {
	fmt.Println("++++++++++++++++++++++++++++++++++++++++++++++++")
	fmt.Println("txHash", txHash.Hex())
	fmt.Println("++++++++++++++++++++++++++++++++++++++++++++++++")

	receiptsA, err := clientA.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return nil, nil, fmt.Errorf("clientA.TransactionReceipts: %s", err)
	}
	receiptsB, err := clientB.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return nil, nil, fmt.Errorf("clientB.TransactionReceipts: %s", err)
	}
	return receiptsA, receiptsB, nil
}

func getBlocks(clientA, clientB ethclient.Client, blockNum uint64) (*types.Block, *types.Block, error) {
	blockNumBig := new(big.Int).SetUint64(blockNum)
	blockA, err := clientA.BlockByNumber(context.Background(), blockNumBig)
	if err != nil {
		return nil, nil, fmt.Errorf("clientA.BlockByNumber: %s", err)
	}
	blockB, err := clientB.BlockByNumber(context.Background(), blockNumBig)
	if err != nil {
		return nil, nil, fmt.Errorf("clientB.BlockByNumber: %s", err)
	}
	return blockA, blockB, nil
}
