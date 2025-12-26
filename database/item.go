package database

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const SECRET = "PBG892FXX982ABC*"

// ItemFlag represents various boolean flags for an item
type ItemFlag struct {
	Flippable   bool
	Editable    bool
	Seedless    bool
	Permanent   bool
	Dropless    bool
	NoSelf      bool
	NoShadow    bool
	WorldLocked bool
	Beta        bool
	AutoPickup  bool
	ModFlag     bool
	RandomGrow  bool
	Public      bool
	Foreground  bool
	Holiday     bool
	Untradeable bool
}

// FromBits creates ItemFlag from a 16-bit integer
func (f *ItemFlag) FromBits(bits uint16) {
	f.Flippable = bits&0x1 != 0
	f.Editable = bits&0x2 != 0
	f.Seedless = bits&0x4 != 0
	f.Permanent = bits&0x8 != 0
	f.Dropless = bits&0x10 != 0
	f.NoSelf = bits&0x20 != 0
	f.NoShadow = bits&0x40 != 0
	f.WorldLocked = bits&0x80 != 0
	f.Beta = bits&0x100 != 0
	f.AutoPickup = bits&0x200 != 0
	f.ModFlag = bits&0x400 != 0
	f.RandomGrow = bits&0x800 != 0
	f.Public = bits&0x1000 != 0
	f.Foreground = bits&0x2000 != 0
	f.Holiday = bits&0x4000 != 0
	f.Untradeable = bits&0x8000 != 0
}

// Item represents a single Growtopia item
type Item struct {
	ID                 uint32
	Flags              ItemFlag
	ActionType         uint8
	Material           uint8
	Name               string
	TextureFileName    string
	TextureHash        uint32
	CookingIngredient  uint32
	VisualEffect       uint8
	TextureX           uint8
	TextureY           uint8
	RenderType         uint8
	IsStripeyWallpaper uint8
	CollisionType      uint8
	BlockHealth        uint8
	DropChance         uint32
	ClothingType       uint8
	Rarity             uint16
	MaxItem            uint8
	FileName           string
	FileHash           uint32
	AudioVolume        uint32
	PetName            string
	PetPrefix          string
	PetSuffix          string
	PetAbility         string
	SeedBaseSprite     uint8
	SeedOverlaySprite  uint8
	TreeBaseSprite     uint8
	TreeOverlaySprite  uint8
	BaseColor          uint32
	OverlayColor       uint32
	Ingredient         uint32
	GrowTime           uint32
	IsRayman           uint16
	ExtraOptions       string
	TexturePath2       string
	ExtraOption2       string
	PunchOption        string
	Description        string
}

// ItemDatabase holds all items from items.dat
type ItemDatabase struct {
	Version   uint16
	ItemCount uint32
	Items     map[uint32]*Item
	Loaded    bool
	mu        sync.RWMutex
}

// NewItemDatabase creates a new empty item database
func NewItemDatabase() *ItemDatabase {
	return &ItemDatabase{
		Items:  make(map[uint32]*Item),
		Loaded: false,
	}
}

// LoadFromFile loads items.dat from a file path
func (db *ItemDatabase) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	return db.LoadFromMemory(data)
}

// LoadFromMemory loads items.dat from byte slice
func (db *ItemDatabase) LoadFromMemory(data []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	reader := bytes.NewReader(data)

	// Read version
	if err := binary.Read(reader, binary.LittleEndian, &db.Version); err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}

	// Read item count
	if err := binary.Read(reader, binary.LittleEndian, &db.ItemCount); err != nil {
		return fmt.Errorf("failed to read item count: %w", err)
	}

	// Read all items
	for i := uint32(0); i < db.ItemCount; i++ {
		item, err := readItem(reader, db.Version)
		if err != nil {
			return fmt.Errorf("failed to read item %d: %w", i, err)
		}

		if item.ID != i {
			return fmt.Errorf("item ID mismatch: expected %d, got %d", i, item.ID)
		}

		db.Items[item.ID] = item
	}

	db.Loaded = true
	return nil
}

// GetItem retrieves an item by ID
func (db *ItemDatabase) GetItem(id uint32) *Item {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.Items[id]
}

// GetItemByName retrieves an item by name (case-insensitive)
func (db *ItemDatabase) GetItemByName(name string) *Item {
	db.mu.RLock()
	defer db.mu.RUnlock()

	nameLower := strings.ToLower(name)
	for _, item := range db.Items {
		if strings.ToLower(item.Name) == nameLower {
			return item
		}
	}
	return nil
}

// GetItemsByRarity retrieves all items with a specific rarity
func (db *ItemDatabase) GetItemsByRarity(rarity uint16) []*Item {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var items []*Item
	for _, item := range db.Items {
		if item.Rarity == rarity {
			items = append(items, item)
		}
	}
	return items
}

// GetItemsByClothingType retrieves all items with a specific clothing type
func (db *ItemDatabase) GetItemsByClothingType(clothingType uint8) []*Item {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var items []*Item
	for _, item := range db.Items {
		if item.ClothingType == clothingType {
			items = append(items, item)
		}
	}
	return items
}

