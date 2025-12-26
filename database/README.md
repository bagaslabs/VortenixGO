# Item Database

Package untuk membaca dan mengelola data item dari file `items.dat` Growtopia.

## Fitur

- ✅ Parsing file binary `items.dat`
- ✅ Dekripsi nama item (menggunakan SECRET key)
- ✅ Support semua versi items.dat (v11-v24+)
- ✅ Thread-safe dengan mutex
- ✅ Query item berdasarkan ID, nama, rarity, clothing type
- ✅ Search item dengan partial match
- ✅ Item flags parsing (untradeable, foreground, dll)

## Instalasi

```bash
# Pastikan file items.dat ada di root project
# Atau tentukan path custom saat initialize
```

## Penggunaan

### 1. Initialize Database

```go
import "vortenixgo/database"

func main() {
    // Load items.dat
    err := database.InitializeItemDatabase("items.dat")
    if err != nil {
        log.Fatal(err)
    }
    
    // Get global instance
    db := database.GetGlobalItemDB()
    fmt.Printf("Loaded %d items\n", db.ItemCount)
}
```

### 2. Get Item by ID

```go
// Get item dengan ID tertentu
dirt := db.GetItem(2)
if dirt != nil {
    fmt.Printf("Name: %s\n", dirt.Name)
    fmt.Printf("Rarity: %d\n", dirt.Rarity)
}
```

### 3. Get Item by Name

```go
// Get item berdasarkan nama (case-insensitive)
worldLock := db.GetItemByName("World Lock")
if worldLock != nil {
    fmt.Printf("ID: %d\n", worldLock.ID)
    fmt.Printf("Untradeable: %v\n", worldLock.Flags.Untradeable)
}
```

### 4. Search Items

```go
// Cari item yang mengandung kata tertentu
seeds := db.SearchItems("Seed")
for _, item := range seeds {
    fmt.Printf("- %s (ID: %d)\n", item.Name, item.ID)
}
```

### 5. Filter by Rarity

```go
// Get semua item dengan rarity tertentu
legendaryItems := db.GetItemsByRarity(999)
fmt.Printf("Found %d legendary items\n", len(legendaryItems))
```

### 6. Filter by Clothing Type

```go
// Get semua item clothing dengan type tertentu
// 0 = none, 1 = hat, 2 = shirt, 3 = pants, 4 = feet, 5 = face, 6 = hand, 7 = back, 8 = hair, 9 = neck
hats := db.GetItemsByClothingType(1)
fmt.Printf("Found %d hats\n", len(hats))
```

## Struktur Data

### Item

```go
type Item struct {
    ID                  uint32
    Name                string
    Description         string
    Rarity              uint16
    ActionType          uint8
    ClothingType        uint8
    GrowTime            uint32
    Flags               ItemFlag
    // ... dan banyak field lainnya
}
```

### ItemFlag

```go
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
```

## Helper Functions

```go
// Check if item is wearable
if database.IsWearable(item) {
    fmt.Println("This is a clothing item")
}

// Check if item is a seed
if database.IsSeed(item) {
    growTime := database.GetSeedGrowTime(item)
    fmt.Printf("Grow time: %d seconds\n", growTime)
}

// Check if item is a block
if database.IsBlock(item) {
    fmt.Println("This is a placeable block")
}

// Get formatted item info
info := database.GetItemInfo(db, 242) // World Lock
fmt.Println(info)
```

## Testing

```bash
# Run tests
go test ./database

# Run benchmarks
go test -bench=. ./database

# Run specific test
go test -run TestGetItem ./database
```

## Format File items.dat

File items.dat memiliki struktur binary:
1. Version (uint16) - Versi format file
2. Item Count (uint32) - Jumlah total item
3. Items[] - Array of items dengan struktur kompleks

Nama item di-encrypt menggunakan XOR cipher dengan key `PBG892FXX982ABC*`.

## Performance

- Loading ~15,000 items: ~50-100ms
- GetItem by ID: O(1) - instant
- GetItemByName: O(n) - linear search
- SearchItems: O(n) - linear search dengan string matching

## Thread Safety

Semua operasi read menggunakan `RWMutex` untuk thread-safety:
- Multiple goroutines bisa read secara bersamaan
- Write operation (Load) akan lock exclusive

## Notes

- File items.dat harus dari versi Growtopia yang sama dengan bot
- Jika items.dat tidak ditemukan, fungsi akan return error
- Database di-load sekali saat initialize (singleton pattern)
