package endless

import (
	"encoding/base64"
	"github.com/endless-labs/endless-go-sdk/bcs"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
)

func decodeB64(x string) ([]byte, error) {
	reader := strings.NewReader(x)
	dec := base64.NewDecoder(base64.StdEncoding, reader)
	return io.ReadAll(dec)
}

func TestMoveResourceBCS(t *testing.T) {
	// curl -o /tmp/ar_bcs --header "Accept: application/x-bcs" http://127.0.0.1:8080/v1/accounts/{addr}/resources
	// base64 < /tmp/ar_bcs
	b64text := "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABB2FjY291bnQHQWNjb3VudAA6ASAoJOVv\nbuFF0+C4aKaA1KaWubBIsTAA65TF9lLMugWD50wAAAAAAAAAAAAAAAAAAAABAAAAAAAAAA=="

	blob, err := decodeB64(b64text)
	assert.NoError(t, err)
	assert.NotNil(t, blob)

	deserializer := bcs.NewDeserializer(blob)
	resources := bcs.DeserializeSequence[AccountResourceRecord](deserializer)
	assert.NoError(t, deserializer.Error())

	assert.Equal(t, 1, len(resources))
	assert.Equal(t, "0x1::account::Account", resources[0].Tag.String())
}
