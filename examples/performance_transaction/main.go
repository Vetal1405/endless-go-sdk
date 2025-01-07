// performance_transaction shows how to improve performance of the transaction submission of a single transaction.
package main

import (
	"encoding/json"
	"github.com/endless-labs/endless-go-sdk"
	"time"
)

// example This example shows you how to improve performance of the transaction submission
//
// Speed can be improved by locally handling the sequence number, gas price, and other factors
func example(networkConfig endless.NetworkConfig) {
	start := time.Now()
	before := time.Now()

	// Create a client for Endless
	client, err := endless.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}
	println("New client:    ", time.Since(before).Milliseconds(), "ms")

	// Create a sender locally
	sender, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create sender:" + err.Error())
	}
	println("Create sender:", time.Since(before).Milliseconds(), "ms")

	before = time.Now()

	// Fund the sender with the faucet to create it on-chain
	err = client.Faucet(*sender, endless.SequenceNumber(0)) // Use the sequence number to skip fetching it
	println("Fund sender:", time.Since(before).Milliseconds(), "ms")

	before = time.Now()

	// Prep arguments
	receiver := endless.AccountOne
	amount := uint64(100)

	// Serialize arguments
	entryFunction, err := endless.CoinTransferPayload(nil, receiver, amount)
	if err != nil {
		panic("Failed to serialize arguments:" + err.Error())
	}

	rawTxn, err := client.BuildTransaction(
		sender.Address,
		endless.TransactionPayload{
			Payload: entryFunction,
		},
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}
	println("Build transaction:", time.Since(before).Milliseconds(), "ms")

	// Sign transaction
	before = time.Now()
	signedTxn, err := rawTxn.SignedTransaction(sender)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}
	println("Sign transaction:", time.Since(before).Milliseconds(), "ms")

	// Submit transaction
	before = time.Now()
	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash
	println("Submit transaction:", time.Since(before).Milliseconds(), "ms")

	// Wait for the transaction
	before = time.Now()
	userTransaction, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
	println("Wait for transaction:", time.Since(before).Milliseconds(), "ms")

	println("Total time:    ", time.Since(start).Milliseconds(), "ms")
	txnStr, _ := json.Marshal(userTransaction)
	println(string(txnStr))
}

func main() {
	example(endless.TestnetConfig)
}
