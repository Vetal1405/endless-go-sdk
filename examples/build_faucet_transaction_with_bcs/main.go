// build_faucet_transaction_with_bcs is an example of how to build a faucet transaction with bcs in the simplest possible way.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
	"github.com/endless-labs/endless-go-sdk/bcs"
	"math/big"
)

const FaucetAmount = uint64(1_000_000_000)

func assertBalance(client *endless.Client, address endless.AccountAddress, expectedBalance uint64) {
	amount, err := client.AccountEDSBalance(address)
	if err != nil {
		panic("failed to get balance: " + err.Error())
	}

	expectedBalanceBigInt := big.NewInt(int64(expectedBalance))
	if amount.Cmp(expectedBalanceBigInt) != 0 {
		panic(fmt.Sprintf("balance mismatch, got %d instead of %d", amount, expectedBalance))
	}
}

func example(networkConfig endless.NetworkConfig) {
	// Create a client for Endless
	client, err := endless.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client " + err.Error())
	}

	// Create account locally for sender
	sender, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create sender " + err.Error())
	}

	fmt.Printf("sender: %s\n\n", sender.Address.String())

	//1. Build TransactionPayload
	account1Bcs, err := bcs.Serialize(&sender.Address)
	if err != nil {
		panic("Failed to bcs sender " + err.Error())
	}

	entryFunction := &endless.EntryFunction{
		Module: endless.ModuleId{
			Address: endless.AccountOne,
			Name:    "faucet",
		},
		Function: "fund",
		ArgTypes: []endless.TypeTag{},
		Args: [][]byte{
			account1Bcs,
		},
	}

	transactionPayload := endless.TransactionPayload{
		Payload: entryFunction,
	}
	_, err = client.Account(sender.Address)
	var rawTransaction *endless.RawTransaction
	if err == nil {
		rawTransaction, err = client.BuildTransaction(
			sender.Address,
			transactionPayload,
		)
	} else {
		rawTransaction, err = client.BuildTransaction(
			sender.Address,
			transactionPayload,
			endless.SequenceNumber(0), // Use the sequence number to skip fetching it
		)
	}

	// 2. Simulate transaction (optional)
	// This is useful for understanding how much the transaction will cost
	// and to ensure that the transaction is valid before sending it to the network
	// This is optional, but recommended
	simulationResult, err := client.SimulateTransaction(rawTransaction, sender)
	if err != nil {
		panic("Failed to simulate transaction:" + err.Error())
	}
	fmt.Printf("\n================ Simulation ================\n")
	fmt.Printf("Gas unit price: %d\n", simulationResult[0].GasUnitPrice)
	fmt.Printf("Gas used: %d\n", simulationResult[0].GasUsed)
	fmt.Printf("Total gas fee: %d\n", simulationResult[0].GasUsed*simulationResult[0].GasUnitPrice)
	fmt.Printf("Status: %s\n", simulationResult[0].VmStatus)

	// 3. Sign transaction
	signedTransaction, err := rawTransaction.SignedTransaction(sender)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}
	if signedTransaction.Verify() != nil {
		panic("Failed to signed")
	}

	// 4. Submit transaction
	pendingTransaction, err := client.SubmitTransaction(signedTransaction)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}

	// 5. Wait for the transaction to complete
	userTransaction, err := client.WaitForTransaction(pendingTransaction.Hash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
	fmt.Printf("The transaction completed with hash: %s and version %d \n\n", userTransaction.Hash, userTransaction.Version)

	// 6. Check balance
	account1Balance, err := client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve percy balance:" + err.Error())
	}
	fmt.Printf("sender EDS: %d\n\n", account1Balance)

	// 7. Assert balance
	assertBalance(client, sender.Address, FaucetAmount)
}
func main() {
	example(endless.TestnetConfig)
}
