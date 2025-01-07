package endless

import (
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/endless-labs/endless-go-sdk/api"
)

// NetworkConfig a configuration for the Client and which network to use.  Use one of the preconfigured  [TestnetConfig], or [MainnetConfig] unless you have your own full node.
//
// Name, ChainId, IndexerUrl are not required.
//
// If ChainId is 0, the ChainId wil be fetched on-chain
// If IndexerUrl or FaucetUrl are an empty string "", clients will not be made for them.
type NetworkConfig struct {
	Name       string
	ChainId    uint8
	NodeUrl    string
	IndexerUrl string
}

// TestnetConfig is for use with testnet. Testnet does not reset.
var TestnetConfig = NetworkConfig{
	Name:    "testnet",
	ChainId: 221,
	NodeUrl: "https://rpc-test.endless.link/v1",
}

// MainnetConfig is for use with mainnet.  There is no singleSignerFlowsFaucet for Mainnet, as these are real user assets.
var MainnetConfig = NetworkConfig{
	Name:    "mainnet",
	ChainId: 220,
	NodeUrl: "https://rpc.endless.link/v1",
}

// NamedNetworks Map from network name to NetworkConfig
var NamedNetworks map[string]NetworkConfig

func init() {
	NamedNetworks = make(map[string]NetworkConfig, 4)
	setNN := func(nc NetworkConfig) {
		NamedNetworks[nc.Name] = nc
	}
	setNN(TestnetConfig)
	setNN(MainnetConfig)
}

// EndlessClient is an interface for all functionality on the Client.
// It is a combination of [EndlessRpcClient], [EndlessIndexerClient], and [EndlessFaucetClient] for the purposes
// of mocking and convenince.
type EndlessClient interface {
	EndlessFaucetClient
	EndlessRpcClient
}

// EndlessFaucetClient is an interface for all functionality on the Client that is Faucet related.  Its main implementation
// is [FaucetClient]
type EndlessFaucetClient interface {
	// Fund Uses the singleSignerFlowsFaucet to fund an address, only applies to non-production networks
	Fund(address AccountAddress, amount uint64) error
}

