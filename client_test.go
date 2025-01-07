package endless

import (
	"github.com/btcsuite/btcd/btcutil/base58"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/endless-labs/endless-go-sdk/api"
	"github.com/stretchr/testify/assert"
)

const (
	vmStatusSuccess = "Executed successfully"
)

type CreateSigner func() (TransactionSigner, error)

var TestSigners map[string]CreateSigner

type CreateSingleSignerPayload func(client *Client, sender TransactionSigner, options ...any) (*RawTransaction, error)

var TestSingleSignerPayloads map[string]CreateSingleSignerPayload

func init() {
	initSigners()
	initSingleSignerPayloads()
}

func initSigners() {
	TestSigners = make(map[string]CreateSigner)

	TestSigners["Standard Ed25519"] = func() (TransactionSigner, error) {
		signer, err := NewEd25519Account()
		return any(signer).(TransactionSigner), err
	}
	TestSigners["Single Sender Ed25519"] = func() (TransactionSigner, error) {
		signer, err := NewEd25519SingleSenderAccount()
		return any(signer).(TransactionSigner), err
	}
	TestSigners["Single Sender Secp256k1"] = func() (TransactionSigner, error) {
		signer, err := NewSecp256k1Account()
		return any(signer).(TransactionSigner), err
	}

	TestSigners["2-of-3 MultiKey"] = func() (TransactionSigner, error) {
		signer, err := NewMultiKeyTestSigner(3, 2)
		return any(signer).(TransactionSigner), err
	}

	/* TODO: MultiEd25519 is not supported ATM
	TestSigners["MultiEd25519"] = func() (TransactionSigner, error) {
		signer, err := NewMultiEd25519Signer(3, 2)
		return any(signer).(TransactionSigner), err
	}
	*/
}

func initSingleSignerPayloads() {
	TestSingleSignerPayloads = make(map[string]CreateSingleSignerPayload)
	TestSingleSignerPayloads["Entry Function"] = buildSingleSignerEntryFunction
}

func buildSingleSignerEntryFunction(client *Client, sender TransactionSigner, options ...any) (*RawTransaction, error) {
	return EDSTransferTransaction(client, sender, AccountOne, 100, options...)
}

func Test_Info(t *testing.T) {
	client, err := createTestClient()
	assert.NoError(t, err)

	info, err := client.Info()
	assert.NoError(t, err)
	assert.Greater(t, info.BlockHeight(), uint64(0))
}
func TestNamedConfig(t *testing.T) {
	names := []string{"mainnet", "testnet"}
	for _, name := range names {
		assert.Equal(t, name, NamedNetworks[name].Name)
	}
}

func TestClient_NodeAPIHealthCheck(t *testing.T) {
	client, err := createTestClient()
	assert.NoError(t, err)

	t.Run("Node API health check default", func(t *testing.T) {
		t.Parallel()
		response, err := client.NodeAPIHealthCheck()
		assert.NoError(t, err)
		assert.True(t, strings.Contains(response.Message, "ok"), "Node API health check failed"+response.Message)
	})

	// Now, check node API health check with a future time that should never fail
	t.Run("Node API health check far future", func(t *testing.T) {
		t.Parallel()
		response, err := client.NodeAPIHealthCheck(10000)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(response.Message, "ok"), "Node API health check failed"+response.Message)
	})

	// Now, check node API health check with 0
	t.Run("Node API health check fail", func(t *testing.T) {
		t.Parallel()
		// Now, check node API health check with a time that should probably fail
		_, err := client.NodeAPIHealthCheck(0)
		assert.Error(t, err)
	})
}

func TestClient_BlockByHeight(t *testing.T) {
	client, err := createTestClient()
	assert.NoError(t, err)

	_, err = client.BlockByHeight(1, true)
	assert.NoError(t, err)
}

func TestClient_View(t *testing.T) {
	client, err := createTestClient()
	assert.NoError(t, err)

	payload := &ViewPayload{
		Module: ModuleId{
			Address: AccountOne,
			Name:    "primary_fungible_store",
		},
		Function: "balance",
		ArgTypes: []TypeTag{
			EndlessCoinTypeTag,
		},
		Args: [][]byte{
			AccountOne[:],
			base58.Decode(EndlessCoin),
		},
	}
	vals, err := client.View(payload)
	assert.NoError(t, err)
	assert.Len(t, vals, 1)
	_, err = StrToBigInt(vals[0].(string))
	assert.NoError(t, err)
}

