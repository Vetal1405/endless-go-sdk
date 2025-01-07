// safe_transfer_coin is an example of how to make a coin safe transfer transaction in the simplest possible way.
package main

import (
	"fmt"
	"github.com/endless-labs/endless-go-sdk"
	"github.com/endless-labs/endless-go-sdk/bcs"
	"golang.org/x/crypto/sha3"
	"math/big"
	"strconv"
)

const TransferAmount = 1_000

func example(networkConfig endless.NetworkConfig) {
	// Create a client for Endless
	client, err := endless.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}

	// Create accounts locally for sender and recipient1,recipient2
	sender, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create sender:" + err.Error())
	}
	recipient1, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create recipient1:" + err.Error())
	}
	recipient2, err := endless.NewEd25519Account()
	if err != nil {
		panic("Failed to create recipient2:" + err.Error())
	}

	fmt.Printf("\n================ Addresses ================\n")
	fmt.Printf("sender: %s\n", sender.Address.String())
	fmt.Printf("recipient1: %s\n", recipient1.Address.String())
	fmt.Printf("recipient2: %s\n", recipient2.Address.String())

	// Fund the sender with the faucet to create it on-chain
	err = client.Faucet(*sender, endless.SequenceNumber(0)) // Use the sequence number to skip fetching it
	if err != nil {
		panic("Failed to fund sender:" + err.Error())
	}

	senderBalance, err := client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipient1Balance, err := client.AccountEDSBalance(recipient1.Address)
	if err != nil {
		panic("Failed to retrieve recipient1 balance:" + err.Error())
	}
	recipient2Balance, err := client.AccountEDSBalance(recipient2.Address)
	if err != nil {
		panic("Failed to retrieve recipient2 balance:" + err.Error())
	}
	fmt.Printf("\n================ Initial Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient1 EDS: %d\n", recipient1Balance)
	fmt.Printf("recipient2 EDS: %d\n", recipient2Balance)

	fmt.Printf("\n================ 1. safe transfer transaction ================\n")

	// 1. Build transaction
	entryFunction, err := endless.CoinTransferPayload(nil, recipient1.Address, TransferAmount)
	// entryFunction, err := endless.CoinSafeTransferPayload(nil, recipient1.Address, TransferAmount, [32]byte{})
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

	// 2. Simulate transaction
	simulationResult, err := client.SimulateTransaction(rawTxn, sender)
	if err != nil {
		panic("Failed to simulate transaction:" + err.Error())
	}
	fmt.Printf("\n================ Simulation ================\n")
	fmt.Printf("Gas unit price: %d\n", simulationResult[0].GasUnitPrice)
	fmt.Printf("Gas used: %d\n", simulationResult[0].GasUsed)
	fmt.Printf("Total gas fee: %d\n", simulationResult[0].GasUsed*simulationResult[0].GasUnitPrice)
	fmt.Printf("Status: %s\n", simulationResult[0].VmStatus)

	var bytes [][32 + 16]byte
	for _, userTransaction := range simulationResult {
		if !userTransaction.Success {
			panic("Simulate transaction err:" + userTransaction.VmStatus)
		}

		for _, event := range userTransaction.Events {
			if event.Type == "0x1::fungible_asset::Withdraw" {
				owner, _ := event.Data["owner"]
				if owner == sender.Address.String() {
					//store
					storeAccount := &endless.AccountAddress{}
					err = storeAccount.ParseStringRelaxed(event.Data["store"].(string))
					storeBcs, _ := bcs.Serialize(storeAccount)

					//amount
					amountString := event.Data["amount"].(string)
					amountInt, _ := strconv.Atoi(amountString)
					amountBcs, _ := bcs.SerializeU128(*big.NewInt(int64(amountInt)))

					fixBytes := [48]byte{}
					for i := 0; i < len(storeBcs); i++ {
						fixBytes[i] = storeBcs[i]
					}
					for i := 0; i < len(amountBcs); i++ {
						fixBytes[i+32] = amountBcs[i]
					}
					bytes = append(bytes, fixBytes)
				}
			}
		}
	}

	serializer := &bcs.Serializer{}
	serializer.Uleb128(uint32(len(bytes)))
	for _, b := range bytes {
		serializer.FixedBytes(b[:])
	}
	digestHash := sha3.Sum256(serializer.ToBytes())

	// 3. Build transaction again with safe
	safeEntryFunction, err := endless.CoinSafeTransferPayload(nil, recipient1.Address, TransferAmount, digestHash)
	if err != nil {
		panic("Failed to build transfer payload:" + err.Error())
	}

	// 4.Sign and Submit transaction
	resp, err := client.BuildSignAndSubmitTransaction(
		sender,
		endless.TransactionPayload{
			Payload: safeEntryFunction,
		},
	)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}

	userTransaction, err := client.WaitForTransaction(resp.Hash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	if !userTransaction.Success {
		panic("Failed to on chain success:" + userTransaction.VmStatus)
	}

	// 5. Wait for the transaction to complete
	senderBalance, err = client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipient1Balance, err = client.AccountEDSBalance(recipient1.Address)
	if err != nil {
		panic("Failed to retrieve recipient1 balance:" + err.Error())
	}

	senderBalance, err = client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipient1Balance, err = client.AccountEDSBalance(recipient1.Address)
	if err != nil {
		panic("Failed to retrieve recipient1 balance:" + err.Error())
	}
	recipient2Balance, err = client.AccountEDSBalance(recipient2.Address)
	if err != nil {
		panic("Failed to retrieve recipient2 balance:" + err.Error())
	}
	fmt.Printf("\n================ Intermediate Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient1 EDS: %d\n", recipient1Balance)
	fmt.Printf("recipient2 EDS: %d\n", recipient2Balance)

	fmt.Printf("\n================ 2. safe batch transfer transaction ================\n")
	// 1. Build transaction
	entryFunction, err = endless.CoinBatchTransferPayload(
		nil,
		[]endless.AccountAddress{
			recipient1.Address,
			recipient2.Address,
		},
		[]uint64{
			TransferAmount,
			TransferAmount * 2,
		},
	)
	if err != nil {
		panic("Failed to build transfer payload:" + err.Error())
	}

	rawTxn, err = client.BuildTransaction(
		sender.AccountAddress(),
		endless.TransactionPayload{
			Payload: entryFunction,
		},
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// 2. Simulate transaction
	simulationResult, err = client.SimulateTransaction(rawTxn, sender)
	if err != nil {
		panic("Failed to simulate transaction:" + err.Error())
	}
	fmt.Printf("\n================ Simulation ================\n")
	fmt.Printf("Gas unit price: %d\n", simulationResult[0].GasUnitPrice)
	fmt.Printf("Gas used: %d\n", simulationResult[0].GasUsed)
	fmt.Printf("Total gas fee: %d\n", simulationResult[0].GasUsed*simulationResult[0].GasUnitPrice)
	fmt.Printf("Status: %s\n", simulationResult[0].VmStatus)

	var bytes2 [][32 + 16]byte
	for _, userTransaction := range simulationResult {
		if !userTransaction.Success {
			panic("Simulate transaction err:" + userTransaction.VmStatus)
		}

		for _, event := range userTransaction.Events {
			if event.Type == "0x1::fungible_asset::Withdraw" {
				owner, _ := event.Data["owner"]
				if owner == sender.Address.String() {
					//store
					storeAccount := &endless.AccountAddress{}
					err = storeAccount.ParseStringRelaxed(event.Data["store"].(string))
					storeBcs, _ := bcs.Serialize(storeAccount)

					//amount
					amountString := event.Data["amount"].(string)
					amountInt, _ := strconv.Atoi(amountString)
					amountBcs, _ := bcs.SerializeU128(*big.NewInt(int64(amountInt)))

					fixBytes := [48]byte{}
					for i := 0; i < len(storeBcs); i++ {
						fixBytes[i] = storeBcs[i]
					}
					for i := 0; i < len(amountBcs); i++ {
						fixBytes[i+32] = amountBcs[i]
					}
					bytes2 = append(bytes2, fixBytes)
				}
			}
		}
	}

	serializer2 := &bcs.Serializer{}
	serializer2.Uleb128(uint32(len(bytes2)))
	for _, b := range bytes2 {
		serializer2.FixedBytes(b[:])
	}
	digestHash2 := sha3.Sum256(serializer2.ToBytes())

	// 3. Build transaction again with safe
	safeEntryFunction, err = endless.CoinBatchSafeTransferPayload(
		nil,
		[]endless.AccountAddress{
			recipient1.Address,
			recipient2.Address,
		},
		[]uint64{
			TransferAmount,
			TransferAmount * 2,
		},
		digestHash2,
	)

	if err != nil {
		panic("Failed to build transfer payload:" + err.Error())
	}

	// 4.Sign and Submit transaction
	resp, err = client.BuildSignAndSubmitTransaction(
		sender,
		endless.TransactionPayload{
			Payload: safeEntryFunction,
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

	// 5. Wait for the transaction to complete
	senderBalance, err = client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipient1Balance, err = client.AccountEDSBalance(recipient1.Address)
	if err != nil {
		panic("Failed to retrieve recipient1 balance:" + err.Error())
	}

	senderBalance, err = client.AccountEDSBalance(sender.Address)
	if err != nil {
		panic("Failed to retrieve sender balance:" + err.Error())
	}
	recipient1Balance, err = client.AccountEDSBalance(recipient1.Address)
	if err != nil {
		panic("Failed to retrieve recipient1 balance:" + err.Error())
	}
	recipient2Balance, err = client.AccountEDSBalance(recipient2.Address)
	if err != nil {
		panic("Failed to retrieve recipient2 balance:" + err.Error())
	}
	fmt.Printf("\n================ Intermediate Balances ================\n")
	fmt.Printf("sender EDS: %d\n", senderBalance)
	fmt.Printf("recipient1 EDS: %d\n", recipient1Balance)
	fmt.Printf("recipient2 EDS: %d\n", recipient2Balance)
}

func main() {
	example(endless.TestnetConfig)
}