// EndlessRpcClient is an interface for all functionality on the Client that is Node RPC related.  Its main implementation
// is [NodeClient]
type EndlessRpcClient interface {
	// SetTimeout adjusts the HTTP client timeout
	//
	//	client.SetTimeout(5 * time.Millisecond)
	SetTimeout(timeout time.Duration)

	// SetHeader sets the header for all future requests
	//
	//	client.SetHeader("Authorization", "Bearer abcde")
	SetHeader(key string, value string)

	// RemoveHeader removes the header from being automatically set all future requests.
	//
	//	client.RemoveHeader("Authorization")
	RemoveHeader(key string)

	// Info Retrieves the node info about the network and it's current state
	Info() (info NodeInfo, err error)

	// Account Retrieves information about the account such as [SequenceNumber] and [crypto.AuthenticationKey]
	Account(address AccountAddress, ledgerVersion ...uint64) (info AccountInfo, err error)

	// AccountResource Retrieves a single resource given its struct name.
	//
	//	address := AccountOne
	//	dataMap, _ := client.AccountResource(address, "0x1::coin::CoinStore")
	//
	// Can also fetch at a specific ledger version
	//
	//	address := AccountOne
	//	dataMap, _ := client.AccountResource(address, "0x1::coin::CoinStore", 1)
	AccountResource(address AccountAddress, resourceType string, ledgerVersion ...uint64) (data map[string]any, err error)

	// AccountResources fetches resources for an account into a JSON-like map[string]any in AccountResourceInfo.Data
	// For fetching raw Move structs as BCS, See #AccountResourcesBCS
	//
	//	address := AccountOne
	//	dataMap, _ := client.AccountResources(address)
	//
	// Can also fetch at a specific ledger version
	//
	//	address := AccountOne
	//	dataMap, _ := client.AccountResource(address, 1)
	AccountResources(address AccountAddress, ledgerVersion ...uint64) (resources []AccountResourceInfo, err error)

	// AccountResourcesBCS fetches account resources as raw Move struct BCS blobs in AccountResourceRecord.Data []byte
	AccountResourcesBCS(address AccountAddress, ledgerVersion ...uint64) (resources []AccountResourceRecord, err error)

	// BlockByHeight fetches a block by height
	//
	//	block, _ := client.BlockByHeight(1, false)
	//
	// Can also fetch with transactions
	//
	//	block, _ := client.BlockByHeight(1, true)
	BlockByHeight(blockHeight uint64, withTransactions bool) (data *api.Block, err error)

	// BlockByVersion fetches a block by ledger version
	//
	//	block, _ := client.BlockByVersion(123, false)
	//
	// Can also fetch with transactions
	//
	//	block, _ := client.BlockByVersion(123, true)
	BlockByVersion(ledgerVersion uint64, withTransactions bool) (data *api.Block, err error)

	// TransactionByHash gets info on a transaction
	// The transaction may be pending or recently committed.
	//
	//	data, err := client.TransactionByHash("0xabcd")
	//	if err != nil {
	//		if httpErr, ok := err.(endless.HttpError) {
	//			if httpErr.StatusCode == 404 {
	//				// if we're sure this has been submitted, assume it is still pending elsewhere in the mempool
	//			}
	//		}
	//	} else {
	//		if data["type"] == "pending_transaction" {
	//			// known to local mempool, but not committed yet
	//		}
	//	}
	TransactionByHash(txnHash string) (data *api.Transaction, err error)

	// TransactionByVersion gets info on a transaction from its LedgerVersion.  It must have been
	// committed to have a ledger version
	//
	//	data, err := client.TransactionByVersion("0xabcd")
	//	if err != nil {
	//		if httpErr, ok := err.(endless.HttpError) {
	//			if httpErr.StatusCode == 404 {
	//				// if we're sure this has been submitted, the full node might not be caught up to this version yet
	//			}
	//		}
	//	}
	TransactionByVersion(version uint64) (data *api.CommittedTransaction, err error)

	// PollForTransactions Waits up to 10 seconds for transactions to be done, polling at 10Hz
	// Accepts options PollPeriod and PollTimeout which should wrap time.Duration values.
	//
	//	hashes := []string{"0x1234", "0x4567"}
	//	err := client.PollForTransactions(hashes)
	//
	// Can additionally configure different options
	//
	//	hashes := []string{"0x1234", "0x4567"}
	//	err := client.PollForTransactions(hashes, PollPeriod(500 * time.Milliseconds), PollTimeout(5 * time.Seconds))
	PollForTransactions(txnHashes []string, options ...any) error

	// WaitForTransaction Do a long-GET for one transaction and wait for it to complete
	//
	//	data, err := client.WaitForTransaction("0x1234")
	WaitForTransaction(txnHash string, options ...any) (data *api.UserTransaction, err error)

	// Transactions Get recent transactions.
	// Start is a version number. Nil for most recent transactions.
	// Limit is a number of transactions to return. 'about a hundred' by default.
	//
	//	client.Transactions(0, 2)   // Returns 2 transactions
	//	client.Transactions(1, 100) // Returns 100 transactions
	Transactions(start *uint64, limit *uint64) (data []*api.CommittedTransaction, err error)

	// AccountTransactions Get transactions associated with an account.
	// Start is a version number. Nil for most recent transactions.
	// Limit is a number of transactions to return. 'about a hundred' by default.
	//
	//	client.AccountTransactions(AccountOne, 0, 2)   // Returns 2 transactions for 0x1
	//	client.AccountTransactions(AccountOne, 1, 100) // Returns 100 transactions for 0x1
	AccountTransactions(address AccountAddress, start *uint64, limit *uint64) (data []*api.CommittedTransaction, err error)

	// SubmitTransaction Submits an already signed transaction to the blockchain
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "endless_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
	//	signedTxn, _ := sender.SignTransaction(rawTxn)
	//	submitResponse, err := client.SubmitTransaction(signedTxn)
	SubmitTransaction(signedTransaction *SignedTransaction) (data *api.SubmitTransactionResponse, err error)

	// BatchSubmitTransaction submits a collection of signed transactions to the network in a single request
	//
	// It will return the responses in the same order as the input transactions that failed.  If the response is empty, then
	// all transactions succeeded.
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "endless_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
	//	signedTxn, _ := sender.SignTransaction(rawTxn)
	//	submitResponse, err := client.BatchSubmitTransaction([]*SignedTransaction{signedTxn})
	BatchSubmitTransaction(signedTxns []*SignedTransaction) (response *api.BatchSubmitTransactionResponse, err error)

	// SimulateTransaction Simulates a raw transaction without sending it to the blockchain
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "endless_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
	//	simResponse, err := client.SimulateTransaction(rawTxn, sender)
	SimulateTransaction(rawTxn *RawTransaction, sender TransactionSigner, options ...any) (data []*api.UserTransaction, err error)

	// GetChainId Retrieves the ChainId of the network
	// Note this will be cached forever, or taken directly from the config
	GetChainId() (chainId uint8, err error)

	// BuildTransaction Builds a raw transaction from the payload and fetches any necessary information from on-chain
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "endless_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, err := client.BuildTransaction(sender.AccountAddress(), txnPayload)
	BuildTransaction(sender AccountAddress, payload TransactionPayload, options ...any) (rawTxn *RawTransaction, err error)

	// BuildTransactionMultiAgent Builds a raw transaction for MultiAgent or FeePayer from the payload and fetches any necessary information from on-chain
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "endless_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, err := client.BuildTransactionMultiAgent(sender.AccountAddress(), txnPayload, FeePayer(AccountZero))
	BuildTransactionMultiAgent(sender AccountAddress, payload TransactionPayload, options ...any) (rawTxn *RawTransactionWithData, err error)

	// BuildSignAndSubmitTransaction Convenience function to do all three in one
	// for more configuration, please use them separately
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "endless_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	submitResponse, err := client.BuildSignAndSubmitTransaction(sender, txnPayload)
	BuildSignAndSubmitTransaction(sender *Account, payload TransactionPayload, options ...any) (data *api.SubmitTransactionResponse, err error)

	// View Runs a view function on chain returning a list of return values.
	//
	//	 address := AccountOne
	//		payload := &ViewPayload{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name:    "coin",
	//			},
	//			Function: "balance",
	//			ArgTypes: []TypeTag{EndlessCoinTypeTag},
	//			Args:     [][]byte{address[:]},
	//		}
	//		vals, err := client.endlessClient.View(payload)
	//		balance := StrToU64(vals.(any[])[0].(string))
	View(payload *ViewPayload, ledgerVersion ...uint64) (vals []any, err error)

	// EstimateGasPrice Retrieves the gas estimate from the network.
	EstimateGasPrice() (info EstimateGasInfo, err error)

	// AccountEDSBalance retrieves the EDS balance in the account
	AccountEDSBalance(address AccountAddress, ledgerVersion ...uint64) (uint64, error)

	// AccountCoinBalance retrieves the other coin balance in the account
	AccountCoinBalance(coinAddress string, address AccountAddress, ledgerVersion ...uint64) (uint64, error)

	// NodeAPIHealthCheck checks if the node is within durationSecs of the current time, if not provided the node default is used
	NodeAPIHealthCheck(durationSecs ...uint64) (api.HealthCheckResponse, error)

	Faucet(account Account, options ...any) error
}

