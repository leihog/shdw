package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// VaultVersion identifies the encryption scheme used in a vault file.
// The version byte is always the first byte of the encrypted blob.
type VaultVersion byte

const (
	// Version1 used PBKDF2-SHA256 + AES-256-GCM. No longer written; reading
	// it will return ErrUnsupportedVersion so the user knows to recreate the vault.
	Version1 VaultVersion = 0x01

	// Version2 uses Argon2id + AES-256-GCM. Current version.
	Version2 VaultVersion = 0x02

	currentVersion = Version2
)

// ErrUnsupportedVersion is returned when opening a vault written by an older
// (or newer) version of shdw that used a different encryption scheme.
var ErrUnsupportedVersion = errors.New(
	"vault was created with an unsupported version of shdw — " +
		"delete the vault file and create a new one",
)

// Argon2id parameters. Tuned for ~100ms on modern hardware.
// time=1, memory=64MB, threads=4, keyLen=32
const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB in KiB
	argonThreads = 4
	argonKeyLen  = 32

	saltSize = 16
)

// Encrypt encrypts plaintext using AES-256-GCM with an Argon2id-derived key.
//
// Output format:
//   [version (1 byte)] [salt (16 bytes)] [nonce (12 bytes)] [ciphertext+tag]
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	out := make([]byte, 0, 1+saltSize+len(ciphertext))
	out = append(out, byte(currentVersion))
	out = append(out, salt...)
	out = append(out, ciphertext...)
	return out, nil
}

// Decrypt decrypts data produced by Encrypt.
// Returns ErrUnsupportedVersion if the version byte is not recognised.
func Decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < 1 {
		return nil, errors.New("vault data is empty")
	}

	version := VaultVersion(data[0])
	rest := data[1:]

	switch version {
	case Version2:
		return decryptV2(rest, password)
	case Version1:
		return nil, ErrUnsupportedVersion
	default:
		return nil, fmt.Errorf("unknown vault version 0x%02x — "+
			"you may need a newer version of shdw", version)
	}
}

func decryptV2(data []byte, password string) ([]byte, error) {
	if len(data) < saltSize {
		return nil, errors.New("vault data is too short")
	}

	salt := data[:saltSize]
	rest := data[saltSize:]
	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(rest) < gcm.NonceSize() {
		return nil, errors.New("vault data is too short")
	}

	nonce := rest[:gcm.NonceSize()]
	ciphertext := rest[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed — wrong password or corrupted vault")
	}

	return plaintext, nil
}

func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)
}