func TestEndlessClientHeaderValue(t *testing.T) {
	assert.Greater(t, len(ClientHeaderValue), 0)
	assert.NotEqual(t, "endless-go-sdk/unk", ClientHeaderValue)
}

func Test_SingleSignerFlows(t *testing.T) {
	for name, signer := range TestSigners {
		for payloadName, buildSingleSignerPayload := range TestSingleSignerPayloads {
			t.Run(name+" "+payloadName, func(t *testing.T) {
				singleSignerFlowsTransaction(t, signer, buildSingleSignerPayload)
			})
			t.Run(name+" "+payloadName+" simulation", func(t *testing.T) {
				singleSignerFlowsSimulation(t, signer, buildSingleSignerPayload)
			})
		}
	}
}
func singleSignerFlowsFaucet(t *testing.T, createAccount CreateSigner) (*Client, TransactionSigner) {
	// All of these run against localnet
	if testing.Short() {
		t.Skip("integration test expects network connection to localnet")
	}
	// Create a client
	client, err := createTestClient()
	assert.NoError(t, err)

	// Verify chain id retrieval works
	_, err = client.GetChainId()
	assert.NoError(t, err)

	// Verify gas estimation works
	_, err = client.EstimateGasPrice()
	assert.NoError(t, err)

	// Create an account
	account, err := createAccount()
	assert.NoError(t, err)

	// Fund the account with 1 EDS
	err = client.Faucet(
		Account{
			Address: account.AccountAddress(),
			Signer:  account,
		},
		SequenceNumber(0),
	)
	assert.NoError(t, err)

	return client, account
}
func singleSignerFlowsTransaction(t *testing.T, createAccount CreateSigner, buildTransaction CreateSingleSignerPayload) {
	client, account := singleSignerFlowsFaucet(t, createAccount)

	// Build transaction
	rawTxn, err := buildTransaction(client, account)
	assert.NoError(t, err)

	// Sign transaction
	signedTxn, err := rawTxn.SignedTransaction(account)
	assert.NoError(t, err)

	// Send transaction
	result, err := client.SubmitTransaction(signedTxn)
	assert.NoError(t, err)

	hash := result.Hash

	// Wait for the transaction
	_, err = client.WaitForTransaction(hash)
	assert.NoError(t, err)

	// Read transaction by hash
	txn, err := client.TransactionByHash(hash)
	assert.NoError(t, err)

	// Read transaction by version
	userTxn, _ := txn.Inner.(*api.UserTransaction)
	version := userTxn.Version

	// Load the transaction again
	txnByVersion, err := client.TransactionByVersion(version)
	assert.NoError(t, err)

	// Assert that both are the same
	expectedTxn, err := txn.UserTransaction()
	assert.NoError(t, err)

	actualTxn, err := txnByVersion.UserTransaction()
	assert.NoError(t, err)

	assert.Equal(t, expectedTxn, actualTxn)
}
func singleSignerFlowsSimulation(t *testing.T, createAccount CreateSigner, buildTransaction CreateSingleSignerPayload) {
	client, account := singleSignerFlowsFaucet(t, createAccount)

	// Simulate transaction (no options)
	rawTxn, err := buildTransaction(client, account)
	assert.NoError(t, err)

	simulatedTxn, err := client.SimulateTransaction(rawTxn, account)
	switch account.(type) {
	case *MultiKeyTestSigner:
		// multikey simulation currently not supported
		assert.Error(t, err)
		assert.ErrorContains(t, err, "currently unsupported sender derivation scheme")
		return // skip rest of the tests
	default:
		assert.NoError(t, err)
		assert.Equal(t, true, simulatedTxn[0].Success)
		assert.Equal(t, vmStatusSuccess, simulatedTxn[0].VmStatus)
		assert.Greater(t, simulatedTxn[0].GasUsed, uint64(0))
	}

	// simulate transaction (estimate gas unit price)
	rawTxnZeroGasUnitPrice, err := buildTransaction(client, account, GasUnitPrice(0))
	assert.NoError(t, err)
	simulatedTxn, err = client.SimulateTransaction(rawTxnZeroGasUnitPrice, account, EstimateGasUnitPrice(true))
	assert.NoError(t, err)
	assert.Equal(t, true, simulatedTxn[0].Success)
	assert.Equal(t, vmStatusSuccess, simulatedTxn[0].VmStatus)
	estimatedGasUnitPrice := simulatedTxn[0].GasUnitPrice
	assert.Greater(t, estimatedGasUnitPrice, uint64(0))

	// simulate transaction (estimate max gas amount)
	rawTxnZeroMaxGasAmount, err := buildTransaction(client, account, MaxGasAmount(0))
	assert.NoError(t, err)
	simulatedTxn, err = client.SimulateTransaction(rawTxnZeroMaxGasAmount, account, EstimateMaxGasAmount(true))
	assert.NoError(t, err)
	assert.Equal(t, true, simulatedTxn[0].Success)
	assert.Equal(t, vmStatusSuccess, simulatedTxn[0].VmStatus)
	assert.Greater(t, simulatedTxn[0].MaxGasAmount, uint64(0))

	// simulate transaction (estimate prioritized gas unit price and max gas amount)
	rawTxnZeroGasConfig, err := buildTransaction(client, account, GasUnitPrice(0), MaxGasAmount(0))
	assert.NoError(t, err)
	simulatedTxn, err = client.SimulateTransaction(rawTxnZeroGasConfig, account, EstimatePrioritizedGasUnitPrice(true), EstimateMaxGasAmount(true))
	assert.NoError(t, err)
	assert.Equal(t, true, simulatedTxn[0].Success)
	assert.Equal(t, vmStatusSuccess, simulatedTxn[0].VmStatus)
	estimatedGasUnitPrice = simulatedTxn[0].GasUnitPrice
	assert.Greater(t, estimatedGasUnitPrice, uint64(0))
	assert.Greater(t, simulatedTxn[0].MaxGasAmount, uint64(0))
}

