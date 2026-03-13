package keychain

import (
	"github.com/zalando/go-keyring"
)

const (
	service = "shdw"
	account = "master"
)

// Get retrieves the cached master password from the OS keychain.
// Returns ("", nil) if not set.
func Get() (string, error) {
	pw, err := keyring.Get(service, account)
	if err == keyring.ErrNotFound {
		return "", nil
	}
	return pw, err
}

// Set stores the master password in the OS keychain.
func Set(password string) error {
	return keyring.Set(service, account, password)
}

// Delete removes the cached master password.
func Delete() error {
	return keyring.Delete(service, account)
}
