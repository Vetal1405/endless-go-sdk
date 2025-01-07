// onchain_multisig is an example of how to create a multisig account and perform transactions with it.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
	"github.com/endless-labs/endless-go-sdk/api"
	"github.com/endless-labs/endless-go-sdk/bcs"
	"math/big"
	"time"
)

const TransferAmount = uint64(1_000_000)

func example(networkConfig endless.NetworkConfig) {
	client, err := endless.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client " + err.Error())
	}

	fmt.Printf("\n================ 1. Create three owners and recipient, Faucet all owners ================\n")

	// Create owners
	accounts := GenerateOwnerAccounts()
	println("Owner 1 =", accounts[0].Address.String())
	println("Owner 2 =", accounts[1].Address.String())
	println("Owner 3 =", accounts[2].Address.String())

	// Create recipient
	recipient, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create recipient " + err.Error())
	}
	println("recipient =", recipient.Address.String())

	// Fund the accounts with the faucet to create it on-chain
	FaucetAccounts(client, []*endless.Account{
		accounts[0],
		accounts[1],
		accounts[2],
	})

	fmt.Printf("\n================ Initial Balances ================\n")
	owner1Balance, err := client.AccountEDSBalance(accounts[0].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner2Balance, err := client.AccountEDSBalance(accounts[1].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner3Balance, err := client.AccountEDSBalance(accounts[2].Address)
	if err != nil {
		panic("Failed to retrieve accounts[1] balance:" + err.Error())
	}
	recipientBalance, err := client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)

	fmt.Printf("\n================ 2. Owner 1 create multi-sig ================\n")

	multisigAddress := SetUpMultisig(client, accounts)

	fmt.Printf("\n================ Intermediate Balances ================\n")
	owner1Balance, err = client.AccountEDSBalance(accounts[0].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner2Balance, err = client.AccountEDSBalance(accounts[1].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner3Balance, err = client.AccountEDSBalance(accounts[2].Address)
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	multisigAddressBalance, err := client.AccountEDSBalance(*multisigAddress)
	if err != nil {
		panic("Failed to retrieve multisigAddress balance:" + err.Error())
	}
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("multisigAddress EDS: %d\n", multisigAddressBalance)

	fmt.Printf("\n================ 3. Owner 1 tranfer EDS to Multi-sig ================\n")

	// Prep arguments
	entryFunction, err := endless.CoinTransferPayload(nil, *multisigAddress, TransferAmount*100)
	if err != nil {
		panic("Failed to build payload:" + err.Error())
	}

	// Submit transaction
	submitResult, err := client.BuildSignAndSubmitTransaction(accounts[0], endless.TransactionPayload{Payload: entryFunction})
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	// Wait for the transaction
	fmt.Printf("And we wait for the transaction %s to complete...\n", txnHash)
	userTransaction, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTransaction.Hash, userTransaction.Version)

	fmt.Printf("\n================ Intermediate Balances ================\n")
	owner1Balance, err = client.AccountEDSBalance(accounts[0].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner2Balance, err = client.AccountEDSBalance(accounts[1].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner3Balance, err = client.AccountEDSBalance(accounts[2].Address)
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	multisigAddressBalance, err = client.AccountEDSBalance(*multisigAddress)
	if err != nil {
		panic("Failed to retrieve multisigAddress balance:" + err.Error())
	}
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("multisigAddress EDS: %d\n", multisigAddressBalance)

	fmt.Printf("\n================ 4. Multi-sig tranfer EDS to recipient, sender is Owner 2, Owner 3 approve, execute is Owner 2 ================\n")
	/*
		1 	reject
		2 	sender 	Execute
		3 	approve
	*/
	println("Creating a multisig transaction to transfer coins...")
	multisigTransactionPayload := CreateMultisigTransferTransaction(client, accounts[1], *multisigAddress, recipient.Address)
	println("Owner 1 rejects but owner 3 approves.")
	RejectAndApprove(client, *multisigAddress, accounts[0], accounts[2], 1)
	println("Owner 2 can now execute the transfer transaction as it already has 2 approvals (from owners 2 and 3).")
	userTransaction = ExecuteTransaction(client, *multisigAddress, accounts[1], multisigTransactionPayload)
	if *userTransaction.Sender != accounts[1].Address {
		panic("sender not match")
	}

	assertBalance(client, recipient.Address, TransferAmount)
	println("Recipient's balance after transfer 1000000")

	fmt.Printf("\n================ Intermediate Balances ================\n")
	owner1Balance, err = client.AccountEDSBalance(accounts[0].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner2Balance, err = client.AccountEDSBalance(accounts[1].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner3Balance, err = client.AccountEDSBalance(accounts[2].Address)
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	multisigAddressBalance, err = client.AccountEDSBalance(*multisigAddress)
	if err != nil {
		panic("Failed to retrieve multisigAddress balance:" + err.Error())
	}
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("multisigAddress EDS: %d\n", multisigAddressBalance)

	fmt.Printf("\n================ 5. Multi-sig tranfer EDS to recipient again, sender is Owner 2, Owner 1 approve, execute is Owner 1 ================\n")
	/*
		1 	approve 	Execute
		2 	sender
		3 	reject
	*/
	println("Creating another multisig transaction using payload hash...")
	multisigTransactionPayload = CreateMultisigTransferTransactionWithHash(client, accounts[1], *multisigAddress, recipient.Address)
	println("Owner 3 rejects but owner 1 approves.")
	RejectAndApprove(client, *multisigAddress, accounts[2], accounts[0], 2)
	println("Owner 1 can now execute the transfer with hash transaction as it already has 2 approvals (from owners 1 and 2).")
	userTransaction = ExecuteTransaction(client, *multisigAddress, accounts[0], multisigTransactionPayload)
	if *userTransaction.Sender != accounts[0].Address {
		panic("sender not match")
	}

	// Check balance of recipient, should be 2_000_000
	assertBalance(client, recipient.Address, TransferAmount*2)
	println("Recipient's balance after transfer 2000000")

	fmt.Printf("\n================ Intermediate Balances ================\n")
	owner1Balance, err = client.AccountEDSBalance(accounts[0].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner2Balance, err = client.AccountEDSBalance(accounts[1].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner3Balance, err = client.AccountEDSBalance(accounts[2].Address)
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	multisigAddressBalance, err = client.AccountEDSBalance(*multisigAddress)
	if err != nil {
		panic("Failed to retrieve multisigAddress balance:" + err.Error())
	}
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("multisigAddress EDS: %d\n", multisigAddressBalance)

	fmt.Printf("\n================ 6. Add an owner, sender is Owner 2, Owner 3 approve, execute is Owner 2 ================\n")
	/*
		1 	reject
		2 	sender	Execute
		3 	approve
	*/
	println("Adding an owner to the multisig account...")
	multisigTransactionPayload = AddOwnerTransaction(client, accounts[1], *multisigAddress, recipient.Address)
	println("Owner 1 rejects but owner 3 approves.")
	RejectAndApprove(client, *multisigAddress, accounts[0], accounts[2], 3)
	println("Owner 2 can now execute the adding an owner transaction as it already has 2 approvals (from owners 2 and 3).")
	userTransaction = ExecuteTransaction(client, *multisigAddress, accounts[1], multisigTransactionPayload)
	if *userTransaction.Sender != accounts[1].Address {
		panic("sender not match")
	}

	time.Sleep(time.Second)

	_, owners := MultisigResource(client, multisigAddress)
	println("Number of Owners:", len(owners))
	if len(owners) != 4 {
		panic(fmt.Sprintf("Expected 4 owners got %d txn %s", len(owners), userTransaction.Hash))
	}

	fmt.Printf("\n================ Intermediate Balances ================\n")
	owner1Balance, err = client.AccountEDSBalance(accounts[0].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner2Balance, err = client.AccountEDSBalance(accounts[1].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner3Balance, err = client.AccountEDSBalance(accounts[2].Address)
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	multisigAddressBalance, err = client.AccountEDSBalance(*multisigAddress)
	if err != nil {
		panic("Failed to retrieve multisigAddress balance:" + err.Error())
	}
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("multisigAddress EDS: %d\n", multisigAddressBalance)

	fmt.Printf("\n================ 7. Remove an owner, sender is Owner 2, Owner 3 approve, execute is Owner 2 ================\n")
	/*
		1 	reject
		2 	sender	Execute
		3 	approve
	*/
	println("Removing an owner from the multisig account...")
	multisigTransactionPayload = RemoveOwnerTransaction(client, accounts[1], *multisigAddress, recipient.Address)
	println("Owner 1 rejects but owner 3 approves.")
	RejectAndApprove(client, *multisigAddress, accounts[0], accounts[2], 4)
	println("Owner 2 can now execute the removing an owner transaction as it already has 2 approvals (from owners 2 and 3).")
	userTransaction = ExecuteTransaction(client, *multisigAddress, accounts[1], multisigTransactionPayload)
	if *userTransaction.Sender != accounts[1].Address {
		panic("sender not match")
	}

	_, owners = MultisigResource(client, multisigAddress)
	println("Number of Owners:", len(owners))
	if len(owners) != 3 {
		panic(fmt.Sprintf("Expected 3 owners got %d txn %s", len(owners), userTransaction.Hash))
	}

	fmt.Printf("\n================ Intermediate Balances ================\n")
	owner1Balance, err = client.AccountEDSBalance(accounts[0].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner2Balance, err = client.AccountEDSBalance(accounts[1].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner3Balance, err = client.AccountEDSBalance(accounts[2].Address)
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	multisigAddressBalance, err = client.AccountEDSBalance(*multisigAddress)
	if err != nil {
		panic("Failed to retrieve multisigAddress balance:" + err.Error())
	}
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("multisigAddress EDS: %d\n", multisigAddressBalance)

	fmt.Printf("\n================ 8. Change threshold, sender is Owner 2, Owner 3 approve, execute is Owner 2 ================\n")
	/*
		1 	reject
		2 	sender	Execute
		3 	approve
	*/
	println("Changing the signature threshold to 3-of-3...")
	multisigTransactionPayload = ChangeThresholdTransaction(client, accounts[1], *multisigAddress, 3)
	println("Owner 1 rejects but owner 3 approves.")
	RejectAndApprove(client, *multisigAddress, accounts[0], accounts[2], 5)
	println("Owner 2 can now execute the change signature threshold transaction as it already has 2 approvals (from owners 2 and 3).")
	userTransaction = ExecuteTransaction(client, *multisigAddress, accounts[1], multisigTransactionPayload)
	if *userTransaction.Sender != accounts[1].Address {
		panic("sender not match")
	}

	threshold, _ := MultisigResource(client, multisigAddress)
	println("Signature Threshold: ", threshold)
	if threshold != 3 {
		panic(fmt.Sprintf("Expected 3-of-3 owners got %d-of-3 txn %s", threshold, userTransaction.Hash))
	}

	fmt.Printf("\n================ Final Balances ================\n")
	owner1Balance, err = client.AccountEDSBalance(accounts[0].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner2Balance, err = client.AccountEDSBalance(accounts[1].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner3Balance, err = client.AccountEDSBalance(accounts[2].Address)
	recipientBalance, err = client.AccountEDSBalance(recipient.Address)
	if err != nil {
		panic("Failed to retrieve recipient balance:" + err.Error())
	}
	multisigAddressBalance, err = client.AccountEDSBalance(*multisigAddress)
	if err != nil {
		panic("Failed to retrieve multisigAddress balance:" + err.Error())
	}
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)
	fmt.Printf("recipient EDS: %d\n", recipientBalance)
	fmt.Printf("multisigAddress EDS: %d\n", multisigAddressBalance)

	println("Multisig setup and transactions complete.")
}

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

func GenerateOwnerAccounts() []*endless.Account {
	accounts := make([]*endless.Account, 3)
	for i := 0; i < 3; i++ {
		account, err := endless.NewEd25519Account()
		if err != nil {
			panic("Failed to create account " + err.Error())
		}
		accounts[i] = account
	}
	return accounts
}

func FaucetAccounts(client *endless.Client, accounts []*endless.Account) {
	for _, account := range accounts {
		err := client.Faucet(*account, endless.SequenceNumber(0)) // Use the sequence number to skip fetching it
		if err != nil {
			panic("Failed to fund account " + err.Error())
		}
	}
}

func SetUpMultisig(client *endless.Client, accounts []*endless.Account) *endless.AccountAddress {
	println("Setting up a 2-of-3 multisig account...")

	// Step 1: Set up a 2-of-3 multisig account
	// ===========================================================================================
	// Get the next multisig account address. This will be the same as the account address of the multisig account we'll
	// be creating.
	multisigAddress, err := client.FetchNextMultisigAddress(accounts[0].Address)
	if err != nil {
		panic("Failed to fetch next multisig address: " + err.Error())
	}

	// Create the multisig account with 3 owners and a signature threshold of 2.
	CreateMultisig(client, accounts[0], []endless.AccountAddress{accounts[1].Address, accounts[2].Address})
	println("Multisig Account Address:", multisigAddress.String())

	// should be 2
	threshold, owners := MultisigResource(client, multisigAddress)
	println("Signature Threshold:", threshold)

	// should be 3
	println("Number of Owners:", len(owners))

	return multisigAddress
}

func CreateMultisig(client *endless.Client, account *endless.Account, additionalAddresses []endless.AccountAddress) {
	metadataValue, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
		bcs.SerializeSequenceWithFunction([]string{"example"}, ser, func(ser *bcs.Serializer, item string) {
			ser.WriteString(item)
		})
	})
	if err != nil {
		panic("Failed to serialize metadata value" + err.Error())
	}
	payload, err := endless.MultisigCreateAccountPayload(
		2,                   // Required signers
		additionalAddresses, // Other owners
		[]string{"example"}, // Metadata keys
		metadataValue,       //Metadata values
	)
	if err != nil {
		panic("Failed to create multisig account payload " + err.Error())
	}

	submitAndWait(client, account, payload)
}

func MultisigResource(client *endless.Client, multisigAddress *endless.AccountAddress) (uint64, []any) {
	resource, err := client.AccountResource(*multisigAddress, "0x1::multisig_account::MultisigAccount")
	if err != nil {
		panic("Failed to get resource for multisig account: " + err.Error())
	}

	resourceData := resource["data"].(map[string]any)
	numSigsRequiredStr := resourceData["num_signatures_required"].(string)

	numSigsRequired, err := endless.StrToUint64(numSigsRequiredStr)
	if err != nil {
		panic("Failed to convert string to u64: " + err.Error())
	}
	ownersArray := resourceData["owners"].([]any)

	return numSigsRequired, ownersArray
}

func CreateMultisigTransferTransaction(
	client *endless.Client,
	sender *endless.Account,
	multisigAddress endless.AccountAddress,
	recipient endless.AccountAddress,
) *endless.MultisigTransactionPayload {
	entryFunction, err := endless.CoinTransferPayload(nil, recipient, TransferAmount)
	if err != nil {
		panic("Failed to create payload for multisig transfer: " + err.Error())
	}

	multisigPayload := &endless.MultisigTransactionPayload{
		Variant: endless.MultisigTransactionPayloadVariantEntryFunction,
		Payload: entryFunction,
	}

	createTransactionPayload, err := endless.MultisigCreateTransactionPayload(multisigAddress, multisigPayload)
	if err != nil {
		panic("Failed to create payload to create transaction for multisig transfer: " + err.Error())
	}

	submitAndWait(client, sender, createTransactionPayload)
	return multisigPayload
}

func CreateMultisigTransferTransactionWithHash(
	client *endless.Client,
	sender *endless.Account,
	multisigAddress endless.AccountAddress,
	recipient endless.AccountAddress,
) *endless.MultisigTransactionPayload {
	entryFunction, err := endless.CoinTransferPayload(nil, recipient, TransferAmount)
	if err != nil {
		panic("Failed to create payload for multisig transfer: " + err.Error())
	}

	return createTransactionPayloadCommon(client, sender, multisigAddress, entryFunction)
}

func AddOwnerTransaction(client *endless.Client, sender *endless.Account, multisigAddress endless.AccountAddress, newOwner endless.AccountAddress) *endless.MultisigTransactionPayload {
	entryFunctionPayload := endless.MultisigAddOwnerPayload(newOwner)
	return createTransactionPayloadCommon(client, sender, multisigAddress, entryFunctionPayload)
}

func RemoveOwnerTransaction(client *endless.Client, sender *endless.Account, multisigAddress endless.AccountAddress, removedOwner endless.AccountAddress) *endless.MultisigTransactionPayload {
	entryFunctionPayload := endless.MultisigRemoveOwnerPayload(removedOwner)
	return createTransactionPayloadCommon(client, sender, multisigAddress, entryFunctionPayload)
}

func ChangeThresholdTransaction(client *endless.Client, sender *endless.Account, multisigAddress endless.AccountAddress, numSignaturesRequired uint64) *endless.MultisigTransactionPayload {
	entryFunctionPayload, err := endless.MultisigChangeThresholdPayload(numSignaturesRequired)
	if err != nil {
		panic("Failed to create payload for multisig remove owner: " + err.Error())
	}

	return createTransactionPayloadCommon(client, sender, multisigAddress, entryFunctionPayload)
}

func createTransactionPayloadCommon(client *endless.Client, sender *endless.Account, multisigAddress endless.AccountAddress, entryFunctionPayload *endless.EntryFunction) *endless.MultisigTransactionPayload {
	multisigPayload := &endless.MultisigTransactionPayload{
		Variant: endless.MultisigTransactionPayloadVariantEntryFunction,
		Payload: entryFunctionPayload,
	}

	createTransactionPayload, err := endless.MultisigCreateTransactionPayloadWithHash(multisigAddress, multisigPayload)
	if err != nil {
		panic("Failed to create payload to create transaction for multisig: " + err.Error())
	}

	submitAndWait(client, sender, createTransactionPayload)
	return multisigPayload
}

func RejectAndApprove(client *endless.Client, multisigAddress endless.AccountAddress, rejector *endless.Account, approver *endless.Account, transactionId uint64) {
	rejectPayload, err := endless.MultisigRejectPayload(multisigAddress, transactionId)
	if err != nil {
		panic("Failed to build reject transaction payload: " + err.Error())
	}
	submitAndWait(client, rejector, rejectPayload)

	approvePayload, err := endless.MultisigApprovePayload(multisigAddress, transactionId)
	if err != nil {
		panic("Failed to build approve transaction payload: " + err.Error())
	}

	submitAndWait(client, approver, approvePayload)
}

func ExecuteTransaction(client *endless.Client, multisigAddress endless.AccountAddress, sender *endless.Account, payload *endless.MultisigTransactionPayload) *api.UserTransaction {
	executionPayload := &endless.Multisig{
		MultisigAddress: multisigAddress,
		Payload:         payload,
	}
	return submitAndWait(client, sender, executionPayload)
}

func submitAndWait(client *endless.Client, sender *endless.Account, payload endless.TransactionPayloadImpl) *api.UserTransaction {
	submitResponse, err := client.BuildSignAndSubmitTransaction(sender, endless.TransactionPayload{Payload: payload})
	if err != nil {
		panic("Failed to submit transaction: " + err.Error())
	}

	userTransaction, err := client.WaitForTransaction(submitResponse.Hash)
	if err != nil {
		panic("Failed to wait for transaction: " + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}

	// Now check that there's no event for failed multisig
	for _, event := range userTransaction.Events {
		if event.Type == "0x1::multisig_account::TransactionExecutionFailed" {
			eventStr, _ := json.Marshal(event)
			panic(fmt.Sprintf("Multisig transaction failed. details: %s", eventStr))
		}
	}

	return userTransaction
}

func main() {
	example(endless.TestnetConfig)
}
