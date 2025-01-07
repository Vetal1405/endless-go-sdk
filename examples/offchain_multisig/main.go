// offchain_multisig is an example of how to create a multisig account and perform transactions with it.
package main

import (
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/endless-labs/endless-go-sdk"
	"github.com/endless-labs/endless-go-sdk/bcs"
	"github.com/endless-labs/endless-go-sdk/crypto"
	"math/big"
)

const TransferAmount = uint64(1_000_000)

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
			panic("Failed to faucet account " + err.Error())
		}
	}
}

func multisigResource(client *endless.Client, multisigAddress *endless.AccountAddress) (uint64, []string) {
	accountInfo, err := client.Account(*multisigAddress)
	if err != nil {
		panic("Failed to get resource for multisig account: " + err.Error())
	}

	return uint64(accountInfo.NumSignaturesRequired), accountInfo.AuthenticationKeyHex
}

// different
func CreateMultisig(client *endless.Client, accounts []*endless.Account, threshold uint64) *endless.AccountAddress {
	println("Setting up a 2-of-3 multisig account...")

	numSignaturesRequired, err := bcs.SerializeU64(threshold)
	if err != nil {
		panic("Signature Threshold error")
	}

	// 1. Build transaction multi agent
	rawTransactionWithData, err := client.BuildTransactionMultiAgent(
		accounts[0].Address,

		endless.TransactionPayload{
			Payload: &endless.EntryFunction{
				Module: endless.ModuleId{
					Address: endless.AccountOne,
					Name:    "account",
				},
				Function: "create_multisig_account",
				ArgTypes: []endless.TypeTag{},
				Args: [][]byte{
					numSignaturesRequired,
				},
			},
		},

		endless.AdditionalSigners{
			accounts[0].Address,
			accounts[1].Address,
			accounts[2].Address,
		},
	)
	if err != nil {
		panic("Failed to build transfer payload:" + err.Error())
	}

	// 2. Sign transaction
	owner1Authenticator, err := rawTransactionWithData.Sign(accounts[0])
	if err != nil {
		panic("Owner 1 Failed to sign transaction:" + err.Error())
	}
	owner2Authenticator, err := rawTransactionWithData.Sign(accounts[1])
	if err != nil {
		panic("Owner 2 Failed to sign transaction:" + err.Error())
	}
	owner3Authenticator, err := rawTransactionWithData.Sign(accounts[2])
	if err != nil {
		panic("Owner 3 Failed to sign transaction:" + err.Error())
	}

	// 3. signedTransaction
	signedTransaction, ok := rawTransactionWithData.ToMultiAgentSignedTransaction(
		owner1Authenticator,
		[]crypto.AccountAuthenticator{
			*owner1Authenticator,
			*owner2Authenticator,
			*owner3Authenticator,
		},
	)
	if !ok {
		panic("ToMultiAgentSignedTransaction error")
	}

	err = signedTransaction.Verify()
	if err != nil {
		panic("Failed to verify:" + err.Error())
	}

	// 4. Submit transaction
	pendingTransaction, err := client.SubmitTransaction(signedTransaction)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := pendingTransaction.Hash

	// 5. Wait for the transaction to complete
	userTransaction, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}

	// 6. Get multisigAddress
	var multisigAddress string
	for i := 0; i < len(userTransaction.Events); i++ {
		if userTransaction.Events[i].Type == "0x1::account::AddAuthenticationKey" {
			value, ok := userTransaction.Events[i].Data["account"]
			if ok {
				multisigAddress = value.(string)
			}
		}
	}
	println("Multisig Account Address:", multisigAddress)

	address := endless.AccountAddress(base58.Decode(multisigAddress))

	// should be 2
	threshold, owners := multisigResource(client, &address)
	println("Signature Threshold:", threshold)

	// should be 3
	println("Number of Owners:", len(owners))

	return &address
}

