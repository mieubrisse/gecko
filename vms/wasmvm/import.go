package wasmvm

// void print(void *context, int ptr, int len);
// int dbPut(void *context, int key, int keyLen, int value, int valueLen);
// int dbGet(void *context, int key, int keyLen, int value);
// int returnValue(void *context, int valuePtr, int valueLen);
// int dbGetValueLen(void *context, int keyPtr, int keyLen);
// int getArgs(void *context, int ptr);
// int getSender(void *context, int ptr);
import "C"
import (
	"fmt"
	"unicode/utf8"
	"unsafe"

	"github.com/ava-labs/gecko/ids"

	"github.com/ava-labs/gecko/database"
	"github.com/ava-labs/gecko/utils/math"

	"github.com/ava-labs/gecko/utils/logging"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

const (
	addressSize            = 20
	maxContractDbKeySize   = 1024
	maxContractDbValueSize = 1024
)

type ctx struct {
	log        logging.Logger    // this chain's logger
	contractDb database.Database // DB for the contract to read/write
	memory     *wasm.Memory      // the instance's memory
	txID       ids.ID            // ID of transaction that triggered current SC method invocation
	sender     ids.ShortID       // Address of sender of the tx that triggered current SC method invocation
}

// Print bytes in the smart contract's memory
// The bytes are interpreted as a string
//export print
func print(context unsafe.Pointer, ptr C.int, strLen C.int) {
	ctxRaw := wasm.IntoInstanceContext(context)
	ctx := ctxRaw.Data().(ctx)
	instanceMemory := ctx.memory.Data()
	finalIndex, err := math.Add32(uint32(ptr), uint32(strLen))
	if err != nil || int(finalIndex) > len(instanceMemory) {
		ctx.log.Error("Print from smart contract failed. Index out of bounds.")
		return
	}

	toPrint := instanceMemory[ptr:finalIndex]
	if asStr := string(toPrint); utf8.ValidString(asStr) {
		ctx.log.Info("Print from smart contract: %s", asStr)
	} else {
		ctx.log.Info("Print from smart contract: %v", toPrint)
	}

}

// Put a KV pair where the key/value are defined by a pointer to the first byte
// and the length of the key/value.
// Returns 0 if successful, otherwise unsuccessful
//export dbPut
func dbPut(context unsafe.Pointer, keyPtr C.int, keyLen C.int, valuePtr C.int, valueLen C.int) C.int {
	// Get the context
	ctxRaw := wasm.IntoInstanceContext(context)
	ctx := ctxRaw.Data().(ctx)

	// Validate arguments
	if keyPtr < 0 || keyLen < 0 {
		ctx.log.Error("dbPut failed. Key pointer and length must be non-negative")
		return 1
	} else if valuePtr < 0 || valueLen < 0 {
		ctx.log.Error("dbPut failed. Value pointer and length must be non-negative")
		return 1
	}
	contractState := ctx.memory.Data()
	keyFinalIndex, err := math.Add32(uint32(keyPtr), uint32(keyLen))
	if err != nil || int(keyFinalIndex) > len(contractState) {
		ctx.log.Error("dbPut failed. Key index out of bounds.")
		return 1
	}
	valueFinalIndex, err := math.Add32(uint32(valuePtr), uint32(valueLen))
	if err != nil || int(valueFinalIndex) > len(contractState) {
		ctx.log.Error("dbPut failed. Value index out of bounds.")
		return 1
	}

	// Do the put
	key := contractState[keyPtr:keyFinalIndex]
	value := contractState[valuePtr:valueFinalIndex]
	ctx.log.Debug("Putting K/V pair for contract.\n  key: %v\n  value: %v", key, value)
	if err := ctx.contractDb.Put(key, value); err != nil {
		ctx.log.Error("dbPut failed: %s", err)
		return 1
	}
	return 0
}

// Get a value from the database. The key is in the contract's memory.
// It starts at [keyPtr] and is [keyLen] bytes long.
// The value is written to the contract's memory starting at [valuePtr]
// Returns the length of the returned value, or -1 if the get failed.
//export dbGet
func dbGet(context unsafe.Pointer, keyPtr C.int, keyLen C.int, valuePtr C.int) C.int {
	// Get the context
	ctxRaw := wasm.IntoInstanceContext(context)
	ctx := ctxRaw.Data().(ctx)

	// Validate arguments
	if keyPtr < 0 || keyLen < 0 {
		ctx.log.Error("dbGet failed. Key pointer and length must be non-negative")
		return -1
	} else if valuePtr < 0 {
		ctx.log.Error("dbGet failed. Value pointer must be non-negative")
		return -1
	}
	contractState := ctx.memory.Data()
	keyFinalIndex, err := math.Add32(uint32(keyPtr), uint32(keyLen))
	if err != nil || int(keyFinalIndex) > len(contractState) {
		ctx.log.Error("dbGet failed. Key index out of bounds")
		return -1
	}

	key := contractState[keyPtr:keyFinalIndex]
	value, err := ctx.contractDb.Get(key)
	if err != nil {
		ctx.log.Error("dbGet failed: %s", err)
		return -1
	}
	ctx.log.Verbo("dbGet returning\n  key: %v\n value: %v\n", key, value)
	copy(contractState[valuePtr:], value)
	return C.int(len(value))
}

// Get the length in bytes of the value associated with a key in the database.
// The key is in the contract's memory at [keyPtr, keyLen]
// Returns -1 if the value corresponding to the key couldn't be found
//export dbGetValueLen
func dbGetValueLen(context unsafe.Pointer, keyPtr C.int, keyLen C.int) C.int {
	// Get the context
	ctxRaw := wasm.IntoInstanceContext(context)
	ctx := ctxRaw.Data().(ctx)

	// Validate arguments
	if keyPtr < 0 || keyLen < 0 {
		ctx.log.Error("dbGetValueLen failed. Key pointer and length must be non-negative")
		return -1
	}
	contractState := ctx.memory.Data()
	keyFinalIndex, err := math.Add32(uint32(keyPtr), uint32(keyLen))
	if err != nil || int(keyFinalIndex) > len(contractState) {
		ctx.log.Error("dbGetValueLen failed. Key index out of bounds")
		return -1
	}

	key := contractState[keyPtr:keyFinalIndex]
	value, err := ctx.contractDb.Get(key)
	if err != nil {
		ctx.log.Error("dbGetValueLen failed: %s", err)
		return -1
	}
	ctx.log.Verbo("dbGetValueLen returning %v", len(value))
	return C.int(len(value))
}

// Write the address that triggered this contract invocation to the contract's memory, starting at [ptr]
// The address is guaranteed to be addressSize bytes
// Returns 0 on success, other return value indicates failure
//export getSender
func getSender(context unsafe.Pointer, ptr C.int) C.int {
	// Get the context
	ctxRaw := wasm.IntoInstanceContext(context)
	ctx := ctxRaw.Data().(ctx)

	// Get the sender
	sender, err := ctx.contractDb.Get(senderKey)
	if err != nil {
		ctx.log.Error("getSender failed. Couldn't get sender: %v", err)
		return -1
	}
	if senderLen := len(sender); senderLen != addressSize {
		ctx.log.Error("getSender failed. Expected sender address to be %v bytes but got %v", addressSize, senderLen)
		return -1
	}

	// Check for array bounds
	contractState := ctx.memory.Data()
	finalIndex, err := math.Add32(uint32(ptr), uint32(len(sender)))
	if err != nil || int(finalIndex) > len(contractState) {
		ctx.log.Error("getSender failed. Index out of bounds")
		return -1
	}

	// Write the sender's address
	copy(contractState[ptr:], sender)
	return 0
}

// Write the byte arguments to a contract method to the contract's memory, starting at [ptr]
// The arguments are guaranteed to be no more than maxContractDbValueSize
// Returns the length of the args, or -1 if the call was unsuccessful
//export getArgs
func getArgs(context unsafe.Pointer, ptr C.int) C.int {
	// Get the context
	ctxRaw := wasm.IntoInstanceContext(context)
	ctx := ctxRaw.Data().(ctx)

	// Get the args
	args, err := ctx.contractDb.Get(argsKey)
	if err != nil {
		ctx.log.Error("getArgs failed. Couldn't get arguments %v", err)
	}

	// Check for array bounds
	contractState := ctx.memory.Data()
	finalIndex, err := math.Add32(uint32(ptr), uint32(len(args)))
	if err != nil || int(finalIndex) > len(contractState) {
		ctx.log.Error("getArgs failed. Index out of bounds")
		return -1
	}

	// Put the args
	copy(contractState[ptr:], args)
	return C.int(len(args))
}

// Smart contract methods call this method to return a value
// Creates a mapping:
//   Key: returnKey (defined in invoke_tx.go)
//   Value: contract's memory in [ptr,ptr+len)
// Should be called at most once by a smart contract
// If called multiple times, final call determined return value
// Returns 0 on success, other return value indicates failure
//export returnValue
func returnValue(context unsafe.Pointer, valuePtr C.int, valueLen C.int) C.int {
	// Get the context
	ctxRaw := wasm.IntoInstanceContext(context)
	ctx := ctxRaw.Data().(ctx)

	// Validate arguments
	if valueLen > maxContractDbValueSize {
		ctx.log.Error("returnValue failed. valueLen > macContractDbValueSize")
		return -1
	}
	if valuePtr < 0 || valueLen < 0 {
		ctx.log.Error("returnValue failed. Value pointer and length must be non-negative")
		return -1
	}
	contractState := ctx.memory.Data()
	finalIndex, err := math.Add32(uint32(valuePtr), uint32(valueLen))
	if err != nil || int(finalIndex) > len(contractState) {
		ctx.log.Error("returnValue failed. Index out of bounds")
		return -1
	}

	// Put the value
	value := contractState[valuePtr:finalIndex]
	if err := ctx.contractDb.Put(returnKey, value); err != nil {
		return -1
	}

	return 0
}

// Return the imports (host methods) available to smart contracts
func standardImports() *wasm.Imports {
	imports, err := wasm.NewImportObject().Imports()
	if err != nil {
		panic(fmt.Sprintf("couldn't create wasm imports: %v", err))
	}
	imports, err = imports.AppendFunction("print", print, C.print)
	if err != nil {
		panic(fmt.Sprintf("couldn't add print import: %v", err))
	}
	imports, err = imports.AppendFunction("dbPut", dbPut, C.dbPut)
	if err != nil {
		panic(fmt.Sprintf("couldn't add dbPut import: %v", err))
	}
	imports, err = imports.AppendFunction("dbGet", dbGet, C.dbGet)
	if err != nil {
		panic(fmt.Sprintf("couldn't add dbGet import: %v", err))
	}
	imports, err = imports.AppendFunction("dbGetValueLen", dbGetValueLen, C.dbGetValueLen)
	if err != nil {
		panic(fmt.Sprintf("couldn't add dbGetValueLen import: %v", err))
	}
	imports, err = imports.AppendFunction("returnValue", returnValue, C.returnValue)
	if err != nil {
		panic(fmt.Sprintf("couldn't add returnValue import: %v", err))
	}
	imports, err = imports.AppendFunction("getArgs", getArgs, C.getArgs)
	if err != nil {
		panic(fmt.Sprintf("couldn't add getArgs import: %v", err))
	}
	imports, err = imports.AppendFunction("getSender", getSender, C.getSender)
	if err != nil {
		panic(fmt.Sprintf("couldn't add getSender import: %v", err))
	}
	return imports
}