// Client is a facade over the multiple types of underlying clients, as the user doesn't actually care where the data
// comes from.  It will be then handled underneath
//
// To create a new client, please use [NewClient].  An example below for Testnet:
//
//	client := NewClient(TestnetConfig)
//
// Implements EndlessClient
type Client struct {
	nodeClient    *NodeClient
	indexerClient *IndexerClient
}

// NewClient Creates a new client with a specific network config that can be extended in the future
func NewClient(config NetworkConfig, options ...any) (client *Client, err error) {
	var httpClient *http.Client = nil
	for i, arg := range options {
		switch value := arg.(type) {
		case *http.Client:
			if httpClient != nil {
				err = fmt.Errorf("NewClient only accepts one http.Client")
				return
			}
			httpClient = value
		default:
			err = fmt.Errorf("NewClient arg %d bad type %T", i+1, arg)
			return
		}
	}
	var nodeClient *NodeClient
	if httpClient == nil {
		nodeClient, err = NewNodeClient(config.NodeUrl, config.ChainId)
	} else {
		nodeClient, err = NewNodeClientWithHttpClient(config.NodeUrl, config.ChainId, httpClient)
	}
	if err != nil {
		return nil, err
	}
	// Indexer may not be present
	var indexerClient *IndexerClient = nil
	if config.IndexerUrl != "" {
		indexerClient = NewIndexerClient(nodeClient.client, config.IndexerUrl)
	}

	// Fetch the chain Id if it isn't in the config
	if config.ChainId == 0 {
		_, _ = nodeClient.GetChainId()
	}

	client = &Client{
		nodeClient,
		indexerClient,
	}
	return
}