// same
func MultisigTransferEDS(client *endless.Client, multisigAddress *endless.AccountAddress, entryFunction *endless.EntryFunction, oldOwners ...*endless.Account) {
	// 1. Build transaction
	rawTransaction, err := client.BuildTransaction(
		*multisigAddress,
		endless.TransactionPayload{
			Payload: entryFunction,
		},
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// 2.Sign transaction
	oldAccountAuthenticator := []*crypto.AccountAuthenticator{}
	for _, account := range oldOwners {
		authenticator, err := rawTransaction.Sign(account)
		if err != nil {
			panic("Old Owner Failed to sign transaction:" + err.Error())
		}
		oldAccountAuthenticator = append(oldAccountAuthenticator, authenticator)
	}

	// 3. Verity Sign
	rawTransactionBcs, err := bcs.Serialize(rawTransaction)
	if err != nil {
		panic("rawTransaction bcs error:" + err.Error())
	}
	signMsg := append([]byte{}, endless.RawTransactionPrehash()[:]...)
	signMsg = append(signMsg, rawTransactionBcs...)

	multiAuthKeyAuthenticator := &crypto.MultiAuthKeyAuthenticator{}
	err = multiAuthKeyAuthenticator.FromAuthenticators(oldAccountAuthenticator)
	if err != nil {
		panic("Failed to MultiAuthKeyAuthenticator:" + err.Error())
	}
	if !multiAuthKeyAuthenticator.Verify(signMsg) {
		panic("Old Owner Verity Sign error")
	}

	// 4. signedTransaction
	transactionAuthenticator, err := endless.NewTransactionAuthenticator(&crypto.AccountAuthenticator{
		Variant: crypto.AccountAuthenticatorMultiAuthKey,
		Auth:    multiAuthKeyAuthenticator,
	})
	if err != nil {
		panic("Failed to NewTransactionAuthenticator:" + err.Error())
	}
	signedTransaction := &endless.SignedTransaction{
		Transaction:   rawTransaction,
		Authenticator: transactionAuthenticator,
	}

	// 5. Submit transaction
	submitResult, err := client.SubmitTransaction(signedTransaction)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	// 6. Wait for the transaction
	fmt.Printf("And we wait for the transaction %s to complete...\n", txnHash)
	userTransaction, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTransaction.Hash, userTransaction.Version)
}

// same
func MultisigAddOwner(client *endless.Client, multisigAddress *endless.AccountAddress, newOwner *endless.Account, threshold uint64, oldOwners ...*endless.Account) {
	numSignaturesRequired, err := bcs.SerializeU64(threshold)
	if err != nil {
		panic("Signature Threshold error")
	}

	// 1. Build transaction multi agent
	rawTransactionWithData, err := client.BuildTransactionMultiAgent(
		*multisigAddress,

		endless.TransactionPayload{
			Payload: &endless.EntryFunction{
				Module: endless.ModuleId{
					Address: endless.AccountOne,
					Name:    "account",
				},
				Function: "batch_add_authentication_key",
				ArgTypes: []endless.TypeTag{},
				Args: [][]byte{
					numSignaturesRequired,
				},
			},
		},

		endless.AdditionalSigners{
			newOwner.Address,
		},
	)
	if err != nil {
		panic("Failed to build transfer payload:" + err.Error())
	}

	// 2. Sign transaction
	oldAccountAuthenticatorPoint := []*crypto.AccountAuthenticator{}
	for _, account := range oldOwners {
		authenticator, err := rawTransactionWithData.Sign(account)
		if err != nil {
			panic("Old Owner Failed to sign transaction:" + err.Error())
		}
		oldAccountAuthenticatorPoint = append(oldAccountAuthenticatorPoint, authenticator)
	}

	newOwnerAuthenticator, err := rawTransactionWithData.Sign(newOwner)
	if err != nil {
		panic("New Owner Failed to sign transaction:" + err.Error())
	}

	// 3. Verity Sign
	rawTransactionWithDataBcs, err := bcs.Serialize(rawTransactionWithData)
	if err != nil {
		panic("rawTransactionWithData bcs error:" + err.Error())
	}
	signMsg := append([]byte{}, endless.RawTransactionWithDataPrehash()[:]...)
	signMsg = append(signMsg, rawTransactionWithDataBcs...)

	if !newOwner.Signer.PubKey().Verify(signMsg, newOwnerAuthenticator.Auth.Signature()) {
		panic("New Owner Verity Sign error")
	}

	multiAuthKeyAuthenticator := &crypto.MultiAuthKeyAuthenticator{}
	err = multiAuthKeyAuthenticator.FromAuthenticators(oldAccountAuthenticatorPoint)
	if err != nil {
		panic("Failed to MultiAuthKeyAuthenticator:" + err.Error())
	}
	if !multiAuthKeyAuthenticator.Verify(signMsg) {
		panic("Old Owner Verity Sign error")
	}

	// 4. signedTransaction
	signedTransaction, ok := rawTransactionWithData.ToMultiAgentSignedTransaction(
		&crypto.AccountAuthenticator{
			Variant: crypto.AccountAuthenticatorMultiAuthKey,
			Auth:    multiAuthKeyAuthenticator,
		},

		[]crypto.AccountAuthenticator{
			*newOwnerAuthenticator,
		},
	)
	if !ok {
		panic("signedTransaction error")
	}

	// 5. Submit transaction
	submitResult, err := client.SubmitTransaction(signedTransaction)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	// 6. Wait for the transaction
	fmt.Printf("And we wait for the transaction %s to complete...\n", txnHash)
	userTransaction, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTransaction.Hash, userTransaction.Version)
}

