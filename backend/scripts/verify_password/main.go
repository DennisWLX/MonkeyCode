package main

import (
	"fmt"
	"os"

	"github.com/chaitin/MonkeyCode/backend/pkg/crypto"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run verify_password.go <hash> <password>")
		fmt.Println("Example: go run verify_password.go $2a$10$xxx... 123456")
		os.Exit(1)
	}

	hash := os.Args[1]
	password := os.Args[2]

	fmt.Println("Hash:", hash)
	fmt.Println("Password:", password)
	fmt.Println("Hash length:", len(hash))
	fmt.Println()

	err := crypto.VerifyPassword(hash, password)
	if err != nil {
		fmt.Printf("❌ Verification failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Password verified successfully!")
}
