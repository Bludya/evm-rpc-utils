package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"reflect"

	"github.com/ledgerwatch/log/v3"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gopkg.in/yaml.v2"
)

// compare block hashes and binary search the first block where they mismatch
// then print the block number and the field differences
func main() {
	rpcConfig, err := getConf()
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
			log.Warn("ParentHash", "Rpc", blockA.ParentHash().Hex(), "Rpc", blockB.ParentHash().Hex())
		}
		if blockA.UncleHash() != blockB.UncleHash() {
			log.Warn("UnclesHash", "Rpc", blockA.UncleHash().Hex(), "Local", blockB.UncleHash().Hex())
		}
		if blockA.Root() != blockB.Root() {
			log.Warn("Root", "Rpc", blockA.Root().Hex(), "Local", blockB.Root().Hex())
		}
		if blockA.TxHash() != blockB.TxHash() {
			log.Warn("TxHash", "Rpc", blockA.TxHash().Hex(), "Local", blockB.TxHash().Hex())
		}

		if blockA.ReceiptHash() != blockB.ReceiptHash() {
			log.Warn("ReceiptHash mismatch. Checking receipts", "Rpc", blockA.ReceiptHash().Hex(), "Local", blockB.ReceiptHash().Hex())
			for _, tx := range blockA.Transactions() {
				receiptA, receiptB, err := getReceipt(*rpcClientA, *rpcClientB, tx.Hash())
				if err != nil {
					log.Error(fmt.Sprintf("getReceipt: %s", err))
					return
				}
				log.Info("-------------------------------------------------")
				log.Info("Checking Receipts for tx.", "TxHash", tx.Hash().Hex())

				if receiptB.Status != receiptA.Status {
					log.Warn("ReceiptStatus", "Rpc", receiptA.Status, "Local", receiptB.Status)
				}
				if receiptB.CumulativeGasUsed != receiptA.CumulativeGasUsed {
					log.Warn("CumulativeGasUsed", "Rpc", receiptA.CumulativeGasUsed, "Local", receiptB.CumulativeGasUsed)
				}
				if !reflect.DeepEqual(receiptB.PostState, receiptA.PostState) {
					log.Warn("PostState", "Rpc", common.BytesToHash(receiptA.PostState), "Local", common.BytesToHash(receiptB.PostState))
				}
				if receiptB.ContractAddress != receiptA.ContractAddress {
					log.Warn("ContractAddress", "Rpc", receiptA.ContractAddress, "Local", receiptB.ContractAddress)
				}
				if receiptB.GasUsed != receiptA.GasUsed {
					log.Warn("GasUsed", "Rpc", receiptA.GasUsed, "Local", receiptB.GasUsed)
				}
				if receiptB.Bloom != receiptA.Bloom {
					log.Warn("LogsBloom", "Rpc", receiptA.Bloom, "Local", receiptB.Bloom)
				}

				for i, logA := range receiptA.Logs {
					logB := receiptB.Logs[i]
					if logA.Address != logB.Address {
						log.Warn("Log Address", "index", i, "Rpc", logA.Address, "Local", logB.Address)
					}
					if !reflect.DeepEqual(logA.Data, logB.Data) {
						log.Warn("Log Data", "index", i, "Rpc", logA.Data, "Local", logB.Data)
					}

					for j, topicA := range logA.Topics {
						topicB := logB.Topics[j]
						if topicA != topicB {
							log.Warn("Log Topic", "Log index", i, "Topic index", j, "Rpc", topicA, "Local", topicB)
						}
					}
				}
				log.Warn("Finished tx check")
				log.Warn("-------------------------------------------------")
			}
		}
		if blockA.Bloom() != blockB.Bloom() {
			log.Warn("Bloom", "Rpc", blockA.Bloom(), "Local", blockB.Bloom())
		}
		if blockA.Difficulty().Cmp(blockB.Difficulty()) != 0 {
			log.Warn("Difficulty", "Rpc", blockA.Difficulty().Uint64(), "Local", blockB.Difficulty().Uint64())
		}
		if blockA.NumberU64() != blockB.NumberU64() {
			log.Warn("NumberU64", "Rpc", blockA.NumberU64(), "Local", blockB.NumberU64())
		}
		if blockA.GasLimit() != blockB.GasLimit() {
			log.Warn("GasLimit", "Rpc", blockA.GasLimit(), "Local", blockB.GasLimit())
		}
		if blockA.GasUsed() != blockB.GasUsed() {
			log.Warn("GasUsed", "Rpc", blockA.GasUsed(), "Local", blockB.GasUsed())
		}
		if blockA.Time() != blockB.Time() {
			log.Warn("Time", "Rpc", blockA.Time(), "Local", blockB.Time())
		}
		if blockA.MixDigest() != blockB.MixDigest() {
			log.Warn("MixDigest", "Rpc", blockA.MixDigest().Hex(), "Local", blockB.MixDigest().Hex())
		}
		if blockA.Nonce() != blockB.Nonce() {
			log.Warn("Nonce", "Rpc", blockA.Nonce(), "Local", blockB.Nonce())
		}
		if blockA.BaseFee() != blockB.BaseFee() {
			log.Warn("BaseFee", "Rpc", blockA.BaseFee(), "Local", blockB.BaseFee())
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

type RpcConfig struct {
	Url1 string `yaml:"url1"`
	Url2 string `yaml:"url2"`
}

func getConf() (RpcConfig, error) {
	yamlFile, err := os.ReadFile("debugToolsConfig.yaml")
	if err != nil {
		return RpcConfig{}, err
	}

	c := RpcConfig{}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		return RpcConfig{}, err
	}

	return c, nil
}