func TestEDSTransferTransaction(t *testing.T) {
	sender, err := NewEd25519Account()
	assert.NoError(t, err)
	dest, err := NewEd25519Account()
	assert.NoError(t, err)

	client, err := createTestClient()
	assert.NoError(t, err)

	signedTxn, err := EDSTransferTransaction(client, sender, dest.Address, 1337, MaxGasAmount(123123), GasUnitPrice(111), ExpirationSeconds(time.Now().Unix()+42), ChainIdOption(71), SequenceNumber(31337))
	assert.NoError(t, err)
	assert.NotNil(t, signedTxn)

	// use defaults for: max gas amount, gas unit price
	signedTxn, err = EDSTransferTransaction(client, sender, dest.Address, 1337, ExpirationSeconds(time.Now().Unix()+42), ChainIdOption(71), SequenceNumber(31337))
	assert.NoError(t, err)
	assert.NotNil(t, signedTxn)
}

func Test_Genesis(t *testing.T) {
	client, err := createTestClient()
	assert.NoError(t, err)

	genesis, err := client.BlockByHeight(0, true)
	assert.NoError(t, err)

	txn, err := genesis.Transactions[0].GenesisTransaction()
	assert.NoError(t, err)

	assert.Equal(t, uint64(0), *txn.TxnVersion())
}

func Test_Block(t *testing.T) {
	client, err := createTestClient()
	assert.NoError(t, err)
	info, err := client.Info()
	assert.NoError(t, err)

	// TODO: I need to add hardcoded testing sets for these conversions
	numToCheck := uint64(10)
	blockHeight := info.BlockHeight()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(int(numToCheck))

	for i := uint64(0); i < numToCheck; i++ {
		go func() {
			blockNumber := blockHeight - i
			blockByHeight, err := client.BlockByHeight(blockNumber, true)
			assert.NoError(t, err)

			assert.Equal(t, blockNumber, blockByHeight.BlockHeight)

			// Block should always be last - first + 1 (since they would be 1 if they're the same (inclusive)
			assert.Equal(t, 1+blockByHeight.LastVersion-blockByHeight.FirstVersion, uint64(len(blockByHeight.Transactions)))

			// Version should be the same
			blockByVersion, err := client.BlockByVersion(blockByHeight.FirstVersion, true)
			assert.NoError(t, err)

			assert.Equal(t, blockByHeight, blockByVersion)
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()
}

func Test_Account(t *testing.T) {
	client, err := createTestClient()
	assert.NoError(t, err)
	account, err := client.Account(AccountOne)
	assert.NoError(t, err)

	sequenceNumber, err := account.SequenceNumber()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), sequenceNumber)
	authKey, err := account.AuthenticationKey()
	assert.NoError(t, err)
	assert.Equal(t, AccountOne[:], authKey[0][:])
}

func Test_Transactions(t *testing.T) {
	client, err := createTestClient()
	assert.NoError(t, err)

	start := uint64(1)
	count := uint64(2)
	// Specific 2 should only give 2
	transactions, err := client.Transactions(&start, &count)
	assert.NoError(t, err)
	assert.Len(t, transactions, 2)

	// This will give the latest 2
	transactions, err = client.Transactions(nil, &count)
	assert.NoError(t, err)
	assert.Len(t, transactions, 2)

	// This will give the 25 from 2
	transactions, err = client.Transactions(&start, nil)
	assert.NoError(t, err)
	assert.Len(t, transactions, 25)

	// This will give the latest 25
	transactions, err = client.Transactions(nil, nil)
	assert.NoError(t, err)
	assert.Len(t, transactions, 25)
}

