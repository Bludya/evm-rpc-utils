package utils

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ledgerwatch/log/v3"
	"gopkg.in/yaml.v2"
)

type RpcConfig struct {
	Url1  string `yaml:"url1"`
	Url2  string `yaml:"url2"`
	Block int64  `yaml:"block"`
}

func GetConf() (RpcConfig, error) {
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

func GetReceipt(ctx context.Context, clientA, clientB *ethclient.Client, txHash common.Hash) (*types.Receipt, *types.Receipt, error) {
	fmt.Println("++++++++++++++++++++++++++++++++++++++++++++++++")
	fmt.Println("txHash", txHash.Hex())
	fmt.Println("++++++++++++++++++++++++++++++++++++++++++++++++")

	receiptsA, err := clientA.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, nil, fmt.Errorf("clientA.TransactionReceipts: %s", err)
	}
	receiptsB, err := clientB.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, nil, fmt.Errorf("clientB.TransactionReceipts: %s", err)
	}
	return receiptsA, receiptsB, nil
}

func GetBlocks(ctx context.Context, clientA, clientB *ethclient.Client, blockNum uint64) (*types.Block, *types.Block, error) {
	blockNumBig := new(big.Int).SetUint64(blockNum)
	blockA, err := clientA.BlockByNumber(ctx, blockNumBig)
	if err != nil {
		return nil, nil, fmt.Errorf("clientA.BlockByNumber: %s", err)
	}
	blockB, err := clientB.BlockByNumber(ctx, blockNumBig)
	if err != nil {
		return nil, nil, fmt.Errorf("clientB.BlockByNumber: %s", err)
	}
	return blockA, blockB, nil
}

