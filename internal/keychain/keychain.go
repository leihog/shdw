package keychain

// Package keychain provides a simple interface for storing and retrieving the master password in the OS keychain.

import (
	"github.com/zalando/go-keyring"
)

const (
	service = "shdw"
	account = "master"
)

func Get() (string, error) {
	pw, err := keyring.Get(service, account)
	if err == keyring.ErrNotFound {
		return "", nil
	}
	return pw, err
}

func Set(password string) error {
	return keyring.Set(service, account, password)
}

func Delete() error {
	return keyring.Delete(service, account)
}
