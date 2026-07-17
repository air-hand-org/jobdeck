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
	random_password := make([]byte, 32)
	if _, err := rand.Read(random_password); err != nil {
		return "", err
	}
	encoded_password := base64.RawStdEncoding.EncodeToString(random_password)
	return encoded_password, nil
}

func main() {
	encoded_password, err := password()
	if err != nil {
		log.Fatalf("%v", err)
	}
	factory := secret.NewPHCBuilderFactory()
	fmt.Printf("pass: [ %s ], hashed string: [ %s ]\n", encoded_password, factory.NewBuilder(nil).PHC([]byte(encoded_password)).String())
}