// same
func MultisigRemoveOwners(client *endless.Client, multisigAddress *endless.AccountAddress, removeOwners []endless.AccountAddress, threshold uint64, oldOwners ...*endless.Account) {
	numSignaturesRequired, err := bcs.SerializeU64(threshold)
	if err != nil {
		panic("threshold error")
	}

	removeBcs, _ := bcs.SerializeSingle(func(ser *bcs.Serializer) { //destBytes[0]=2  	destBytes[1-33]=32+AccountAddress 		destBytes[34-66]=32+AccountAddress
		ser.Uleb128(uint32(len(removeOwners)))
		for _, removeOwner := range removeOwners {
			ser.WriteBytes(removeOwner[:])
		}
	})

	// 1. Build transaction
	rawTransaction, err := client.BuildTransaction(
		*multisigAddress,

		endless.TransactionPayload{
			Payload: &endless.EntryFunction{
				Module: endless.ModuleId{
					Address: endless.AccountOne,
					Name:    "account",
				},
				Function: "batch_remove_authentication_key",
				ArgTypes: []endless.TypeTag{},
				Args: [][]byte{
					removeBcs,
					numSignaturesRequired,
				},
			},
		},
	)
	if err != nil {
		panic("Failed to build transfer payload:" + err.Error())
	}

	// 2. Sign transaction
	oldAccountAuthenticator := []*crypto.AccountAuthenticator{}
	for _, account := range oldOwners {
		authenticator, err := rawTransaction.Sign(account)
		if err != nil {
			panic("Old Owner Failed to sign transaction:" + err.Error())
		}
		oldAccountAuthenticator = append(oldAccountAuthenticator, authenticator)
	}

	// 3. Verity Sign
	rawTransactionBcs, err := bcs.Serialize(rawTransaction)
	if err != nil {
		panic("rawTransaction bcs error:" + err.Error())
	}
	signMsg := append([]byte{}, endless.RawTransactionPrehash()[:]...)
	signMsg = append(signMsg, rawTransactionBcs...)

	multiAuthKeyAuthenticator := &crypto.MultiAuthKeyAuthenticator{}
	err = multiAuthKeyAuthenticator.FromAuthenticators(oldAccountAuthenticator)
	if err != nil {
		panic("Failed to MultiAuthKeyAuthenticator:" + err.Error())
	}
	if !multiAuthKeyAuthenticator.Verify(signMsg) {
		panic("Old Owner Verity Sign error")
	}

	// 4. signedTransaction
	authenticator, err := endless.NewTransactionAuthenticator(&crypto.AccountAuthenticator{
		Variant: crypto.AccountAuthenticatorMultiAuthKey,
		Auth:    multiAuthKeyAuthenticator,
	})
	if err != nil {
		panic("Failed to NewTransactionAuthenticator:" + err.Error())
	}
	signedTransaction := &endless.SignedTransaction{
		Transaction:   rawTransaction,
		Authenticator: authenticator,
	}

	// 5. Submit transaction
	submitResult, err := client.SubmitTransaction(signedTransaction)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	// 6. Wait for the transaction
	fmt.Printf("And we wait for the transaction %s to complete...\n", txnHash)
	userTransaction, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTransaction.Hash, userTransaction.Version)
}

