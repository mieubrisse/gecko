package wasmvm

import (
	"encoding/json"
	"errors"
	"fmt"

	wasm "github.com/wasmerio/go-ext-wasm/wasmer"

	"github.com/ava-labs/gecko/utils/crypto"
	"github.com/ava-labs/gecko/utils/formatting"

	"github.com/ava-labs/gecko/database"
	"github.com/ava-labs/gecko/ids"
	"github.com/ava-labs/gecko/utils/hashing"
	jsonhelper "github.com/ava-labs/gecko/utils/json"
)

// UnsignedCreateContractTx ...
type UnsignedCreateContractTx struct {
	vm *VM

	// ID of this tx and the contract being created
	id ids.ID

	// Address of the sender of this transaction
	senderAddress ids.ShortID

	// Byte repr. of the contract
	// Must be valid WASM
	ContractBytes []byte `serialize:"true"`

	// Next unused nonce of the sender
	SenderNonce uint64 `serialize:"true"`

	// Byte representation of this transaction, excluding the signature
	unsignedBytes []byte
}

// Creates a contract
type createContractTx struct {
	UnsignedCreateContractTx `serialize:"true"`

	// Signature of the sender of this transaction
	SenderSig [crypto.SECP256K1RSigLen]byte `serialize:"true"`

	// Byte repr. of this tx
	bytes []byte
}

// Bytes returns the byte representation of this transaction
func (tx *createContractTx) Bytes() []byte {
	return tx.bytes
}

// ID returns this tx's ID
// Should only be called after tx is initialized
func (tx *createContractTx) ID() ids.ID {
	return tx.id
}

// Should be called when unmarshaling, before
// any of this tx's methods or fields are accessed
// Sets tx.vm, tx.bytes, tx.id, tx.sender, tx.unsignedbytes
func (tx *createContractTx) initialize(vm *VM) error {
	tx.vm = vm

	// Compute the byte repr. of this tx
	var err error
	tx.unsignedBytes, err = codec.Marshal(tx.UnsignedCreateContractTx)
	if err != nil {
		return fmt.Errorf("couldn't marshal UnsignedCreateContractTx: %v", err)
	}
	tx.bytes, err = codec.Marshal(tx)
	if err != nil {
		return fmt.Errorf("couldn't marshal *createContractTx: %v", err)
	}

	// Compute the ID of this tx
	tx.id = ids.NewID(hashing.ComputeHash256Array(tx.bytes))

	// Compute the sender of this tx
	pubKey, err := keyFactory.RecoverPublicKey(tx.unsignedBytes, tx.SenderSig[:])
	if err != nil {
		return fmt.Errorf("couldn't recover public key on createContractTx: %v", err)
	}
	tx.senderAddress = pubKey.Address()
	return nil
}

// SyntacticVerify returns nil iff tx is syntactically valid
func (tx *createContractTx) SyntacticVerify() error {
	switch {
	case tx.ContractBytes == nil:
		return fmt.Errorf("empty contract")
	case tx.id.Equals(ids.Empty):
		return fmt.Errorf("empty tx ID")
	case tx.senderAddress.Equals(ids.ShortEmpty):
		return errors.New("empty sender address")
	case !wasm.Validate(tx.ContractBytes): // Verify that [tx.ContractBytes] is valid WASM
		return fmt.Errorf("contract is not valid WASM")
	}
	return nil
}

func (tx *createContractTx) SemanticVerify(db database.Database) error {
	sender, err := tx.vm.getAccount(db, tx.senderAddress) // Get the sender
	if err != nil {                                       // Account not found...must not exist yet. Create it.
		sender = &Account{Address: tx.senderAddress, Nonce: 0}
	}
	if err := sender.IncrementNonce(); err != nil { // Get sender's next unused nonce
		return fmt.Errorf("couldn't increment sender's nonce: %v", err)
	}
	if sender.Nonce != tx.SenderNonce { // Make sure nonce in tx is correct
		return fmt.Errorf("expected sender's next unused nonce to be %d but was %d", tx.SenderNonce, sender.Nonce)
	}
	if err := tx.vm.putAccount(db, sender); err != nil {
		return fmt.Errorf("couldn't persist sender: %v", err)
	}

	if err := tx.vm.putContractBytes(db, tx.id, tx.ContractBytes); err != nil {
		return fmt.Errorf("couldn't put new contract in db: %v", err)
	}
	if err := tx.vm.putContractState(db, tx.id, []byte{}); err != nil {
		return fmt.Errorf("couldn't initialize contract's state in db: %v", err)
	}
	persistedTx := &txReturnValue{
		Tx: tx,
	}
	if err := tx.vm.putTx(db, persistedTx); err != nil {
		return err
	}
	return nil
}

// Creates a new tx with the given payload and a random ID
func (vm *VM) newCreateContractTx(contractBytes []byte, senderNonce uint64, senderKey *crypto.PrivateKeySECP256K1R) (*createContractTx, error) {
	tx := &createContractTx{
		UnsignedCreateContractTx: UnsignedCreateContractTx{
			SenderNonce:   senderNonce,
			ContractBytes: contractBytes,
		},
	}
	// Generate signature
	unsignedBytes, err := codec.Marshal(tx.UnsignedCreateContractTx)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal UnsignedCreateContractTx: %v", err)
	}
	sig, err := senderKey.Sign(unsignedBytes)
	if err != nil {
		return nil, fmt.Errorf("couldn't sign createContractTx: %v", err)
	}
	copy(tx.SenderSig[:], sig[:]) // Put signature on tx
	if err := tx.initialize(vm); err != nil {
		return nil, fmt.Errorf("couldn't initialize createContractTx: %v", err)
	}
	return tx, nil
}

func (tx *createContractTx) MarshalJSON() ([]byte, error) {
	asMap := make(map[string]interface{}, 4)
	asMap["id"] = tx.ID().String()
	asMap["sender"] = tx.senderAddress.String()
	asMap["nonce"] = jsonhelper.Uint64(tx.SenderNonce)
	byteFormatter := formatting.CB58{Bytes: tx.ContractBytes}
	asMap["contract"] = byteFormatter.String()
	return json.Marshal(asMap)
}
