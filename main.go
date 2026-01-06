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
	// Alice generates a new key pair and sends the encapsulation key to Bob.
	// private key
	dk, err := mlkem.GenerateKey768()
	if err != nil {
		log.Fatal(err)
	}
	// public key
	encapsulationKey := dk.EncapsulationKey().Bytes()

	// Bob uses the encapsulation key to encapsulate a shared secret, and sends
	// back the ciphertext to Alice.
	ciphertext := keyExchange(encapsulationKey)

	// Alice decapsulates the shared secret from the ciphertext.
	sharedSecret, err := dk.Decapsulate(ciphertext)
	if err != nil {
		log.Fatal(err)
	}

	// Alice and Bob now share a secret.
	alice.sharedSecret = sharedSecret
	alice.sharedSecretHKDF = deriveKey(sharedSecret)

	// encrypt message
	nonce, encryptedMessage, err := encrypt(alice.sharedSecretHKDF, []byte(ENCRYPTED_MESSAGE))
	if err != nil {
		log.Fatal(err)
	}

	sendEncryptedMessage(encryptedMessage, nonce)

	// validate that both of them have the same message

	fmt.Println("Shared secrets match: ", string(alice.sharedSecret) == string(bob.sharedSecret))
	fmt.Println("Messages match: ", ENCRYPTED_MESSAGE == string(bob.message))
}

func sendEncryptedMessage(ciphertext, nonce []byte) {
	// Bob will use its sharedSecret to be able to decrypt the message
	decrypted := decrypt(bob.sharedSecretHKDF, nonce, ciphertext)

	bob.message = decrypted
}

func keyExchange(encapsulationKey []byte) (ciphertext []byte) {
	// Bob encapsulates a shared secret using the encapsulation key (public key).
	ek, err := mlkem.NewEncapsulationKey768(encapsulationKey)
	if err != nil {
		log.Fatal(err)
	}
	sharedSecret, ciphertext := ek.Encapsulate()

	// Alice and Bob now share a secret.
	bob.sharedSecret = sharedSecret
	bob.sharedSecretHKDF = deriveKey(sharedSecret)

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

func encrypt(key, plaintext []byte) ([]byte, []byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		log.Fatal(err)
	}

	nonce := make([]byte, chacha20poly1305.NonceSize)
	rand.Read(nonce)

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	return nonce, ciphertext, nil
}

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
