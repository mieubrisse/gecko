package wasmvm

import (
	"bytes"
	"encoding/json"

	"github.com/ava-labs/gecko/utils/formatting"
)

// txReturnValue is a transaction, its status and, if the tx was a SC method invocation,
// its return value.
type txReturnValue struct {
	vm *VM

	// The transaction itself
	Tx tx `serialize:"true"`
	// True if Tx is an invokeTx and the SC method invocation was successful
	// Otherwise false
	InvocationSuccessful bool `serialize:"true"`
	// If Tx is an invokeTx, ReturnValue is the SC method's return value
	// Otherwise empty.
	ReturnValue []byte `serialize:"true"`
}

// Bytes returns the byte representation
func (rv *txReturnValue) Bytes() []byte {
	bytes, err := codec.Marshal(rv)
	if err != nil {
		rv.vm.Ctx.Log.Error("couldn't marshal TxReturnValue: %v", err)
	}
	return bytes
}

func (rv *txReturnValue) MarshalJSON() ([]byte, error) {
	asMap := make(map[string]interface{}, 5)
	asMap["tx"] = rv.Tx
	switch rv.Tx.(type) {
	case *invokeTx:
		asMap["type"] = "contract invocation"
		asMap["invocationSuccessful"] = rv.InvocationSuccessful
		asMap["returnValue"] = formatReturnValue(rv.ReturnValue)
	case *createContractTx:
		asMap["type"] = "contract creation"
	}

	return json.Marshal(asMap)
}

func formatReturnValue(returnValue []byte) interface{} {
	if bytes.Equal(returnValue, []byte{}) {
		return nil
	}
	var asMap map[string]interface{}
	if err := json.Unmarshal(returnValue, &asMap); err == nil { // If returnValue is JSON, display it as such.
		return asMap
	}
	byteFormatter := formatting.CB58{Bytes: returnValue}
	return byteFormatter.String()
}
