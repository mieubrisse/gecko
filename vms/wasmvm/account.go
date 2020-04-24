package wasmvm

import (
	"fmt"
	"math"

	"github.com/ava-labs/gecko/ids"
)

// Account is an account
type Account struct {
	// Address of this account. Serves as a unique ID.
	Address ids.ShortID `serialize:"true"`
	// Most recently used nonce of this account. 0 on account creation.
	Nonce uint64 `serialize:"true"`
}

// Bytes returns the byte representation of this account
func (acc *Account) Bytes() []byte {
	bytes, _ := codec.Marshal(acc)
	return bytes
}

// IncrementNonce increments this account's nonce
func (acc *Account) IncrementNonce() error {
	if acc.Nonce == math.MaxUint64 {
		return fmt.Errorf("account is out of nonces")
	}
	acc.Nonce++
	return nil
}
