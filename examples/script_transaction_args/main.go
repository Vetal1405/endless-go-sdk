// script_transaction_args is an example of how to make a script transaction with args in the simplest possible way.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
	"math/big"
)

/*
	script {
		use endless_framework::endless_account;

		fun main(from: &signer, to: address, amount: u128) {
			endless_account::transfer(from, to, amount);
		}
	}
*/

const TransferAmount = 1_000
const scriptBytes = "a11ceb0b0600000005010002030205050706070d190826200000000100010003060c0504000f656e646c6573735f6163636f756e74087472616e736665720000000000000000000000000000000000000000000000000000000000000001000001050b000b010b02110002"

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
	fmt.Printf("recipient: %s\n", recipient.Address.String())

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

	// Now run a script version
	fmt.Printf("\n== Now running script version ==\n")
	runScript(client, sender, recipient)
}

func runScript(client *endless.Client, sender *endless.Account, recipient *endless.Account) {
	scriptCode, err := endless.ParseHex(scriptBytes)
	if err != nil {
		panic("Failed to parse script:" + err.Error())
	}

	// 1. Build transaction
	transactionPayload := endless.TransactionPayload{
		Payload: &endless.Script{
			Code:     scriptCode,
			ArgTypes: []endless.TypeTag{},
			Args: []endless.ScriptArgument{
				{
					Variant: endless.ScriptArgumentAddress,
					Value:   recipient.Address,
				},
				{
					Variant: endless.ScriptArgumentU128,
					Value:   *big.NewInt(int64(TransferAmount)),
				},
			},
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
	senderBalance, err := client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipientBalance, err := client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	fmt.Printf("\n================ Intermediate Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)

	// 7. Assert balance
	assertBalance(client, recipient.Address, TransferAmount)
}

func main() {
	example(endless.TestnetConfig)
}
