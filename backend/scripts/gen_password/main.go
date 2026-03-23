package main

import (
	"fmt"
	"os"

	"github.com/chaitin/MonkeyCode/backend/pkg/crypto"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run gen_password.go <password>")
		fmt.Println("Example: go run gen_password.go 123456")
		os.Exit(1)
	}

	password := os.Args[1]
	if len(password) > 32 {
		fmt.Printf("Error: password must be less than 32 characters (current: %d)\n", len(password))
		os.Exit(1)
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		fmt.Printf("Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Password:", password)
	fmt.Println("Hash:", hash)
	fmt.Println()
	fmt.Println("SQL to update password:")
	fmt.Printf("UPDATE users SET password = '%s' WHERE email = 'your-email@example.com';\n", hash)
}
