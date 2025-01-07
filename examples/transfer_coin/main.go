// transfer_coin is an example of how to make a coin transfer transaction in the simplest possible way.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
)

const TransferAmount = 1_000

// example This example shows you how to make an EDS transfer transaction in the simplest possible way
func example(networkConfig endless.NetworkConfig) {
	// Create a client for Endless
	client, err := endless.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}

	// Create accounts locally for sender and recipient
	sender, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create sender:" + err.Error())
	}
	recipient, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create recipient:" + err.Error())
	}

	fmt.Printf("\n================ Addresses ================\n")
	fmt.Printf("sender: %s\n", sender.Address.String())
	fmt.Printf("recipient: %s\n", recipient.Address.String())

	// Fund the sender with the faucet to create it on-chain
	err = client.Faucet(*sender, endless.SequenceNumber(0)) // Use the sequence number to skip fetching it
	if err != nil {
		panic("Failed to fund sender:" + err.Error())
	}

	senderBalance, err := client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipientBalance, err := client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	fmt.Printf("\n================ Initial Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)

	// 1. Build transaction
	entryFunction, err := endless.CoinTransferPayload(nil, recipient.Address, TransferAmount)
	if err != nil {
		panic("Failed to build transfer payload:" + err.Error())
	}

	rawTxn, err := client.BuildTransaction(
		sender.AccountAddress(),
		endless.TransactionPayload{
			Payload: entryFunction,
		},
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// 2. Simulate transaction (optional)
	// This is useful for understanding how much the transaction will cost
	// and to ensure that the transaction is valid before sending it to the network
	// This is optional, but recommended
	simulationResult, err := client.SimulateTransaction(rawTxn, sender)
	if err != nil {
		panic("Failed to simulate transaction:" + err.Error())
	}
	fmt.Printf("\n================ Simulation ================\n")
	fmt.Printf("Gas unit price: %d\n", simulationResult[0].GasUnitPrice)
	fmt.Printf("Gas used: %d\n", simulationResult[0].GasUsed)
	fmt.Printf("Total gas fee: %d\n", simulationResult[0].GasUsed*simulationResult[0].GasUnitPrice)
	fmt.Printf("Status: %s\n", simulationResult[0].VmStatus)

	// 3. Sign transaction
	signedTxn, err := rawTxn.SignedTransaction(sender)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}

	// 4. Submit transaction
	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	// 5. Wait for the transaction to complete
	userTransaction, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}

	// Check balances
	senderBalance, err = client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve percy balance:" + err.Error())
	}
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	fmt.Printf("\n================ Intermediate Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)

	// Now do it again, but with a different method
	resp, err := client.BuildSignAndSubmitTransaction(
		sender,
		endless.TransactionPayload{
			Payload: entryFunction,
		},
	)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}

	userTransaction, err = client.WaitForTransaction(resp.Hash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}

	senderBalance, err = client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	fmt.Printf("\n================ Final Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
}

func main() {
	example(endless.TestnetConfig)
}
