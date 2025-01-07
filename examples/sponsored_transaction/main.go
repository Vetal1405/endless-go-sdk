// sponsored_transaction is an example of how to make a sponsored transaction.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"

	"github.com/endless-labs/endless-go-sdk/crypto"
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
	sponsor, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create sponsor:" + err.Error())
	}

	fmt.Printf("\n================ Addresses ================\n")
	fmt.Printf("sender: %s\n", sender.Address.String())
	fmt.Printf("recipient: %s\n", recipient.Address.String())
	fmt.Printf("sponsor: %s\n", sponsor.Address.String())

	// Fund the sender with the faucet to create it on-chain
	err = client.Faucet(*sender, endless.SequenceNumber(0)) // Use the sequence number to skip fetching it
	if err != nil {
		panic("Failed to fund sender:" + err.Error())
	}

	// Fund the sponsor with the faucet to create it on-chain
	err = client.Faucet(*sponsor, endless.SequenceNumber(0)) // Use the sequence number to skip fetching it
	if err != nil {
		panic("Failed to fund sponsor:" + err.Error())
	}

	senderBalance, err := client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipientBalance, err := client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	sponsorBalance, err := client.AccountEDSBalance(sponsor.Address)
	if err != nil {
		panic("Failed to retrieve sponsor balance:" + err.Error())
	}
	fmt.Printf("\n================ Initial Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("sponsor EDS: %d\n", sponsorBalance)

	// Build transaction
	entryFunction, err := endless.CoinTransferPayload(nil, recipient.Address, TransferAmount)
	if err != nil {
		panic("Failed to build transfer payload:" + err.Error())
	}
	rawTxn, err := client.BuildTransactionMultiAgent(
		sender.Address,
		endless.TransactionPayload{
			Payload: entryFunction,
		},
		endless.FeePayer(&sponsor.Address),
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// Sign transaction
	senderAuth, err := rawTxn.Sign(sender)
	if err != nil {
		panic("Failed to sign transaction as sender:" + err.Error())
	}
	sponsorAuth, err := rawTxn.Sign(sponsor)
	if err != nil {
		panic("Failed to sign transaction as sponsor:" + err.Error())
	}

	signedFeePayerTxn, ok := rawTxn.ToFeePayerSignedTransaction(
		senderAuth,
		sponsorAuth,
		[]crypto.AccountAuthenticator{},
	)
	if !ok {
		panic("Failed to build fee payer signed transaction")
	}

	// Submit transaction
	submitResult, err := client.SubmitTransaction(signedFeePayerTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash
	println("Submitted transaction hash:", txnHash)

	// Wait for the transaction
	userTransaction, err := client.WaitForTransaction(txnHash)
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
	sponsorBalance, err = client.AccountEDSBalance(sponsor.Address)
	if err != nil {
		panic("Failed to retrieve sponsor balance:" + err.Error())
	}
	fmt.Printf("\n================ Intermediate Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("Sponsor EDS: %d\n", sponsorBalance)

	fmt.Printf("\n================ Now do it without knowing the signer ahead of time ================\n")

	rawTxn, err = client.BuildTransactionMultiAgent(
		sender.Address,
		endless.TransactionPayload{
			Payload: entryFunction,
		},
		endless.FeePayer(&endless.AccountZero), // Note that the Address is 0x0, because we don't know the signer
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// Alice signs the transaction, without knowing the sponsor
	senderAuth, err = rawTxn.Sign(sender)
	if err != nil {
		panic("Failed to sign transaction as sender:" + err.Error())
	}

	// The sponsor has to add themselves to the transaction to sign, note that this would likely be on a different
	// server
	ok = rawTxn.SetFeePayer(sponsor.Address)
	if !ok {
		panic("Failed to set fee payer")
	}
	sponsorAuth, err = rawTxn.Sign(sponsor)
	if err != nil {
		panic("Failed to sign transaction as sponsor:" + err.Error())
	}

	signedFeePayerTxn, ok = rawTxn.ToFeePayerSignedTransaction(
		senderAuth,
		sponsorAuth,
		[]crypto.AccountAuthenticator{},
	)
	if !ok {
		panic("Failed to build fee payer signed transaction")
	}

	// Submit transaction
	submitResult, err = client.SubmitTransaction(signedFeePayerTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash = submitResult.Hash
	println("Submitted transaction hash:", txnHash)

	// Wait for the transaction
	userTransaction, err = client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}

	fmt.Printf("\n================ Final Balances ================\n")
	senderBalance, err = client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	sponsorBalance, err = client.AccountEDSBalance(sponsor.Address)
	if err != nil {
		panic("Failed to retrieve sponsor balance:" + err.Error())
	}
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("sponsor EDS: %d\n", sponsorBalance)
}

func main() {
	example(endless.TestnetConfig)
}