// same
func MultisigChangeThreshold(client *endless.Client, multisigAddress *endless.AccountAddress, threshold uint64, oldOwners ...*endless.Account) {
	numSignaturesRequired, err := bcs.SerializeU64(threshold)
	if err != nil {
		panic("threshold error")
	}

	// 1. Build transaction
	rawTransaction, err := client.BuildTransaction(
		*multisigAddress,

		endless.TransactionPayload{
			Payload: &endless.EntryFunction{
				Module: endless.ModuleId{
					Address: endless.AccountOne,
					Name:    "account",
				},
				Function: "set_num_signatures_required",
				ArgTypes: []endless.TypeTag{},
				Args: [][]byte{
					numSignaturesRequired,
				},
			},
		},
	)
	if err != nil {
		panic("Failed to build transfer payload:" + err.Error())
	}

	// 2. Sign transaction
	oldAccountAuthenticator := []*crypto.AccountAuthenticator{}
	for _, account := range oldOwners {
		authenticator, err := rawTransaction.Sign(account)
		if err != nil {
			panic("Old Owner Failed to sign transaction:" + err.Error())
		}
		oldAccountAuthenticator = append(oldAccountAuthenticator, authenticator)
	}

	// 3. Verity Sign
	rawTransactionBcs, err := bcs.Serialize(rawTransaction)
	if err != nil {
		panic("rawTransaction bcs error:" + err.Error())
	}
	signMsg := append([]byte{}, endless.RawTransactionPrehash()[:]...)
	signMsg = append(signMsg, rawTransactionBcs...)

	multiAuthKeyAuthenticator := &crypto.MultiAuthKeyAuthenticator{}
	err = multiAuthKeyAuthenticator.FromAuthenticators(oldAccountAuthenticator)
	if err != nil {
		panic("Failed to MultiAuthKeyAuthenticator:" + err.Error())
	}

	if !multiAuthKeyAuthenticator.Verify(signMsg) {
		panic("Old Owner Verity Sign error")
	}

	// 4. signedTransaction
	authenticator, err := endless.NewTransactionAuthenticator(&crypto.AccountAuthenticator{
		Variant: crypto.AccountAuthenticatorMultiAuthKey,
		Auth:    multiAuthKeyAuthenticator,
	})
	if err != nil {
		panic("Failed to NewTransactionAuthenticator:" + err.Error())
	}
	signedTransaction := &endless.SignedTransaction{
		Transaction:   rawTransaction,
		Authenticator: authenticator,
	}

	// 5. Submit transaction
	submitResult, err := client.SubmitTransaction(signedTransaction)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	// 6. Wait for the transaction
	fmt.Printf("And we wait for the transaction %s to complete...\n", txnHash)
	userTransaction, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTransaction.Hash, userTransaction.Version)
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

	multisigAddress := CreateMultisig(client, accounts, 2)

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
	if err != nil {
		panic("Failed to retrieve accounts[1] balance:" + err.Error())
	}
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

	fmt.Printf("\n================ 3. Owner 1 tranfer EDS to multi-sig ================\n")

	// Prep arguments
	entryFunction, err := endless.CoinTransferPayload(nil, *multisigAddress, TransferAmount*100)
	if err != nil {
		panic("Failed to build payload:" + err.Error())
	}

	// Submit transaction
	submitResult, err := client.BuildSignAndSubmitTransaction(
		accounts[0],
		endless.TransactionPayload{
			Payload: entryFunction,
		},
	)
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
	if err != nil {
		panic("Failed to retrieve accounts[1] balance:" + err.Error())
	}
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

	fmt.Printf("\n================ 4. Multi-sig tranfer EDS to recipient, sender and execute is Multi-sig, Owner 1 and Owner 2 sign ================\n")

	// Prep arguments
	entryFunction, err = endless.CoinTransferPayload(nil, recipient.Address, TransferAmount)
	if err != nil {
		panic("Failed to build payload:" + err.Error())
	}

	MultisigTransferEDS(client, multisigAddress, entryFunction, accounts[0], accounts[1])

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
	if err != nil {
		panic("Failed to retrieve accounts[1] balance:" + err.Error())
	}
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

	fmt.Printf("\n================ 5. Multi-sig tranfer EDS to recipient again, sender and execute is Multi-sig, Owner 1 and Owner 3 sign ================\n")

	// Prep arguments
	entryFunction, err = endless.CoinTransferPayload(nil, recipient.Address, TransferAmount)
	if err != nil {
		panic("Failed to build payload:" + err.Error())
	}

	MultisigTransferEDS(client, multisigAddress, entryFunction, accounts[0], accounts[2])

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
	if err != nil {
		panic("Failed to retrieve accounts[1] balance:" + err.Error())
	}
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

	fmt.Printf("\n================ 6. Add an owner, sender and execute is Multi-sig, Owner 2 and Owner 3 sign ================\n")

	MultisigAddOwner(client, multisigAddress, recipient, 2, accounts[1], accounts[2])

	threshold, owners := multisigResource(client, multisigAddress)
	if threshold != 2 {
		println("multi-sig threshold error")
	}
	if len(owners) != 4 {
		println("multi-sig Owners error")
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
	if err != nil {
		panic("Failed to retrieve accounts[1] balance:" + err.Error())
	}
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

	fmt.Printf("\n================ 7. Remove an owner, sender and execute is Multi-sig, Owner 1 and Owner 2 sign ================\n")

	MultisigRemoveOwners(client, multisigAddress, []endless.AccountAddress{recipient.Address, accounts[2].Address}, 2, accounts[0], accounts[1])
	if threshold != 2 {
		println("multi-sig threshold error")
	}
	if len(owners) != 2 {
		println("multi-sig Owners error")
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
	if err != nil {
		panic("Failed to retrieve accounts[1] balance:" + err.Error())
	}
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

	fmt.Printf("\n================ 8. Change threshold, sender and execute is Multi-sig, Owner 1 and Owner 2 sign ================\n")

	MultisigChangeThreshold(client, multisigAddress, 1, accounts[0], accounts[1])
	threshold, owners = multisigResource(client, multisigAddress)
	if threshold != 1 {
		println("multi-sig threshold error")
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
	if err != nil {
		panic("Failed to retrieve accounts[1] balance:" + err.Error())
	}
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

}

func main() {
	example(endless.TestnetConfig)
}