// SetTimeout adjusts the HTTP client timeout
//
//	client.SetTimeout(5 * time.Millisecond)
func (client *Client) SetTimeout(timeout time.Duration) {
	client.nodeClient.SetTimeout(timeout)
}

// SetHeader sets the header for all future requests
//
//	client.SetHeader("Authorization", "Bearer abcde")
func (client *Client) SetHeader(key string, value string) {
	client.nodeClient.SetHeader(key, value)
}

// RemoveHeader removes the header from being automatically set all future requests.
//
//	client.RemoveHeader("Authorization")
func (client *Client) RemoveHeader(key string) {
	client.nodeClient.RemoveHeader(key)
}

// Info Retrieves the node info about the network and it's current state
func (client *Client) Info() (info NodeInfo, err error) {
	return client.nodeClient.Info()
}

// Account Retrieves information about the account such as [SequenceNumber] and [crypto.AuthenticationKey]
func (client *Client) Account(address AccountAddress, ledgerVersion ...uint64) (info AccountInfo, err error) {
	return client.nodeClient.Account(address, ledgerVersion...)
}

// AccountResource Retrieves a single resource given its struct name.
//
//	address := AccountOne
//	dataMap, _ := client.AccountResource(address, "0x1::coin::CoinStore")
//
// Can also fetch at a specific ledger version
//
//	address := AccountOne
//	dataMap, _ := client.AccountResource(address, "0x1::coin::CoinStore", 1)
func (client *Client) AccountResource(address AccountAddress, resourceType string, ledgerVersion ...uint64) (data map[string]any, err error) {
	return client.nodeClient.AccountResource(address, resourceType, ledgerVersion...)
}

// AccountResources fetches resources for an account into a JSON-like map[string]any in AccountResourceInfo.Data
// For fetching raw Move structs as BCS, See #AccountResourcesBCS
//
//	address := AccountOne
//	dataMap, _ := client.AccountResources(address)
//
// Can also fetch at a specific ledger version
//
//	address := AccountOne
//	dataMap, _ := client.AccountResource(address, 1)
func (client *Client) AccountResources(address AccountAddress, ledgerVersion ...uint64) (resources []AccountResourceInfo, err error) {
	return client.nodeClient.AccountResources(address, ledgerVersion...)
}

// AccountResourcesBCS fetches account resources as raw Move struct BCS blobs in AccountResourceRecord.Data []byte
func (client *Client) AccountResourcesBCS(address AccountAddress, ledgerVersion ...uint64) (resources []AccountResourceRecord, err error) {
	return client.nodeClient.AccountResourcesBCS(address, ledgerVersion...)
}

// BlockByHeight fetches a block by height
//
//	block, _ := client.BlockByHeight(1, false)
//
// Can also fetch with transactions
//
//	block, _ := client.BlockByHeight(1, true)
func (client *Client) BlockByHeight(blockHeight uint64, withTransactions bool) (data *api.Block, err error) {
	return client.nodeClient.BlockByHeight(blockHeight, withTransactions)
}

// BlockByVersion fetches a block by ledger version
//
//	block, _ := client.BlockByVersion(123, false)
//
// Can also fetch with transactions
//
//	block, _ := client.BlockByVersion(123, true)
func (client *Client) BlockByVersion(ledgerVersion uint64, withTransactions bool) (data *api.Block, err error) {
	return client.nodeClient.BlockByVersion(ledgerVersion, withTransactions)
}

