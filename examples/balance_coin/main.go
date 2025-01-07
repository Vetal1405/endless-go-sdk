// balance_coin is an example of how to get balance of a coin.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
)

func example(networkConfig endless.NetworkConfig) {
	client, err := endless.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client " + err.Error())
	}

	USDTCoin := "USDT1asiRNkRvD9auPGp1hr7AHboQARRAYWkFbVFZVX" //https://scan.endless.link/coins?network=testnet 		find One Coin

	accountOneEDSBalance, err := client.AccountEDSBalance(endless.AccountOne)
	if err != nil {
		panic("Failed to retrieve AccountOne EDS balance:" + err.Error())
	}
	fmt.Printf("AccountOne EDS: %d\n", accountOneEDSBalance)
	accountOneUSDTBalance, err := client.AccountCoinBalance(USDTCoin, endless.AccountOne)
	if err != nil {
		panic("Failed to retrieve AccountOne USDT balance:" + err.Error())
	}
	fmt.Printf("AccountOne USDT: %d\n", accountOneUSDTBalance)

	otherAddressBase58 := "2LpbrKKDjN9fzcSpLgzhgpTgb3QnguYqnoceyp2DCnsf" //https://scan.endless.link?network=testnet 	find One Account
	otherAccount := &endless.AccountAddress{}
	err = otherAccount.ParseStringRelaxed(otherAddressBase58)
	if err != nil {
		panic("Failed to retrieve address " + otherAccount.String() + ", err:" + err.Error())
	}
	otherEDSBalance, err := client.AccountEDSBalance(*otherAccount)
	if err != nil {
		panic("Failed to retrieve " + otherAccount.String() + " EDS balance:" + err.Error())
	}
	fmt.Printf("%s EDS: %d\n", otherAccount.String(), otherEDSBalance)
	otherUSDTBalance, err := client.AccountCoinBalance(USDTCoin, *otherAccount)
	if err != nil {
		panic("Failed to retrieve " + otherAccount.String() + " USDT balance:" + err.Error())
	}
	fmt.Printf("%s USDT: %d\n", otherAccount.String(), otherUSDTBalance)
}

func main() {
	example(endless.TestnetConfig)
}