// CompareBlocks compares two blocks and returns true if they match
func CompareBlocks(ctx context.Context, silent bool, blockA, blockB *types.Block, rpcClientA, rpcClientB *ethclient.Client) bool {
	matching := true
	// check all fields
	if blockA.ParentHash() != blockB.ParentHash() {
		if !silent {
			log.Warn("ParentHash", "A", blockA.ParentHash().Hex(), "B", blockB.ParentHash().Hex())
		}
		matching = false
	}
	if blockA.UncleHash() != blockB.UncleHash() {
		if !silent {
			log.Warn("UnclesHash", "A", blockA.UncleHash().Hex(), "B", blockB.UncleHash().Hex())
		}
		matching = false
	}
	if blockA.Root() != blockB.Root() {
		if !silent {
			log.Warn("Root", "A", blockA.Root().Hex(), "B", blockB.Root().Hex())
		}
		matching = false
	}
	if blockA.TxHash() != blockB.TxHash() {
		if !silent {
			log.Warn("TxHash", "A", blockA.TxHash().Hex(), "B", blockB.TxHash().Hex())
		}
		matching = false
	}

	txHashesA := make([]common.Hash, len(blockA.Transactions()))
	for i, tx := range blockA.Transactions() {
		txHashesA[i] = tx.Hash()
		matching = false
	}
	txHashesB := make([]common.Hash, len(blockB.Transactions()))
	for i, tx := range blockB.Transactions() {
		txHashesB[i] = tx.Hash()
	}

	if len(txHashesA) != len(txHashesB) {
		if !silent {
			log.Warn("Transactions amount mismatch", "A", len(txHashesA), "B", len(txHashesB))
		}

		if !silent {
			log.Warn("RPAc transactions", "txs", txHashesA)
		}
		if !silent {
			log.Warn("B transactions", "txs", txHashesB)
		}
		matching = false
	} else {
		for i, txA := range txHashesB {
			txB := txHashesB[i]
			if txA != txB {
				if !silent {
					log.Warn("TxHash", txA.Hex(), txB.Hex())
				}
				matching = false
			}
		}
	}

	if blockA.ReceiptHash() != blockB.ReceiptHash() {
		if !silent {
			log.Warn("ReceiptHash mismatch. Checking receipts", "A", blockA.ReceiptHash().Hex(), "B", blockB.ReceiptHash().Hex())
		}
		for y, tx := range txHashesA {
			receiptA, receiptB, err := GetReceipt(ctx, rpcClientA, rpcClientB, tx)
			if err != nil {
				log.Error(fmt.Sprintf("getReceipt: %s", err))
				return false
			}
			log.Info("-------------------------------------------------")
			log.Info("Checking Receipts for tx.", "TxHash", tx.Hex())

			if receiptB.Status != receiptA.Status {
				if !silent {
					log.Warn("ReceiptStatus", "A", receiptA.Status, "B", receiptB.Status)
				}
				matching = false
			}
			if receiptB.CumulativeGasUsed != receiptA.CumulativeGasUsed {
				if !silent {
					log.Warn("CumulativeGasUsed", "A", receiptA.CumulativeGasUsed, "B", receiptB.CumulativeGasUsed)
				}
				matching = false
			}
			if !reflect.DeepEqual(receiptB.PostState, receiptA.PostState) {
				if !silent {
					log.Warn("PostState", "A", common.BytesToHash(receiptA.PostState), "B", common.BytesToHash(receiptB.PostState))
				}
				matching = false
			}
			if receiptB.ContractAddress != receiptA.ContractAddress {
				if !silent {
					log.Warn("ContractAddress", "A", receiptA.ContractAddress, "B", receiptB.ContractAddress)
				}
				matching = false
			}
			if receiptB.GasUsed != receiptA.GasUsed {
				if !silent {
					log.Warn("GasUsed", "A", receiptA.GasUsed, "B", receiptB.GasUsed)
				}
				matching = false
			}
			if receiptB.Bloom != receiptA.Bloom {
				if !silent {
					log.Warn("LogsBloom", "A", receiptA.Bloom, "B", receiptB.Bloom)
				}
				matching = false
			}

			if len(receiptA.Logs) != len(receiptB.Logs) {
				if !silent {
					log.Warn("Receipt log amount mismatch", "receipt index", y, "A log amount", len(receiptA.Logs), "B log amount", len(receiptB.Logs))
				}

				logIndexesA := make([]uint, len(receiptA.Logs))
				for i, log := range receiptA.Logs {
					logIndexesA[i] = log.Index
				}
				logIndexesB := make([]uint, len(receiptB.Logs))
				for i, log := range receiptB.Logs {
					logIndexesB[i] = log.Index
				}

				if !silent {
					log.Warn("A log indexes", "indexes", logIndexesA)
				}
				if !silent {
					log.Warn("B log indexes", "indexes", logIndexesB)
				}
				matching = false
			}

			// still check the available logs
			// there should be a mismatch on the first index they differ
			smallerLogLength := len(receiptB.Logs)
			if len(receiptA.Logs) < len(receiptB.Logs) {
				smallerLogLength = len(receiptA.Logs)
			}
			for i := 0; i < smallerLogLength; i++ {

				if !silent {
					log.Warn("-------------------------------------------------")
				}
				if !silent {
					log.Warn("	Checking Logs for receipt.", "index", i)
				}
				logB := receiptB.Logs[i]
				logA := receiptA.Logs[i]

				if logA.Address != logB.Address {
					if !silent {
						log.Warn("Log Address", "index", i, "A", logA.Address, "B", logB.Address)
					}
					matching = false
				}
				if !reflect.DeepEqual(logA.Data, logB.Data) {
					if !silent {
						log.Warn("Log Data", "index", i, "A", logA.Data, "B", logB.Data)
					}
					matching = false
				}

				for j, topicA := range logA.Topics {
					topicB := logB.Topics[j]
					if topicA != topicB {
						if !silent {
							log.Warn("Log Topic", "Log index", i, "Topic index", j, "A", topicA, "B", topicB)
						}
						matching = false
					}
				}
			}
			if !silent {
				log.Warn("Finished tx check")
			}
			if !silent {
				log.Warn("-------------------------------------------------")
			}
		}
	}
	if blockA.Bloom() != blockB.Bloom() {
		if !silent {
			log.Warn("Bloom", "A", blockA.Bloom(), "B", blockB.Bloom())
		}
		matching = false
	}
	if blockA.Difficulty().Cmp(blockB.Difficulty()) != 0 {
		if !silent {
			log.Warn("Difficulty", "A", blockA.Difficulty().Uint64(), "B", blockB.Difficulty().Uint64())
		}
		matching = false
	}
	if blockA.NumberU64() != blockB.NumberU64() {
		if !silent {
			log.Warn("NumberU64", "A", blockA.NumberU64(), "B", blockB.NumberU64())
		}
		matching = false
	}
	if blockA.GasLimit() != blockB.GasLimit() {
		if !silent {
			log.Warn("GasLimit", "A", blockA.GasLimit(), "B", blockB.GasLimit())
		}
		matching = false
	}
	if blockA.GasUsed() != blockB.GasUsed() {
		if !silent {
			log.Warn("GasUsed", "A", blockA.GasUsed(), "B", blockB.GasUsed())
		}
		matching = false
	}
	if blockA.Time() != blockB.Time() {
		if !silent {
			log.Warn("Time", "A", blockA.Time(), "B", blockB.Time())
		}
		matching = false
	}
	if blockA.MixDigest() != blockB.MixDigest() {
		if !silent {
			log.Warn("MixDigest", "A", blockA.MixDigest().Hex(), "B", blockB.MixDigest().Hex())
		}
		matching = false
	}
	if blockA.Nonce() != blockB.Nonce() {
		if !silent {
			log.Warn("Nonce", "A", blockA.Nonce(), "B", blockB.Nonce())
		}
		matching = false
	}
	if blockA.BaseFee() != blockB.BaseFee() {
		if !silent {
			log.Warn("BaseFee", "A", blockA.BaseFee(), "B", blockB.BaseFee())
		}
		matching = false
	}

	for i, txA := range blockA.Transactions() {
		txB := blockB.Transactions()[i]
		if txA.Hash() != txB.Hash() {
			if !silent {
				log.Warn("TxHash", txA.Hash().Hex(), txB.Hash().Hex())
			}
			matching = false
		}
	}

	return matching
}