// TransactionByHash gets info on a transaction
// The transaction may be pending or recently committed.
//
//	data, err := client.TransactionByHash("0xabcd")
//	if err != nil {
//		if httpErr, ok := err.(endless.HttpError) {
//			if httpErr.StatusCode == 404 {
//				// if we're sure this has been submitted, assume it is still pending elsewhere in the mempool
//			}
//		}
//	} else {
//		if data["type"] == "pending_transaction" {
//			// known to local mempool, but not committed yet
//		}
//	}
func (client *Client) TransactionByHash(txnHash string) (data *api.Transaction, err error) {
	return client.nodeClient.TransactionByHash(txnHash)
}

// TransactionByVersion gets info on a transaction from its LedgerVersion.  It must have been
// committed to have a ledger version
//
//	data, err := client.TransactionByVersion("0xabcd")
//	if err != nil {
//		if httpErr, ok := err.(endless.HttpError) {
//			if httpErr.StatusCode == 404 {
//				// if we're sure this has been submitted, the full node might not be caught up to this version yet
//			}
//		}
//	}
func (client *Client) TransactionByVersion(version uint64) (data *api.CommittedTransaction, err error) {
	return client.nodeClient.TransactionByVersion(version)
}

// PollForTransactions Waits up to 10 seconds for transactions to be done, polling at 10Hz
// Accepts options PollPeriod and PollTimeout which should wrap time.Duration values.
//
//	hashes := []string{"0x1234", "0x4567"}
//	err := client.PollForTransactions(hashes)
//
// Can additionally configure different options
//
//	hashes := []string{"0x1234", "0x4567"}
//	err := client.PollForTransactions(hashes, PollPeriod(500 * time.Milliseconds), PollTimeout(5 * time.Seconds))
func (client *Client) PollForTransactions(txnHashes []string, options ...any) error {
	return client.nodeClient.PollForTransactions(txnHashes, options...)
}

// WaitForTransaction Do a long-GET for one transaction and wait for it to complete
//
//	data, err := client.WaitForTransaction("0x1234")
func (client *Client) WaitForTransaction(txnHash string, options ...any) (data *api.UserTransaction, err error) {
	return client.nodeClient.WaitForTransaction(txnHash, options...)
}

// Transactions Get recent transactions.
// Start is a version number. Nil for most recent transactions.
// Limit is a number of transactions to return. 'about a hundred' by default.
//
//	client.Transactions(0, 2)   // Returns 2 transactions
//	client.Transactions(1, 100) // Returns 100 transactions
func (client *Client) Transactions(start *uint64, limit *uint64) (data []*api.CommittedTransaction, err error) {
	return client.nodeClient.Transactions(start, limit)
}

// AccountTransactions Get transactions associated with an account.
// Start is a version number. Nil for most recent transactions.
// Limit is a number of transactions to return. 'about a hundred' by default.
//
//	client.AccountTransactions(AccountOne, 0, 2)   // Returns 2 transactions for 0x1
//	client.AccountTransactions(AccountOne, 1, 100) // Returns 100 transactions for 0x1
func (client *Client) AccountTransactions(address AccountAddress, start *uint64, limit *uint64) (data []*api.CommittedTransaction, err error) {
	return client.nodeClient.AccountTransactions(address, start, limit)
}

// SubmitTransaction Submits an already signed transaction to the blockchain
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "endless_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
//	signedTxn, _ := sender.SignTransaction(rawTxn)
//	submitResponse, err := client.SubmitTransaction(signedTxn)
func (client *Client) SubmitTransaction(signedTransaction *SignedTransaction) (data *api.SubmitTransactionResponse, err error) {
	return client.nodeClient.SubmitTransaction(signedTransaction)
}

// BatchSubmitTransaction submits a collection of signed transactions to the network in a single request
//
// It will return the responses in the same order as the input transactions that failed.  If the response is empty, then
// all transactions succeeded.
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "endless_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
//	signedTxn, _ := sender.SignTransaction(rawTxn)
//	submitResponse, err := client.BatchSubmitTransaction([]*SignedTransaction{signedTxn})
func (client *Client) BatchSubmitTransaction(signedTxns []*SignedTransaction) (response *api.BatchSubmitTransactionResponse, err error) {
	return client.nodeClient.BatchSubmitTransaction(signedTxns)
}

