// faucet is an example of fund the account with the faucet to create it on-chain.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
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

	// Create account locally
	account, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create account " + err.Error())
	}

	//Fund the account with the faucet to create it on-chain
	err = client.Faucet(*account, endless.SequenceNumber(0)) // Use the sequence number to skip fetching it
	if err != nil {
		panic("Failed to faucet account " + err.Error())
	}
	balance, err := client.AccountEDSBalance(account.Address)
	if err != nil {
		panic("Failed to retrieve account balance:" + err.Error())
	}
	fmt.Printf("account EDS: %d\n", balance)
	assertBalance(client, account.Address, FaucetAmount)
	println("Account's balance after fund the account with the faucet 1000000000")

	//Do not use sequence number 0 again
	err = client.Faucet(*account, endless.SequenceNumber(0))
	if err != nil {
		fmt.Printf("Failed to faucet account with sequence number 0 again, err:%v \n\n", err.Error())
	}
	balance, err = client.AccountEDSBalance(account.Address)
	if err != nil {
		panic("Failed to retrieve account balance:" + err.Error())
	}
	assertBalance(client, account.Address, FaucetAmount)

	//Do not use error sequence number
	err = client.Faucet(*account, endless.SequenceNumber(99))
	if err != nil {
		fmt.Printf("Failed to faucet account with error sequence number, err:%v \n\n", err.Error())
	}
	balance, err = client.AccountEDSBalance(account.Address)
	if err != nil {
		panic("Failed to retrieve account balance:" + err.Error())
	}
	assertBalance(client, account.Address, FaucetAmount)

	//Fund the account with the faucet once every 24 hours
	err = client.Faucet(*account)
	if err != nil {
		fmt.Printf("Fund the account with the faucet once every 24 hours, err:%v \n\n", err.Error())
	}
	balance, err = client.AccountEDSBalance(account.Address)
	if err != nil {
		panic("Failed to retrieve account balance:" + err.Error())
	}
	assertBalance(client, account.Address, FaucetAmount)
}
func main() {
	example(endless.TestnetConfig)
}
