package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/webpush"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	db := database.GetDB()

	// Get current keys
	fmt.Println("Checking current VAPID keys...")
	keys, err := webpush.GetVAPIDKeys(db)
	if err != nil {
		log.Printf("No existing VAPID keys found: %v", err)
	} else {
		fmt.Printf("Current public key: %s...\n", keys.PublicKey[:20])
		fmt.Printf("Current private key length: %d\n", len(keys.PrivateKey))
	}

	// Ask for confirmation
	fmt.Println("\nWARNING: Regenerating VAPID keys will invalidate ALL existing push subscriptions!")
	fmt.Println("Users will need to re-subscribe to notifications.")
	fmt.Print("Are you sure you want to continue? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		fmt.Println("Aborted.")
		os.Exit(0)
	}

	// Regenerate keys
	fmt.Println("\nRegenerating VAPID keys...")
	if err := webpush.RegenerateVAPIDKeys(db); err != nil {
		log.Fatalf("Failed to regenerate VAPID keys: %v", err)
	}

	// Get new keys
	newKeys, err := webpush.GetVAPIDKeys(db)
	if err != nil {
		log.Fatalf("Failed to retrieve new VAPID keys: %v", err)
	}

	fmt.Println("\n✅ Successfully regenerated VAPID keys!")
	fmt.Printf("New public key: %s\n", newKeys.PublicKey)
	fmt.Printf("Public key length: %d characters\n", len(newKeys.PublicKey))
	fmt.Printf("Private key length: %d characters\n", len(newKeys.PrivateKey))
	fmt.Println("\nAll existing subscriptions have been cleared.")
	fmt.Println("Users will need to re-subscribe to notifications.")
}