// SearchItems searches for items by partial name match
func (db *ItemDatabase) SearchItems(query string) []*Item {
	db.mu.RLock()
	defer db.mu.RUnlock()

	queryLower := strings.ToLower(query)
	var items []*Item
	for _, item := range db.Items {
		if strings.Contains(strings.ToLower(item.Name), queryLower) {
			items = append(items, item)
		}
	}
	return items
}

// Helper functions

func readItem(reader *bytes.Reader, version uint16) (*Item, error) {
	item := &Item{}

	// Read ID
	if err := binary.Read(reader, binary.LittleEndian, &item.ID); err != nil {
		return nil, err
	}

	// Read flags
	var flags uint16
	if err := binary.Read(reader, binary.LittleEndian, &flags); err != nil {
		return nil, err
	}
	item.Flags.FromBits(flags)

	// Read basic properties
	if err := binary.Read(reader, binary.LittleEndian, &item.ActionType); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.Material); err != nil {
		return nil, err
	}

	// Read encrypted name
	var err error
	item.Name, err = decipherItemName(reader, item.ID)
	if err != nil {
		return nil, err
	}

	// Read texture file name
	item.TextureFileName, err = readString(reader)
	if err != nil {
		return nil, err
	}

	// Continue reading properties
	if err := binary.Read(reader, binary.LittleEndian, &item.TextureHash); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.VisualEffect); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.CookingIngredient); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.TextureX); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.TextureY); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.RenderType); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.IsStripeyWallpaper); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.CollisionType); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.BlockHealth); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.DropChance); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.ClothingType); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.Rarity); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.MaxItem); err != nil {
		return nil, err
	}

	// Read file name
	item.FileName, err = readString(reader)
	if err != nil {
		return nil, err
	}

	if err := binary.Read(reader, binary.LittleEndian, &item.FileHash); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &item.AudioVolume); err != nil {
		return nil, err
	}

	// Read pet strings
	item.PetName, _ = readString(reader)
	item.PetPrefix, _ = readString(reader)
	item.PetSuffix, _ = readString(reader)
	item.PetAbility, _ = readString(reader)

	// Read sprite data
	binary.Read(reader, binary.LittleEndian, &item.SeedBaseSprite)
	binary.Read(reader, binary.LittleEndian, &item.SeedOverlaySprite)
	binary.Read(reader, binary.LittleEndian, &item.TreeBaseSprite)
	binary.Read(reader, binary.LittleEndian, &item.TreeOverlaySprite)
	binary.Read(reader, binary.LittleEndian, &item.BaseColor)
	binary.Read(reader, binary.LittleEndian, &item.OverlayColor)
	binary.Read(reader, binary.LittleEndian, &item.Ingredient)
	binary.Read(reader, binary.LittleEndian, &item.GrowTime)

	// Skip 2 bytes
	reader.Seek(2, io.SeekCurrent)
	binary.Read(reader, binary.LittleEndian, &item.IsRayman)

	// Read extra options
	item.ExtraOptions, _ = readString(reader)
	item.TexturePath2, _ = readString(reader)
	item.ExtraOption2, _ = readString(reader)

	// Skip 80 bytes
	reader.Seek(80, io.SeekCurrent)

	// Version-specific fields
	if version >= 11 {
		item.PunchOption, _ = readString(reader)
	}
	if version >= 12 {
		reader.Seek(13, io.SeekCurrent)
	}
	if version >= 13 {
		reader.Seek(4, io.SeekCurrent)
	}
	if version >= 14 {
		reader.Seek(4, io.SeekCurrent)
	}
	if version >= 15 {
		reader.Seek(25, io.SeekCurrent)
		readString(reader) // Discard
	}
	if version >= 16 {
		readString(reader) // Discard
	}
	if version >= 17 {
		reader.Seek(4, io.SeekCurrent)
	}
	if version >= 18 {
		reader.Seek(4, io.SeekCurrent)
	}
	if version >= 19 {
		reader.Seek(9, io.SeekCurrent)
	}
	if version >= 21 {
		reader.Seek(2, io.SeekCurrent)
	}
	if version >= 22 {
		item.Description, _ = readString(reader)
	}
	if version >= 23 {
		reader.Seek(4, io.SeekCurrent)
	}
	if version >= 24 {
		reader.Seek(1, io.SeekCurrent)
	}

	return item, nil
}

func readString(reader *bytes.Reader) (string, error) {
	var length uint16
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return "", err
	}

	if length == 0 {
		return "", nil
	}

	buf := make([]byte, length)
	if _, err := reader.Read(buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

func decipherItemName(reader *bytes.Reader, itemID uint32) (string, error) {
	var length uint16
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return "", err
	}

	if length == 0 {
		return "", nil
	}

	buf := make([]byte, length)
	if _, err := reader.Read(buf); err != nil {
		return "", err
	}

	// Decrypt the name
	secretBytes := []byte(SECRET)
	for i := uint16(0); i < length; i++ {
		charPos := (uint32(i) + itemID) % uint32(len(secretBytes))
		buf[i] ^= secretBytes[charPos]
	}

	return string(buf), nil
}
