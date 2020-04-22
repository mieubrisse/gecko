package wasmvm

import "github.com/ava-labs/gecko/ids"

// Account is an account
type Account struct {
	Address ids.ShortID `serialize:"true"`
	Nonce   uint64      `serialize:"true"`
}

// Bytes returns the byte representation of this account
func (acc *Account) Bytes() []byte {
	bytes, _ := codec.Marshal(acc)
	return bytes
}
