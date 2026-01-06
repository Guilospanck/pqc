package main

import (
	"crypto/mlkem"
	"crypto/rand"
	"fmt"
	"log"

	"crypto/sha256"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

type Bob struct {
	sharedSecret     []byte
	sharedSecretHKDF []byte
	message          []byte
}

type Alice struct {
	sharedSecret     []byte
	sharedSecretHKDF []byte
	message          []byte
}

var bob Bob = Bob{sharedSecret: nil, message: nil}
var alice Alice = Alice{sharedSecret: nil, message: nil}

const ENCRYPTED_MESSAGE string = "hello"

func main() {
	// private key
	decapsulationKey, err := mlkem.GenerateKey768()
	if err != nil {
		log.Fatal(err)
	}
	// public key
	encapsulationKey := decapsulationKey.EncapsulationKey().Bytes()

	ciphertext := keyExchange(encapsulationKey)

	// Alice decapsulates the shared secret from the ciphertext using her private key.
	sharedSecret, err := decapsulationKey.Decapsulate(ciphertext)
	if err != nil {
		log.Fatal(err)
	}

	// Alice and Bob now share a secret that never went out on the wire.
	alice.sharedSecret = sharedSecret
	alice.sharedSecretHKDF = deriveKey(sharedSecret) // uses HKDF

	// encrypt message with symmetric-key algorithm
	nonce, encryptedMessage, err := encrypt(alice.sharedSecretHKDF, []byte(ENCRYPTED_MESSAGE))
	if err != nil {
		log.Fatal(err)
	}

	sendEncryptedMessage(encryptedMessage, nonce)

	// validate that both of them have the same message
	validate()
}

func validate() {
	fmt.Println("Shared secrets match: ", string(alice.sharedSecret) == string(bob.sharedSecret))
	fmt.Println("Messages match: ", ENCRYPTED_MESSAGE == string(bob.message))
}

func sendEncryptedMessage(ciphertext, nonce []byte) {
	// Bob will use its sharedSecret to be able to decrypt the message
	decrypted := decrypt(bob.sharedSecretHKDF, nonce, ciphertext)

	bob.message = decrypted
}

func keyExchange(encapsulationKey []byte) (ciphertext []byte) {
	// Get the public key out of the byte array.
	ek, err := mlkem.NewEncapsulationKey768(encapsulationKey)
	if err != nil {
		log.Fatal(err)
	}

	// Bob encapsulates a shared secret using the encapsulation key (public key).
	sharedSecret, ciphertext := ek.Encapsulate()

	// Alice and Bob now share a secret.
	bob.sharedSecret = sharedSecret
	bob.sharedSecretHKDF = deriveKey(sharedSecret) // uses HKDF

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
func encrypt(key, plaintext []byte) ([]byte, []byte, error) {
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
func decrypt(key, nonce, ciphertext []byte) []byte {
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
