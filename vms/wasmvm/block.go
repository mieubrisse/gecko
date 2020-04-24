package wasmvm

import (
	"errors"
	"fmt"

	"github.com/ava-labs/gecko/database/versiondb"

	"github.com/ava-labs/gecko/ids"
	"github.com/ava-labs/gecko/vms/components/core"
)

// Block is a block of transactions
type Block struct {
	vm          *VM
	*core.Block `serialize:"true"`
	Txs         []tx `serialize:"true"`

	// The state of the chain if this block is accepted
	onAcceptDb *versiondb.Database
}

// Initialize this block
// Should be called when block is parsed from bytes
func (b *Block) Initialize(bytes []byte, vm *VM) {
	b.vm = vm
	b.Block.Initialize(bytes, vm.SnowmanVM)
	for _, tx := range b.Txs {
		if err := tx.initialize(vm); err != nil {
			vm.Ctx.Log.Error("couldn't initialize tx: %v", err)
		}
	}
}

// Accept this block
func (b *Block) Accept() {
	if err := b.onAcceptDb.Commit(); err != nil {
		b.vm.Ctx.Log.Error("couldn't commit onAcceptDb: %v", err)
	}
	if err := b.onAcceptDb.Close(); err != nil {
		b.vm.Ctx.Log.Error("couldn't close onAcceptDb: %v", err)
	}
	b.Block.Accept()
	if err := b.vm.DB.Commit(); err != nil {
		b.vm.Ctx.Log.Error("couldn't commit vm.DB: %v", err)
	}
}

// Reject this block
func (b *Block) Reject() {
	if err := b.onAcceptDb.Close(); err != nil {
		b.vm.Ctx.Log.Error("couldn't close onAcceptDb: %v", err)
	}
	b.Block.Reject()
	if err := b.vm.DB.Commit(); err != nil {
		b.vm.Ctx.Log.Error("couldn't commit vm.DB: %v", err)
	}
}

// Verify returns nil iff this block is valid
func (b *Block) Verify() error {
	switch {
	case b.ID().Equals(ids.Empty):
		return errors.New("block ID is empty")
	case len(b.Txs) == 0:
		return errors.New("no txs in block")
	}

	// TODO: If there's an error, return other txs to mempool
	for _, tx := range b.Txs {
		if err := tx.SyntacticVerify(); err != nil {
			return err
		}
	}

	// TODO: If there's an error, return other txs to mempool
	for _, tx := range b.Txs {
		if err := tx.SemanticVerify(b.onAcceptDb); err != nil {
			return err
		}
	}
	return nil
}

// return a new, initialized block
func (vm *VM) newBlock(parentID ids.ID, txs []tx) (*Block, error) {
	block := &Block{
		Block:      core.NewBlock(parentID),
		Txs:        txs,
		onAcceptDb: versiondb.New(vm.DB),
	}

	bytes, err := codec.Marshal(block)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal block: %s", err)
	}
	block.Initialize(bytes, vm)

	if err := vm.SaveBlock(vm.DB, block); err != nil {
		return nil, fmt.Errorf("couldn't save block %s: %s", block.ID(), err)
	}
	if err := vm.DB.Commit(); err != nil {
		return nil, fmt.Errorf("couldn't commit DB: %s", err)
	}
	return block, nil
}
