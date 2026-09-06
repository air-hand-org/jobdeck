// Package main hashes a supplied or generated test password.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"jobdeck/services/auth/internal/secret"
	"log"
	"os"
)

func password() (string, error) {
	password := os.Getenv("INPUT_PASSWORD")
	if password != "" {
		return password, nil
	}
	randomPassword := make([]byte, 32)
	if _, err := rand.Read(randomPassword); err != nil {
		return "", err
	}
	encodedPassword := base64.RawStdEncoding.EncodeToString(randomPassword)
	return encodedPassword, nil
}

func main() {
	encodedPassword, err := password()
	if err != nil {
		log.Fatalf("%v", err)
	}
	factory := secret.NewPHCBuilderFactory()
	fmt.Printf("pass: [ %s ], hashed string: [ %s ]\n", encodedPassword, factory.NewBuilder(nil).PHC([]byte(encodedPassword)).String())
}
