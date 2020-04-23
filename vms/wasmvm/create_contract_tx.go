package wasmvm

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ava-labs/gecko/snow/choices"

	"github.com/ava-labs/gecko/utils/crypto"
	"github.com/ava-labs/gecko/utils/formatting"

	"github.com/ava-labs/gecko/utils/hashing"

	"github.com/ava-labs/gecko/database"
	"github.com/ava-labs/gecko/ids"
)

// Creates a contract
type createContractTx struct {
	vm *VM

	// ID of this tx and the contract being created
	id ids.ID

	// sender of this transaction
	sender ids.ShortID

	// Byte repr. of the contract
	// Must be valid WASM
	ContractBytes []byte `serialize:"true"`

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

// Should be called when unmarshaling
// Should be called before any of this tx's methods or fields are accessed
// Sets tx.vm, tx.bytes, tx.id, tx.sender
func (tx *createContractTx) initialize(vm *VM) error {
	tx.vm = vm

	// Compute the byte repr. of this tx
	var err error
	tx.bytes, err = codec.Marshal(tx)
	if err != nil {
		return fmt.Errorf("couldn't marshal *createContractTx: %v", err)
	}

	// Compute the ID of this tx
	tx.id = ids.NewID(hashing.ComputeHash256Array(tx.bytes))

	// Compute the sender of this tx
	unsignedBytes, err := codec.Marshal(tx.ContractBytes)
	if err != nil {
		return fmt.Errorf("couldn't marshal createContractTx: %v", err)
	}
	pubKey, err := keyFactory.RecoverPublicKey(unsignedBytes, tx.SenderSig[:])
	if err != nil {
		return fmt.Errorf("couldn't recover public key on createContractTx: %v", err)
	}
	tx.sender = pubKey.Address()
	return nil
}

// SyntacticVerify returns nil iff tx is syntactically valid
func (tx *createContractTx) SyntacticVerify() error {
	switch {
	case tx.ContractBytes == nil:
		return fmt.Errorf("empty contract")
	case tx.id.Equals(ids.Empty):
		return fmt.Errorf("empty tx ID")
	case tx.sender.Equals(ids.ShortEmpty):
		return errors.New("empty sender")
	}
	return nil
}

func (tx *createContractTx) SemanticVerify(db database.Database) error {
	if err := tx.vm.putContractBytes(db, tx.id, tx.ContractBytes); err != nil {
		return fmt.Errorf("couldn't put new contract in db: %v", err)
	}
	if err := tx.vm.putContractState(db, tx.id, []byte{}); err != nil {
		return fmt.Errorf("couldn't initialize contract's state in db: %v", err)
	}
	persistedTx := &txReturnValue{ // TODO: always persist the tx, even if it was unsuccessful
		Tx:     tx,
		Status: choices.Accepted,
	}
	if err := tx.vm.putTx(db, persistedTx); err != nil {
		return err
	}
	return nil
}

// Creates a new tx with the given payload and a random ID
func (vm *VM) newCreateContractTx(contractBytes []byte, senderKey crypto.PrivateKey) (*createContractTx, error) {
	tx := &createContractTx{
		ContractBytes: contractBytes,
	}
	// Generate signature
	sig, err := senderKey.Sign(tx.ContractBytes)
	if err != nil {
		return nil, fmt.Errorf("couldn't sign createContractTx: %v", err)
	}
	copy(tx.SenderSig[:], sig[:]) // Put signature on tx
	if err := tx.initialize(vm); err != nil {
		return nil, err
	}
	return tx, nil
}

func (tx *createContractTx) MarshalJSON() ([]byte, error) {
	asMap := make(map[string]interface{}, 4)
	asMap["id"] = tx.ID().String()
	asMap["sender"] = tx.sender.String()
	byteFormatter := formatting.CB58{Bytes: tx.ContractBytes}
	asMap["contract"] = byteFormatter.String()
	return json.Marshal(asMap)
}
