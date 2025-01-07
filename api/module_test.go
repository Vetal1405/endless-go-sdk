package api

import (
	"encoding/json"
	"testing"

	"github.com/endless-labs/endless-go-sdk/internal/util"
	"github.com/stretchr/testify/assert"
)

// TestModule_MoveScript tests the MoveScript struct
func TestModule_MoveScript(t *testing.T) {
	testJson := `{
		"bytecode": "0xa11ceb0b060000000901000202020403060f0515"
	}`
	data := &MoveScript{}
	err := json.Unmarshal([]byte(testJson), &data)
	assert.NoError(t, err)
	expectedRes, _ := util.ParseHex("0xa11ceb0b060000000901000202020403060f0515")
	assert.Equal(t, HexBytes(expectedRes), data.Bytecode)
}
