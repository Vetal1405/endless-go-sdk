// build_transfer_transaction_with_local_abi is an example of how to build a transfer transaction with local abi in the simplest possible way.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
	"github.com/endless-labs/endless-go-sdk/api"
	"math/big"
)

const TransferAmount = 1_000

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

	// 1. Build transaction, with a single move function ABI
	transferCoinsFunctionAbi := &api.MoveFunction{
		Name:              "transfer_coins",
		Visibility:        "public",
		IsEntry:           true,
		IsView:            false,
		GenericTypeParams: []*api.GenericTypeParam{{Constraints: []api.MoveAbility{}}},
		Params:            []string{"&signer", "address", "u128", "0x1::object::Object<T0>"},
		Return:            []string{},
	}
	entryFunction, err := endless.EntryFunctionFromAbi(
		transferCoinsFunctionAbi,
		endless.AccountOne,
		"endless_account",
		"transfer_coins",
		[]any{"0x1::fungible_asset::Metadata"},
		[]any{recipient.Address, TransferAmount, "ENDLESSsssssssssssssssssssssssssssssssssssss"},
	)
	if err != nil {
		panic("Failed to call EntryFunctionFromAbi:" + err.Error())
	}

	transactionPayload := endless.TransactionPayload{
		Payload: entryFunction,
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
	fmt.Printf("\n================ Simulation ================\n")
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

	// 6. Check balances
	senderBalance, err = client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	fmt.Printf("\n================ Intermediate Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)

	// 7. Assert balance
	assertBalance(client, recipient.Address, TransferAmount)

	// Now do it again

	// 1. Build transaction, but with a different function, with types
	moduleAbi := &api.MoveModule{
		Address:          &endless.AccountOne,
		Name:             "aptos_account",
		Friends:          []string{},
		ExposedFunctions: []*api.MoveFunction{transferCoinsFunctionAbi},
		Structs:          []*api.MoveStruct{},
	}
	entryFunction2, err := endless.EntryFunctionFromAbi(
		moduleAbi,
		endless.AccountOne,
		"endless_account",
		"transfer_coins",
		[]any{"0x1::fungible_asset::Metadata"},
		[]any{recipient.Address, TransferAmount, "ENDLESSsssssssssssssssssssssssssssssssssssss"},
	)
	if err != nil {
		panic("Failed to call EntryFunctionFromAbi:" + err.Error())
	}

	transactionPayload = endless.TransactionPayload{
		Payload: entryFunction,
	}
	rawTransaction, err = client.BuildTransaction(
		sender.AccountAddress(),
		transactionPayload,
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// 2. Simulate transaction (optional)
	// This is useful for understanding how much the transaction will cost
	// and to ensure that the transaction is valid before sending it to the network
	// This is optional, but recommended
	simulationResult, err = client.SimulateTransaction(rawTransaction, sender)
	if err != nil {
		panic("Failed to simulate transaction:" + err.Error())
	}
	fmt.Printf("\n================ Simulation ================\n")
	fmt.Printf("Gas unit price: %d\n", simulationResult[0].GasUnitPrice)
	fmt.Printf("Gas used: %d\n", simulationResult[0].GasUsed)
	fmt.Printf("Total gas fee: %d\n", simulationResult[0].GasUsed*simulationResult[0].GasUnitPrice)
	fmt.Printf("Status: %s\n\n", simulationResult[0].VmStatus)

	// 3. Sign And Submit transaction
	pendingTransaction, err = client.BuildSignAndSubmitTransaction(
		sender,
		endless.TransactionPayload{
			Payload: entryFunction2,
		},
	)
	if err != nil {
		panic("Failed to sign and submit transaction:" + err.Error())
	}

	// 4. Wait for the transaction to complete
	userTransaction, err = client.WaitForTransaction(pendingTransaction.Hash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTransaction.Hash, userTransaction.Version)

	// 5. Check balances
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

	// 6. Assert balance
	assertBalance(client, recipient.Address, TransferAmount*2)
}

func main() {
	example(endless.TestnetConfig)
}
