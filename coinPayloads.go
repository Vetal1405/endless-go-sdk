package endless

import (
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/endless-labs/endless-go-sdk/bcs"
	"math/big"
)

// CoinTransferPayload builds an EntryFunction payload for transferring coins
//
// Args:
//   - coinType is the type of coin to transfer. If none is provided, it will transfer EDS
//   - dest is the destination [AccountAddress]
//   - amount is the amount of coins to transfer
func CoinTransferPayload(coinAddress *string, dest AccountAddress, amount uint64) (payload *EntryFunction, err error) {
	amountBytes, err := bcs.SerializeU128(*big.NewInt(int64(amount)))
	if err != nil {
		return nil, err
	}

	if coinAddress == nil {
		return &EntryFunction{
			Module: ModuleId{
				Address: AccountOne,
				Name:    "endless_account",
			},
			Function: "transfer",
			ArgTypes: []TypeTag{},
			Args: [][]byte{
				dest[:],
				amountBytes,
			},
		}, nil
	} else {
		return &EntryFunction{
			Module: ModuleId{
				Address: AccountOne,
				Name:    "endless_account",
			},
			Function: "transfer_coins",
			ArgTypes: []TypeTag{
				EndlessCoinTypeTag,
			},
			Args: [][]byte{
				dest[:],
				amountBytes,
				base58.Decode(*coinAddress),
			},
		}, nil
	}
}

// CoinBatchTransferPayload builds an EntryFunction payload for transferring coins to multiple receivers
//
// Args:
//   - coinType is the type of coin to transfer. If none is provided, it will transfer EDS
//   - dests are the destination [AccountAddress]s
//   - amounts are the amount of coins to transfer per destination
func CoinBatchTransferPayload(coinAddress *string, dests []AccountAddress, amounts []uint64) (payload *EntryFunction, err error) {
	destBytes, err := bcs.SerializeSequenceOnly(dests) //destBytes[0]=2  	destBytes[1-32]=AccountAddress 		destBytes[33-64]=AccountAddress
	if err != nil {
		return nil, err
	}

	amountsBytes, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
		bcs.SerializeSequenceWithFunction(amounts, ser, func(ser *bcs.Serializer, amount uint64) {
			ser.U128(*big.NewInt(int64(amount)))
		})
	})
	if err != nil {
		return nil, err
	}

	if coinAddress == nil {
		return &EntryFunction{
			Module: ModuleId{
				Address: AccountOne,
				Name:    "endless_account",
			},
			Function: "batch_transfer",
			ArgTypes: []TypeTag{},
			Args: [][]byte{
				destBytes,
				amountsBytes,
			},
		}, nil
	} else {
		return &EntryFunction{
			Module: ModuleId{
				Address: AccountOne,
				Name:    "endless_account",
			},
			Function: "batch_transfer_coins",
			ArgTypes: []TypeTag{
				EndlessCoinTypeTag,
			},
			Args: [][]byte{
				destBytes,
				amountsBytes,
				base58.Decode(*coinAddress),
			},
		}, nil
	}
}

// CoinSafeTransferPayload builds an SafeEntryFunction payload for transferring coins
//
// Args:
//   - coinType is the type of coin to transfer. If none is provided, it will transfer EDS
//   - dest is the destination [AccountAddress]
//   - amount is the amount of coins to transfer
//   - hash is check hash
func CoinSafeTransferPayload(coinAddress *string, dest AccountAddress, amount uint64, hash [32]byte) (payload *SafeEntryFunction, err error) {
	amountBytes, err := bcs.SerializeU128(*big.NewInt(int64(amount)))
	if err != nil {
		return nil, err
	}

	if coinAddress == nil {
		return &SafeEntryFunction{
			Module: ModuleId{
				Address: AccountOne,
				Name:    "endless_account",
			},
			Function: "transfer",
			ArgTypes: []TypeTag{},
			Args: [][]byte{
				dest[:],
				amountBytes,
			},
			Hash: hash,
		}, nil
	} else {
		return &SafeEntryFunction{
			Module: ModuleId{
				Address: AccountOne,
				Name:    "endless_account",
			},
			Function: "transfer_coins",
			ArgTypes: []TypeTag{
				EndlessCoinTypeTag,
			},
			Args: [][]byte{
				dest[:],
				amountBytes,
				base58.Decode(*coinAddress),
			},
			Hash: hash,
		}, nil
	}
}

// CoinBatchSafeTransferPayload builds an EntryFunction payload for transferring coins to multiple receivers
//
// Args:
//   - coinType is the type of coin to transfer. If none is provided, it will transfer EDS
//   - dests are the destination [AccountAddress]s
//   - amounts are the amount of coins to transfer per destination
func CoinBatchSafeTransferPayload(coinAddress *string, dests []AccountAddress, amounts []uint64, hash [32]byte) (payload *SafeEntryFunction, err error) {
	destBytes, err := bcs.SerializeSequenceOnly(dests) //destBytes[0]=2  	destBytes[1-32]=AccountAddress 		destBytes[33-64]=AccountAddress
	if err != nil {
		return nil, err
	}

	amountsBytes, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
		bcs.SerializeSequenceWithFunction(amounts, ser, func(ser *bcs.Serializer, amount uint64) {
			ser.U128(*big.NewInt(int64(amount)))
		})
	})
	if err != nil {
		return nil, err
	}

	if coinAddress == nil {
		return &SafeEntryFunction{
			Module: ModuleId{
				Address: AccountOne,
				Name:    "endless_account",
			},
			Function: "batch_transfer",
			ArgTypes: []TypeTag{},
			Args: [][]byte{
				destBytes,
				amountsBytes,
			},
			Hash: hash,
		}, nil
	} else {
		return &SafeEntryFunction{
			Module: ModuleId{
				Address: AccountOne,
				Name:    "endless_account",
			},
			Function: "batch_transfer_coins",
			ArgTypes: []TypeTag{
				EndlessCoinTypeTag,
			},
			Args: [][]byte{
				destBytes,
				amountsBytes,
				base58.Decode(*coinAddress),
			},
			Hash: hash,
		}, nil
	}
}
