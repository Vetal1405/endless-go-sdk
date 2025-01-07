package endless

import (
	"encoding/hex"
	"strconv"
	"strings"
)

// AccountInfo is returned from calls to #Account()
type AccountInfo struct {
	SequenceNumberStr string `json:"sequence_number"`
	AuthenticationKeyHex  []string `json:"authentication_key"`
	NumSignaturesRequired int      `json:"num_signatures_required"`
}

// AuthenticationKey Hex decode of AuthenticationKeyHex
func (ai AccountInfo) AuthenticationKey() ([][]byte, error) {
	//ak := ai.AuthenticationKeyHex
	//if strings.HasPrefix(ak, "0x") {
	//	ak = ak[2:]
	//}
	//return hex.DecodeString(ak)

	var authenticationKeyBytes [][]byte
	for i, value := range ai.AuthenticationKeyHex {
		if strings.HasPrefix(value, "0x") {
			ai.AuthenticationKeyHex[i] = ai.AuthenticationKeyHex[i][2:]
		}

		authenticationKeyByte, _ := hex.DecodeString(ai.AuthenticationKeyHex[i])
		authenticationKeyBytes = append(authenticationKeyBytes, authenticationKeyByte)
	}

	return authenticationKeyBytes, nil
}

// SequenceNumber ParseUint of SequenceNumberStr
func (ai AccountInfo) SequenceNumber() (uint64, error) {
	return strconv.ParseUint(ai.SequenceNumberStr, 10, 64)
}
