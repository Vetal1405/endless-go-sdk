// script_transaction_no_args is an example of how to make a script transaction with no args in the simplest possible way.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
)

/*
	script {
		use endless_framework::endless_account;

		fun main(from: &signer) {
			endless_account::transfer(from, @0xcafe, 1);
		}
	}
*/
const (
	scriptBytes = "0xa11ceb0b06000000060100020302050507090710190829200649220000000102010001060c0003060c05040f656e646c6573735f6163636f756e74087472616e7366657200000000000000000000000000000000000000000000000000000000000000010520000000000000000000000000000000000000000000000000000000000000cafe000001050b0007003201000000000000000000000000000000110002"
)

func example(networkConfig endless.NetworkConfig) {
	// Create a client for endless
	client, err := endless.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}

	// Create account locally for sender
	sender, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create sender:" + err.Error())
	}

	fmt.Printf("sender: %s\n", sender.Address.String())

	// Fund the sender with the faucet to create it on-chain
	err = client.Faucet(*sender, endless.SequenceNumber(0)) // Use the sequence number to skip fetching it
	if err != nil {
		panic("Failed to fund sender:" + err.Error())
	}

	// Now run a script version
	fmt.Printf("\n== Now running script version ==\n")
	runScript(client, sender)
}

func runScript(client *endless.Client, sender *endless.Account) {
	scriptCode, err := endless.ParseHex(scriptBytes)
	if err != nil {
		panic("Failed to parse script:" + err.Error())
	}

	// 1. Build transaction
	transactionPayload := endless.TransactionPayload{
		Payload: &endless.Script{
			Code:     scriptCode,
			ArgTypes: []endless.TypeTag{},
			Args:     []endless.ScriptArgument{},
		},
	}

	rawTransaction, err := client.BuildTransaction(sender.AccountAddress(), transactionPayload)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// 2. Simulate transaction (optional)
	// This is useful for understanding how much the transaction will cost
	// and to ensure that the transaction is valid before sending it to the network
	// This is optional, but recommended
	simulationResult, err := client.SimulateTransaction(rawTransaction, sender)
	if err != nil {
		panic("Failed to simulate transaction:" + err.Error())
	}
	fmt.Printf("\n=== Simulation ===\n")
	fmt.Printf("Gas unit price: %d\n", simulationResult[0].GasUnitPrice)
	fmt.Printf("Gas used: %d\n", simulationResult[0].GasUsed)
	fmt.Printf("Total gas fee: %d\n", simulationResult[0].GasUsed*simulationResult[0].GasUnitPrice)
	fmt.Printf("Status: %s\n\n", simulationResult[0].VmStatus)

	// 3. Sign transaction
	signedTransaction, err := rawTransaction.SignedTransaction(sender)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
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
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTransaction.Hash, userTransaction.Version)
}

func main() {
	example(endless.TestnetConfig)
}
