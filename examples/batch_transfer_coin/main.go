// batch_transfer_coin is an example of how to make a coin batch transfer transaction in the simplest possible way.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
	"github.com/endless-labs/endless-go-sdk/crypto"
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
		err := client.Faucet(*account, endless.SequenceNumber(0))
		if err != nil {
			panic("Failed to fund account " + err.Error())
		}
	}
}

func example(networkConfig endless.NetworkConfig) {
	client, err := endless.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client " + err.Error())
	}

	fmt.Printf("\n================ 1. Create owners and only Owner 1 Faucet ================\n")

	accounts := GenerateOwnerAccounts()
	println("Owner 1 =", accounts[0].Address.String())
	println("Owner 2 =", accounts[1].Address.String())
	println("Owner 3 =", accounts[2].Address.String())

	// Fund the accounts with the faucet to create it on-chain
	FaucetAccounts(client, []*endless.Account{
		accounts[0],
	})

	fmt.Printf("\n================ Intermediate EDS Balances ================\n")
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
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)

	fmt.Printf("\n================ 2. Owner 1 batch tranfer EDS to Owner 2 and Owner 3 ================\n")

	// Prep arguments
	TransferEntryFunction, err := endless.CoinBatchTransferPayload(
		nil,

		[]endless.AccountAddress{
			accounts[1].Address,
			accounts[2].Address,
		},

		[]uint64{
			TransferAmount,
			TransferAmount * 2,
		},
	)
	if err != nil {
		panic("Failed to build payload:" + err.Error())
	}

	// Submit transaction
	submitResult, err := client.BuildSignAndSubmitTransaction(
		accounts[0],
		endless.TransactionPayload{
			Payload: TransferEntryFunction,
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

	fmt.Printf("\n================ Final EDS Balances ================\n")
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
	fmt.Printf("Owner 1 EDS: %d\n", owner1Balance)
	fmt.Printf("Owner 2 EDS: %d\n", owner2Balance)
	fmt.Printf("Owner 3 EDS: %d\n", owner3Balance)

	fmt.Printf("\n================ 3. One Account batch tranfer Other Coin to Owner 1 and Owner 2 and Owner 3 ================\n")

	OtherCoin := "USDT1asiRNkRvD9auPGp1hr7AHboQARRAYWkFbVFZVX" //https://scan.endless.link/coins?network=testnet 		find One Coin

	privateKey := &crypto.Ed25519PrivateKey{}
	err = privateKey.FromHex("0x12") //todo Your correct privateKey
	if err != nil {
		panic("Failed to get PrivateKey:" + err.Error())
	}
	account, err := endless.NewAccountFromSigner(privateKey)
	if err != nil {
		panic("Failed to get account:" + err.Error())
	}

	// Prep arguments
	TransferEntryFunction, err = endless.CoinBatchTransferPayload(
		&OtherCoin,

		[]endless.AccountAddress{
			accounts[0].Address,
			accounts[1].Address,
			accounts[2].Address,
		},

		[]uint64{
			TransferAmount,
			TransferAmount * 2,
			TransferAmount * 3,
		},
	)
	if err != nil {
		panic("Failed to build payload:" + err.Error())
	}

	// Submit transaction
	submitResult, err = client.BuildSignAndSubmitTransaction(
		account,
		endless.TransactionPayload{
			Payload: TransferEntryFunction,
		},
	)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash = submitResult.Hash

	// Wait for the transaction
	fmt.Printf("And we wait for the transaction %s to complete...\n", txnHash)
	userTransaction, err = client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTransaction.Hash, userTransaction.Version)

	fmt.Printf("\n================ USDT Balances ================\n")
	owner1Balance, err = client.AccountCoinBalance(OtherCoin, accounts[0].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner2Balance, err = client.AccountCoinBalance(OtherCoin, accounts[1].Address)
	if err != nil {
		panic("Failed to retrieve accounts[0] balance:" + err.Error())
	}
	owner3Balance, err = client.AccountCoinBalance(OtherCoin, accounts[2].Address)
	if err != nil {
		panic("Failed to retrieve accounts[1] balance:" + err.Error())
	}
	fmt.Printf("Owner 1 Other Coin: %d\n", owner1Balance)
	fmt.Printf("Owner 2 Other Coin: %d\n", owner2Balance)
	fmt.Printf("Owner 3 Other Coin: %d\n", owner3Balance)
}

func main() {
	example(endless.TestnetConfig)
}
