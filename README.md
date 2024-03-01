# evm-rpc-utils
A collection of utilities for interacting with and debugging EVM-compatible blockchains.

## usage
Copy `debugToolsConfig.yanl.example` to `debugToolsConfig.yaml` and fill in the required fields.

## rpc-match-blockhashes
This utility is used to compare the block hashes of two different RPC providers. It is useful for debugging and verifying the correctness of an RPC provider.

It uses binary search to find the first differing block hash between two RPC providers. It then prints the block number and hash of the differing block. Also it goes into the block fields to show the actual one that contributes to the blockhash difference.