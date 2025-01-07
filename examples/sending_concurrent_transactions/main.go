// sending_concurrent_transactions shows how to submit transactions serially or concurrently on a single account.
package main

import (
	"github.com/endless-labs/endless-go-sdk"
	"github.com/endless-labs/endless-go-sdk/api"
	"time"
)

func setup(networkConfig endless.NetworkConfig) (*endless.Client, endless.TransactionSigner) {
	client, err := endless.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}

	sender, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create sender:" + err.Error())
	}

	//Fund the accounts with the faucet to create it on-chain
	err = client.Faucet(*sender, endless.SequenceNumber(0)) // Use the sequence number to skip fetching it
	if err != nil {
		panic("Failed to fund sender:" + err.Error())
	}

	return client, sender
}

func payload() endless.TransactionPayload {
	receiver := endless.AccountOne
	amount := uint64(100)

	entryFunction, err := endless.CoinTransferPayload(nil, receiver, amount)
	if err != nil {
		panic("Failed to serialize arguments:" + err.Error())
	}
	return endless.TransactionPayload{Payload: entryFunction}
}

func sendManyTransactionsSerially(networkConfig endless.NetworkConfig, numTransactions uint64) {
	client, sender := setup(networkConfig)

	responses := make([]*api.SubmitTransactionResponse, numTransactions)

	sequenceNumber := uint64(1)
	for i := uint64(0); i < numTransactions; i++ {
		rawTxn, err := client.BuildTransaction(sender.AccountAddress(), payload(), endless.SequenceNumber(sequenceNumber))
		if err != nil {
			panic("Failed to build transaction:" + err.Error())
		}
		signedTxn, err := rawTxn.SignedTransaction(sender)
		if err != nil {
			panic("Failed to sign transaction:" + err.Error())
		}
		submitResult, err := client.SubmitTransaction(signedTxn)
		if err != nil {
			panic("Failed to submit transaction:" + err.Error())
		}
		responses[i] = submitResult
		sequenceNumber++
	}

	// Wait on last transaction
	userTransaction, err := client.WaitForTransaction(responses[numTransactions-1].Hash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
}

func sendManyTransactionsConcurrently(networkConfig endless.NetworkConfig, numTransactions uint64) {
	client, sender := setup(networkConfig)

	// start submission goroutine
	payloads := make(chan endless.TransactionBuildPayload, 50)
	results := make(chan endless.TransactionSubmissionResponse, 50)
	go client.BuildSignAndSubmitTransactions(sender, payloads, results)

	// Submit transactions to goroutine
	go func() {
		for i := uint64(0); i < numTransactions; i++ {
			payloads <- endless.TransactionBuildPayload{
				Id:    i,
				Type:  endless.TransactionSubmissionTypeSingle,
				Inner: payload(),
			}
		}
		close(payloads)
	}()

	// Wait for all transactions to be processed
	for result := range results {
		if result.Err != nil {
			panic("Failed to submit and wait for transaction:" + result.Err.Error())
		}
	}
}

// example This example shows you how to improve performance of the transaction submission
//
// Speed can be improved by locally handling the sequence number, gas price, and other factors
func example(networkConfig endless.NetworkConfig, numTransactions uint64) {
	println("Sending", numTransactions, "transactions Serially")
	startSerial := time.Now()
	sendManyTransactionsSerially(networkConfig, numTransactions)
	endSerial := time.Now()
	println("Serial:", time.Duration.Milliseconds(endSerial.Sub(startSerial)), "ms")

	println("Sending", numTransactions, "transactions Concurrently")
	startConcurrent := time.Now()
	sendManyTransactionsConcurrently(networkConfig, numTransactions)
	endConcurrent := time.Now()
	println("Concurrent:", time.Duration.Milliseconds(endConcurrent.Sub(startConcurrent)), "ms")

	println("Concurrent is", time.Duration.Milliseconds(endSerial.Sub(startSerial)-endConcurrent.Sub(startConcurrent)), "ms faster than Serial")
}

func main() {
	example(endless.TestnetConfig, 10)
}
