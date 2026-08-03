package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log"
)

func encodeKey(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

func main() {
	pubkey, privkey, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatalf("failed to generate key: %v", err)
	}
	fmt.Printf("b64encoded public key: [ %s ]\n", encodeKey(pubkey))
	fmt.Printf("b64encoded private key: [ %s ]\n", encodeKey(privkey))
}
