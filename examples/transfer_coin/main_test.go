package main

import (
	"github.com/endless-labs/endless-go-sdk"
	"testing"
)

func Test_Main(t *testing.T) {
	t.Parallel()
	example(endless.TestnetConfig)
}
