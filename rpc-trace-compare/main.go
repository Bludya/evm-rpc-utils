package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"github.com/bludya/evm-rpc-utils/utils"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ledgerwatch/log/v3"
)

func main() {
	rpcConfig, err := utils.GetConf()
	if err != nil {
		log.Error("RPGCOnfig", "err", err)
		return
	}

	log.Warn("Starting trace comparison", "blockNumber", rpcConfig.Block)
	defer log.Warn("Check finished.")

	rpcClientA, err := ethclient.Dial(rpcConfig.Url1)
	if err != nil {
		log.Error("rpcClientA.Dial", "err", err)
	}

	rpcClientB, err := ethclient.Dial(rpcConfig.Url2)
	if err != nil {
		log.Error("rpcClientB.Dial", "err", err)
	}

	blockNum := big.NewInt(rpcConfig.Block)
	// get A block
	blockA, err := rpcClientB.BlockByNumber(context.Background(), blockNum)
	if err != nil {
		log.Error("rpcClientB.BlockByNumber", "err", err)
	}

	// get B block
	blockB, err := rpcClientA.BlockByNumber(context.Background(), blockNum)
	if err != nil {
		log.Error("rpcClientA.BlockByNumber", "err", err)
	}

	// compare block tx hashes
	txHashesA := make([]string, len(blockA.Transactions()))
	for i, tx := range blockA.Transactions() {
		txHashesA[i] = tx.Hash().String()
	}

	txHashesB := make([]string, len(blockB.Transactions()))
	for i, tx := range blockB.Transactions() {
		txHashesB[i] = tx.Hash().String()
	}

	// just print errorand continue
	if len(txHashesA) != len(txHashesB) {
		log.Error("txHashesA != txHashesB", "txHashesA", txHashesA, "txHashesB", txHashesB)
	}

	if len(txHashesA) == 0 {
		log.Warn("Block A has no txs to compare")
		return
	}
	if len(txHashesB) == 0 {
		log.Warn("Block B has no txs to compare")
		return
	}

	// use the txs on A node since we might be limiting them for debugging purposes \
	// and those are the ones we want to check
	for _, txHash := range txHashesA {
		log.Warn("----------------------------------------------")
		log.Warn("Comparing tx", "txHash", txHash)

		traceA, err := getRpcTrace(rpcConfig.Url1, txHash)
		if err != nil {
			log.Error("Getting traceA failed:", "err", err)
			continue
		}

		traceB, err := getRpcTrace(rpcConfig.Url2, txHash)
		if err != nil {
			log.Error("Getting traceB failed:", "err", err)
			continue
		}

		if !compareTraces(traceA, traceB) {
			log.Warn("traces don't match", "txHash", txHash)
		}
	}
	defer log.Warn("----------------------------------------------")
}

func getRpcTrace(url string, txHash string) (*HttpResult, error) {
	payloadbytecode := RequestData{
		Method:  "debug_traceTransaction",
		Params:  []string{txHash},
		ID:      1,
		Jsonrpc: "2.0",
	}

	jsonPayload, err := json.Marshal(payloadbytecode)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	// req.SetBasicAuth(cfg.Username, cfg.Pass)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get rpc: %v", resp.Body)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var httpResp HTTPResponse
	json.Unmarshal(body, &httpResp)

	if httpResp.Error.Code != 0 {
		return nil, fmt.Errorf("failed to get trace: %v", httpResp.Error)
	}
	return &httpResp.Result, nil
}

func compareTraces(traceA, traceB *HttpResult) bool {
	traceMatches := true
	if traceA.Failed != traceB.Failed {
		log.Warn("\"failed\" field mismatch", "A", traceA.Failed, "B", traceB.Failed)
		traceMatches = false
	}

	if traceA.Gas != traceB.Gas {
		log.Warn("\"gas\" field mismatch", "A", traceA.Gas, "B", traceB.Gas)
		traceMatches = false
	}

	if traceA.ReturnValue != traceB.ReturnValue {
		log.Warn("\"returnValue\" field mismatch", "A", traceA.ReturnValue, "B", traceB.ReturnValue)
		traceMatches = false
	}

	logsA := traceA.StructLogs
	logsB := traceB.StructLogs
	traceAlen := len(logsA)
	traceBlen := len(logsB)
	if traceAlen != traceBlen {
		log.Warn("opcode counts mismatch.", "A count", traceAlen, "B count", traceBlen)
		traceMatches = false
	}

	log.Info("Getting the first opcode mismatch...")
	for i, loc := range logsA {
		roc := logsB[i]
		if !loc.cmp(roc) {

			if i == 0 {
				log.Warn("First opcode mismatch", "A", loc, "B", roc)
			} else {
				prevLoc := logsA[i-1]
				prevRoc := logsB[i-1]
				log.Warn("First opcode mismatch", "Previous A", prevLoc, "Previous B", prevRoc)
				traceMatches = false
			}
			break
		}
	}

	return traceMatches
}

type HTTPResponse struct {
	Result HttpResult `json:"result"`
	Error  HttpError  `json:"error"`
}

type HttpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type HttpResult struct {
	Gas         uint64      `json:"gas"`
	Failed      bool        `json:"failed"`
	ReturnValue interface{} `json:"returnValue"`
	StructLogs  []OpContext `json:"structLogs"`
}

type RequestData struct {
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	ID      int      `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
}

type OpContext struct {
	Pc      uint64   `json:"pc"`
	Op      string   `json:"op"`
	Gas     uint64   `json:"gas"`
	GasCost uint64   `json:"gasCost"`
	Depth   int      `json:"depth"`
	Stack   []string `json:"stack"`
	Refund  uint64   `json:"refund"`
}

func (oc *OpContext) cmp(b OpContext) bool {
	opMatches := true
	if oc.Pc != b.Pc {
		log.Warn("pc mismatch", "A", oc.Pc, "B", b.Pc)
		opMatches = false
	}
	if oc.Op != b.Op {
		log.Warn("op mismatch", "A", oc.Op, "B", b.Op)
		opMatches = false
	}

	if oc.Gas != b.Gas {
		log.Warn("gas mismatch", "A", oc.Gas, "B", b.Gas)
		opMatches = false
	}

	if oc.GasCost != b.GasCost {
		log.Warn("gasCost mismatch", "A", oc.GasCost, "B", b.GasCost)
		opMatches = false
	}

	if oc.Depth != b.Depth {
		log.Warn("depth mismatch", "A", oc.Depth, "B", b.Depth)
		opMatches = false
	}

	if oc.Refund != b.Refund {
		log.Warn("refund mismatch", "A", oc.Refund, "B", b.Refund)
		opMatches = false
	}

	if len(oc.Stack) != len(b.Stack) {
		log.Warn("stack length mismatch", "A", len(oc.Stack), "B", len(b.Stack))
		opMatches = false
	}

	foundMismatchingStack := false
	for i, AValue := range oc.Stack {
		BValue := b.Stack[i]
		if AValue != BValue {
			log.Warn("stack value mismatch", "index", i, "total A", len(oc.Stack), "A", AValue, "B", BValue)
			opMatches = false
			break
		}
	}

	if !foundMismatchingStack && !opMatches {
		log.Warn("Didn't find a mismatching stack entry. The stack matches up until the A stack's end.")
	}

	return opMatches
}
