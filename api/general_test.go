package api

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_HealthCheckResponse(t *testing.T) {
	testJson := `{
		"message": "endless-node:ok"
	}`
	data := &HealthCheckResponse{}
	err := json.Unmarshal([]byte(testJson), &data)
	assert.NoError(t, err)
	assert.Equal(t, "endless-node:ok", data.Message)
}
