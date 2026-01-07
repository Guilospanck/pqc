package cryptography

import (
	"crypto/mlkem"
	"crypto/rand"
	"fmt"
	"log"

	"crypto/sha256"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

type Party struct {
	sharedSecret     []byte
	sharedSecretHKDF []byte
	message          []byte
}

type Keys struct {
	Private *mlkem.DecapsulationKey768
	Public  []byte
}

func GenerateKeys() (*Keys, error) {
	// private key
	decapsulationKey, err := mlkem.GenerateKey768()
	if err != nil {
		fmt.Errorf("Error trying to generate private key: %s", err.Error())
		return nil, err
	}
	// public key
	encapsulationKey := decapsulationKey.EncapsulationKey().Bytes()

	keys := Keys{
		Private: decapsulationKey,
		Public:  encapsulationKey,
	}

	return &keys, nil
}

func KeyExchange(encapsulationKey []byte) (ciphertext []byte) {
	// Get the public key out of the byte array.
	ek, err := mlkem.NewEncapsulationKey768(encapsulationKey)
	if err != nil {
		log.Fatal(err)
	}

	// Bob encapsulates a shared secret using the encapsulation key (public key).
	// sharedSecret, ciphertext := ek.Encapsulate()
	_, ciphertext = ek.Encapsulate()

	// Alice and Bob now share a secret.
	// bob.sharedSecret = sharedSecret
	// bob.sharedSecretHKDF = deriveKey(sharedSecret) // uses HKDF

	// Bob sends the ciphertext to Alice.
	return ciphertext
}

// Uses HKDF to make the shared secret even more hard to be discovered and
// also more uniform and able to be used into the symmetric algorithms
func deriveKey(sharedSecret []byte) []byte {
	hkdf := hkdf.New(sha256.New, sharedSecret, nil, nil)
	key := make([]byte, 32) // 256-bit
	hkdf.Read(key)
	return key
}

// Symmetrically encrypts a message using CHACHA20-POLY1305
func EncryptMessage(key, plaintext []byte) ([]byte, []byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		log.Fatal(err)
	}

	// different nonce for each message (plaintext)
	nonce := make([]byte, chacha20poly1305.NonceSize)
	rand.Read(nonce)

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	return nonce, ciphertext, nil
}

// Symmetrically decripts a message using CHACHA20-POLY1305
func DecryptMessage(key, nonce, ciphertext []byte) []byte {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		log.Fatal(err)
	}

	result, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Fatal(err)
	}

	return result
}
