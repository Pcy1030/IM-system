package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		Charset  string `yaml:"charset"`
	} `yaml:"database"`
}

func main() {
	// Load configuration
	config := loadConfig()

	// Build DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		config.Database.Username,
		config.Database.Password,
		config.Database.Host,
		config.Database.Port,
		config.Database.Database,
		config.Database.Charset,
	)

	// Connect DB
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Database connection test failed: %v", err)
	}

	fmt.Println("Database connected successfully")
	fmt.Printf("Database: %s\n", config.Database.Database)

	// Confirm
	fmt.Print("\nWARNING: This operation will CLEAR ALL DATA in tables [message, friendship, user]!\n")
	fmt.Print("Type 'YES' to confirm: ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "YES" {
		fmt.Println("Operation cancelled")
		return
	}

	// Disable FK checks to avoid constraint issues
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS=0")

	// Clear data (child tables first)
	tables := []string{"message", "friendship", "user"}
	for _, table := range tables {
		fmt.Printf("Clearing table %s... ", table)
		if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			fmt.Printf("Failed: %v\n", err)
		} else {
			fmt.Println("Success")
		}
	}

	// Reset auto-increment ids
	fmt.Println("\nResetting auto-increment IDs...")
	for _, table := range tables {
		fmt.Printf("Resetting %s auto-increment... ", table)
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s AUTO_INCREMENT = 1", table)); err != nil {
			fmt.Printf("Failed: %v\n", err)
		} else {
			fmt.Println("Success")
		}
	}

	// Re-enable FK checks
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS=1")

	fmt.Println("\nDatabase reset completed!")
	fmt.Println("All table data cleared, table structure preserved")
	fmt.Println("Auto-increment IDs reset to 1")
}

func loadConfig() *Config {
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		fmt.Println("Config file not found, using default config")
		return &Config{Database: struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
			Database string `yaml:"database"`
			Charset  string `yaml:"charset"`
		}{
			Host:     "localhost",
			Port:     3306,
			Username: "im_user",
			Password: "Pcy010728.",
			Database: "im_system",
			Charset:  "utf8mb4",
		}}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Config file parsing failed: %v", err)
	}
	return &cfg
}
