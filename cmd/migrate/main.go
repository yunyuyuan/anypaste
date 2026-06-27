package main

import (
	"log"

	"yunyuyuan/anypaste/internal/model"
)

func main() {
	db, err := model.Open("data.db")
	if err != nil {
		log.Fatalf("failed to open data.db: %v", err)
	}

	if err := db.AutoMigrate(&model.Paste{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	log.Println("migration complete: pastes table created in data.db")
}
