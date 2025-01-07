// Package endless is a Go interface into the EndlessCoin blockchain.
//
// The EndlessCoin Go SDK provides a way to read on-chain data, submit transactions, and generally interact with the blockchain.
//
// Quick links:
//
//   - [Endless Docs] for learning more about EndlessCoin and how to use it.
//   - [Examples] are standalone runnable examples of how to use the SDK.
//
// You can create a client and send a transfer transaction with the below example:
//
//	// Create a Client
//	client, err := endless.NewClient(endless.TestnetConfig)
//	if err != nil {
//	panic("Failed to create client " + err.Error())
//	}
//
//	// Create an account
//	account, err := endless.NewEd25519Account()
//	if err != nil {
//	panic("Failed to create sender:" + err.Error())
//	}
//
//	// Fund the sender with the faucet to create it on-chain
//	err = client.Faucet(*account, endless.SequenceNumber(0))
//	if err != nil {
//	panic(fmt.Sprintf("Failed to fund account %s %w", account.AccountAddress(), err))
//	}
//
//	// Send funds to a different address
//	receiver, err := endless.NewEd25519Account()
//	if err != nil {
//	panic("Failed to create sender:" + err.Error())
//	}
//
//	// Build a transaction to send 1 EDS to the receiver
//	amount := 100_000_000 // 1 EDS
//	rawTxn, err := endless.EDSTransferTransaction(client, account, receiver.Address, uint64(amount))
//	if err != nil {
//	panic(fmt.Sprintf("Failed to build transaction %w", err))
//	}
//
//	// Sign transaction
//	signedTxn, err := rawTxn.SignedTransaction(account)
//	if err != nil {
//	panic("Failed to sign transaction:" + err.Error())
//	}
//
//	// Submit transaction
//	submitResult, err := client.SubmitTransaction(signedTxn)
//	if err != nil {
//	panic("Failed to submit transaction:" + err.Error())
//	}
//	txnHash := submitResult.Hash
//
//	// Wait for the transaction
//	userTransaction, err := client.WaitForTransaction(txnHash)
//	if err != nil {
//	panic("Failed to wait for transaction:" + err.Error())
//	}
//	if !userTransaction.Success {
//	panic("Failed to on chain success:" + userTransaction.VmStatus)
//	}
//	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTransaction.Hash, userTransaction.Version)
//
// [Examples]: https://github.com/endless-labs/endless-go-sdk/examples
//
// [EndlessCoin Docs]: https://docs.endless.link
package endless