func Test_AccountTransactions(t *testing.T) {
	t.Parallel()
	client, err := createTestClient()
	assert.NoError(t, err)

	// Create a bunch of transactions so we can test the pagination
	account, err := NewEd25519Account()
	assert.NoError(t, err)
	err = client.Faucet(*account, SequenceNumber(0))
	assert.NoError(t, err)

	println("account =", account.Address.String())

	// Build and submit 100 transactions
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(100)
	for i := 1; i < 101; i++ {
		go func(seqNo int) {
			accountTransactionsSubmit(t, client, account, uint64(seqNo))
			waitGroup.Done()
		}(i)
	}
	waitGroup.Wait()

	// Submit one more transaction
	accountTransactionsSubmit(t, client, account, 101)

	// Fetch default
	transactions, err := client.AccountTransactions(account.AccountAddress(), nil, nil)
	assert.NoError(t, err)
	assert.Len(t, transactions, 25)

	// Fetch 101 with no start
	zero := uint64(0)
	one := uint64(1)
	ten := uint64(10)
	hundredOne := uint64(101)

	transactions, err = client.AccountTransactions(account.AccountAddress(), nil, &hundredOne)
	assert.NoError(t, err)
	assert.Len(t, transactions, 101)

	// Fetch 101 with start
	transactions, err = client.AccountTransactions(account.AccountAddress(), &zero, &hundredOne)
	assert.NoError(t, err)
	assert.Len(t, transactions, 101)

	// Fetch 100 from 1
	transactions, err = client.AccountTransactions(account.AccountAddress(), &one, &hundredOne)
	assert.NoError(t, err)
	assert.Len(t, transactions, 101) //todo

	// Fetch default from 0
	transactions, err = client.AccountTransactions(account.AccountAddress(), &zero, nil)
	assert.NoError(t, err)
	assert.Len(t, transactions, 25)

	// Check global transactions API

	t.Run("Default transaction size, no start", func(t *testing.T) {
		transactions, err = client.Transactions(nil, nil)
		assert.NoError(t, err)
		assert.Len(t, transactions, 25)
	})
	t.Run("Default transaction size, start from zero", func(t *testing.T) {
		transactions, err = client.Transactions(&zero, nil)
		assert.NoError(t, err)
		assert.Len(t, transactions, 25)
	})
	t.Run("Default transaction size, start from one", func(t *testing.T) {
		transactions, err = client.Transactions(&one, nil)
		assert.NoError(t, err)
		assert.Len(t, transactions, 25)
	})

	t.Run("101 transactions, no start", func(t *testing.T) {
		transactions, err = client.Transactions(nil, &hundredOne)
		assert.NoError(t, err)
		assert.Len(t, transactions, 101)
	})

	t.Run("101 transactions, start zero", func(t *testing.T) {
		transactions, err = client.Transactions(&zero, &hundredOne)
		assert.NoError(t, err)
		assert.Len(t, transactions, 101)
	})

	t.Run("101 transactions, start one", func(t *testing.T) {
		transactions, err = client.Transactions(&one, &hundredOne)
		assert.NoError(t, err)
		assert.Len(t, transactions, 101)
	})

	t.Run("10 transactions, no start", func(t *testing.T) {
		transactions, err = client.Transactions(nil, &ten)
		assert.NoError(t, err)
		assert.Len(t, transactions, 10)
	})

	t.Run("10 transactions, start one", func(t *testing.T) {
		transactions, err = client.Transactions(&one, &ten)
		assert.NoError(t, err)
		assert.Len(t, transactions, 10)
	})
}
func accountTransactionsSubmit(t *testing.T, client *Client, account *Account, seqNo uint64) {
	payload, err := CoinTransferPayload(nil, AccountOne, 1)
	assert.NoError(t, err)
	rawTxn, err := client.BuildTransaction(account.AccountAddress(), TransactionPayload{Payload: payload}, SequenceNumber(seqNo))
	assert.NoError(t, err)
	signedTxn, err := rawTxn.SignedTransaction(account)
	assert.NoError(t, err)
	txn, err := client.SubmitTransaction(signedTxn)
	assert.NoError(t, err)
	_, err = client.WaitForTransaction(txn.Hash)
	assert.NoError(t, err)
}

