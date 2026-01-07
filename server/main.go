package main

// const ENCRYPTED_MESSAGE string = "hello"
//
// func main() {
//
// 	ciphertext := keyExchange(encapsulationKey)
//
// 	// Alice decapsulates the shared secret from the ciphertext using her private key.
// 	sharedSecret, err := decapsulationKey.Decapsulate(ciphertext)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	// Alice and Bob now share a secret that never went out on the wire.
// 	alice.sharedSecret = sharedSecret
// 	alice.sharedSecretHKDF = deriveKey(sharedSecret) // uses HKDF
//
// 	// encrypt message with symmetric-key algorithm
// 	nonce, encryptedMessage, err := encrypt(alice.sharedSecretHKDF, []byte(ENCRYPTED_MESSAGE))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	sendEncryptedMessage(encryptedMessage, nonce)
//
// 	// validate that both of them have the same message
// 	validate()
// }
//
// func validate() {
// 	fmt.Println("Shared secrets match: ", string(alice.sharedSecret) == string(bob.sharedSecret))
// 	fmt.Println("Messages match: ", ENCRYPTED_MESSAGE == string(bob.message))
// }
//
// func sendEncryptedMessage(ciphertext, nonce []byte) {
// 	// Bob will use its sharedSecret to be able to decrypt the message
// 	decrypted := decrypt(bob.sharedSecretHKDF, nonce, ciphertext)
//
// 	bob.message = decrypted
// }

func main() {
	NewWSServer()
}
