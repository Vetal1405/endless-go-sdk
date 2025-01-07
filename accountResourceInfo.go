package endless

// AccountResourceInfo is returned by #AccountResource() and #AccountResources()
type AccountResourceInfo struct {
	// e.g. "0x1::account::Account"
	Type string `json:"type"`

	// Decoded from Move contract data, could really be anything
	Data map[string]any `json:"data"`
}