// SimulateTransaction Simulates a raw transaction without sending it to the blockchain
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "endless_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
//	simResponse, err := client.SimulateTransaction(rawTxn, sender)
func (client *Client) SimulateTransaction(rawTxn *RawTransaction, sender TransactionSigner, options ...any) (data []*api.UserTransaction, err error) {
	return client.nodeClient.SimulateTransaction(rawTxn, sender, options...)
}

// GetChainId Retrieves the ChainId of the network
// Note this will be cached forever, or taken directly from the config
func (client *Client) GetChainId() (chainId uint8, err error) {
	return client.nodeClient.GetChainId()
}

func (client *Client) Faucet(account Account, options ...any) error {
	return client.nodeClient.Faucet(account, options...)
}

// BuildTransaction Builds a raw transaction from the payload and fetches any necessary information from on-chain
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "endless_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, err := client.BuildTransaction(sender.AccountAddress(), txnPayload)
func (client *Client) BuildTransaction(sender AccountAddress, payload TransactionPayload, options ...any) (rawTxn *RawTransaction, err error) {
	return client.nodeClient.BuildTransaction(sender, payload, options...)
}

// BuildTransactionMultiAgent Builds a raw transaction for MultiAgent or FeePayer from the payload and fetches any necessary information from on-chain
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "endless_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, err := client.BuildTransactionMultiAgent(sender.AccountAddress(), txnPayload, FeePayer(AccountZero))
func (client *Client) BuildTransactionMultiAgent(sender AccountAddress, payload TransactionPayload, options ...any) (rawTxn *RawTransactionWithData, err error) {
	return client.nodeClient.BuildTransactionMultiAgent(sender, payload, options...)
}

// BuildSignAndSubmitTransaction Convenience function to do all three in one
// for more configuration, please use them separately
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "endless_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	submitResponse, err := client.BuildSignAndSubmitTransaction(sender, txnPayload)
func (client *Client) BuildSignAndSubmitTransaction(sender *Account, payload TransactionPayload, options ...any) (data *api.SubmitTransactionResponse, err error) {
	return client.nodeClient.BuildSignAndSubmitTransaction(sender, payload, options...)
}

// View Runs a view function on chain returning a list of return values.
//
//	 address := AccountOne
//		payload := &ViewPayload{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name:    "primary_fungible_store",
//			},
//			Function: "balance",
//			ArgTypes: []TypeTag{EndlessCoinTypeTag},
//			Args:     [][]byte{address[:], coinAddress[:]},
//		}
//		vals, err := client.endlessClient.View(payload)
//		balance := StrToBigInt(vals.(any[])[0].(string))
func (client *Client) View(payload *ViewPayload, ledgerVersion ...uint64) (vals []any, err error) {
	return client.nodeClient.View(payload, ledgerVersion...)
}

// EstimateGasPrice Retrieves the gas estimate from the network.
func (client *Client) EstimateGasPrice() (info EstimateGasInfo, err error) {
	return client.nodeClient.EstimateGasPrice()
}

// AccountEDSBalance retrieves the EDS balance in the account
func (client *Client) AccountEDSBalance(address AccountAddress, ledgerVersion ...uint64) (*big.Int, error) {
	return client.nodeClient.AccountEDSBalance(address, ledgerVersion...)
}

// AccountCoinBalance retrieves the other coin balance in the account
func (client *Client) AccountCoinBalance(coinAddress string, address AccountAddress, ledgerVersion ...uint64) (*big.Int, error) {
	return client.nodeClient.AccountCoinBalance(coinAddress, address, ledgerVersion...)
}

// NodeAPIHealthCheck checks if the node is within durationSecs of the current time, if not provided the node default is used
func (client *Client) NodeAPIHealthCheck(durationSecs ...uint64) (api.HealthCheckResponse, error) {
	return client.nodeClient.NodeHealthCheck(durationSecs...)
}
