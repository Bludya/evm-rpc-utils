# evm-rpc-utils
A collection of utilities for interacting with and debugging EVM-compatible blockchains.
They is useful for debugging and verifying the correctness of an RPC provider.

## usage
Copy `debugToolsConfig.yanl.example` to `debugToolsConfig.yaml` and fill in the required fields.

## rpc-match-blockhashes
This utility is used to compare the block hashes of two different RPC providers. 

It uses binary search to find the first differing block hash between two RPC providers. It then prints the block number and hash of the differing block. Also it goes into the block fields to show the actual one that contributes to the blockhash difference.

This is useful to find the first occurance of a bug, because the next block hash will differ simply because of the parent hash of the first differing block and the so on with the next blocks.

## rpc-trace-compare
This utility is used to compare the traces of two different RPC providers.

It fetches the block set in the config and then for each transaction in the block, it fetches the traces from both RPC providers and compares them. It goes into the opcodes and their stacks. Stops at the first opcode there is a difference and tells what the difference is.

Then it outputs the previous opcode, because it is the actual causer of the problem that is detected in the next opcode.

## rpc-blockreceipts-compare
Goes over the whole range of available blocks with some step, let's say each 100 blocks, and compares them deeply.

This is useful to find some RPC differences that do not affect the blockhash.