func Test_AccountResources(t *testing.T) {
	client, err := createTestClient()
	assert.NoError(t, err)

	resources, err := client.AccountResources(AccountOne)
	assert.NoError(t, err)
	assert.Greater(t, len(resources), 0)

	resourcesBcs, err := client.AccountResourcesBCS(AccountOne)
	assert.NoError(t, err)
	assert.Greater(t, len(resourcesBcs), 0)
}

func Test_Concurrent_Submission(t *testing.T) {
	const numTxns = uint64(100)
	const numWaiters = 4

	client, err := createTestClient()
	assert.NoError(t, err)

	account1, err := NewEd25519Account()
	assert.NoError(t, err)
	err = client.Faucet(*account1, SequenceNumber(0))
	assert.NoError(t, err)

	// start submission goroutine
	payloads := make(chan TransactionBuildPayload, 50)
	results := make(chan TransactionSubmissionResponse, 50)
	go client.BuildSignAndSubmitTransactions(account1, payloads, results, ExpirationSeconds(time.Now().Unix()+60))

	transferPayload, err := CoinTransferPayload(nil, AccountOne, 100)
	assert.NoError(t, err)

	// Generate transactions
	for i := uint64(0); i < numTxns; i++ {
		payloads <- TransactionBuildPayload{
			Id:   i,
			Type: TransactionSubmissionTypeSingle, // TODO: not needed?
			Inner: TransactionPayload{
				Payload: transferPayload,
			},
		}
	}
	close(payloads)
	t.Log("done submitting txns")

	// Start waiting on txns
	waitResults := make(chan ConcResponse[*api.UserTransaction], numWaiters*10)

	var wg sync.WaitGroup
	wg.Add(numWaiters)
	for range numWaiters {
		go concurrentTxnWaiter(results, waitResults, client, t, &wg)
	}

	// Wait on all the results, recording the succeeding ones
	txnMap := make(map[uint64]bool)

	waitersRunning := numWaiters

	// We could wait on a close, but I'm going to be a little pickier here
	i := uint64(0)
	txnGoodEvents := 0
	for {
		response := <-waitResults
		if response.Err == nil && response.Result == nil {
			t.Log("txn waiter signaled done")
			waitersRunning--
			if waitersRunning == 0 {
				close(results)
				t.Log("last txn waiter done")
				break
			}
			continue
		}
		assert.NoError(t, response.Err)
		assert.True(t, (response.Result != nil) && response.Result.Success)
		if response.Result != nil {
			txnMap[response.Result.SequenceNumber] = true
			txnGoodEvents++
		}
		i++
		if i >= numTxns {
			t.Logf("waited on %d txns, done", i)
			break
		}
	}
	t.Log("done waiting for txns, waiting for txn waiter threads")

	wg.Wait()

	// Check all transactions were successful from [0-numTxns)
	t.Logf("got %d(%d) successful txns of %d attempted, error submission indexes:", len(txnMap), txnGoodEvents, numTxns)

	allTrue := true
	for i := uint64(1); i < numTxns+1; i++ {
		allTrue = allTrue && txnMap[i]
		if !txnMap[i] {
			t.Logf("%d", i)
		}
	}
	assert.True(t, allTrue, "all txns successful")
	assert.Equal(t, len(txnMap), int(numTxns), "num txns successful == num txns sent")
}

// A worker thread that reads from a chan of transactions that have been submitted and waits on their completion status
func concurrentTxnWaiter(
	results chan TransactionSubmissionResponse,
	waitResults chan ConcResponse[*api.UserTransaction],
	client *Client,
	t *testing.T,
	wg *sync.WaitGroup,
) {
	if wg != nil {
		defer wg.Done()
	}
	responseCount := 0
	for response := range results {
		responseCount++
		assert.NoError(t, response.Err)

		waitResponse, err := client.WaitForTransaction(response.Response.Hash, PollTimeout(21*time.Second))
		if err != nil {
			t.Logf("%s err %s", response.Response.Hash, err)
		} else if waitResponse == nil {
			t.Logf("%s nil response", response.Response.Hash)
		} else if !waitResponse.Success {
			t.Logf("%s !Success", response.Response.Hash)
		}
		waitResults <- ConcResponse[*api.UserTransaction]{Result: waitResponse, Err: err}
	}
	t.Logf("concurrentTxnWaiter done, %d responses", responseCount)
	// signal completion
	// (do not close the output as there may be other workers writing to it)
	waitResults <- ConcResponse[*api.UserTransaction]{Result: nil, Err: nil}
}
