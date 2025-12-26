package database

import (
	"log"
	"sync"
)

var (
	// GlobalItemDB is the global item database instance
	GlobalItemDB *ItemDatabase
	once         sync.Once
)

// InitializeItemDatabase loads the items.dat file and initializes the global database
func InitializeItemDatabase(path string) error {
	var err error
	once.Do(func() {
		GlobalItemDB = NewItemDatabase()
		err = GlobalItemDB.LoadFromFile(path)
		if err != nil {
			log.Printf("[Database] Failed to load items.dat: %v", err)
			return
		}
		log.Printf("[Database] Loaded %d items (version %d)", GlobalItemDB.ItemCount, GlobalItemDB.Version)
	})
	return err
}

// GetGlobalItemDB returns the global item database instance
func GetGlobalItemDB() *ItemDatabase {
	return GlobalItemDB
}
