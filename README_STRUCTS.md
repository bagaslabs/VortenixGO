# VortenixGO - Bot Structs & Usage Recommendation

Project ini menggunakan bahasa pemrograman Go untuk mensimulasikan bot game dengan struktur data yang terorganisir. Berikut adalah penjelasan mengenai struktur data baru yang telah ditambahkan.

## Struktur Data Utama

### 1. `Local` Struct
Struktur ini menyimpan seluruh status internal untuk satu sesi bot, termasuk posisi, inventaris, dan data dunia.

**Field Utama:**
- `Name`, `Level`, `GemCount`: Informasi dasar karakter.
- `Pos`, `PosX`, `PosY`: Data posisi karakter di dalam world.
- `Inventory`: Daftar item yang dimiliki bot (`Items []Inventory`).
- `Players`: Daftar pemain lain yang terlihat di world yang sama.
- `World`: Informasi dunia tempat bot berada.

### 2. `Players` Struct
Menyimpan data pemain lain (Remote Players) yang berada di sekitar bot.

### 3. `Inventory` Struct
Menyimpan ID item dan jumlah (`Count`) yang dimiliki bot.

---

## Cara Mengakses Data

Semua struct ini berada di package `bot`. Instance `Local` sekarang telah diintegrasikan ke dalam struct `Bot` utama.

### Mengakses dari dalam Code (Go)

Anda dapat mengakses data tersebut melalui instance `Bot` yang sedang berjalan:

```go
// Contoh mengakses Gem Count
gems := myBot.Local.GemCount

// Contoh iterasi inventaris
for _, item := range myBot.Local.Items {
    fmt.Printf("Item ID: %d, Jumlah: %d\n", item.ID, item.Count)
}

// Contoh mendapatkan koordinat
x := myBot.Local.Pos.X
y := myBot.Local.Pos.Y
```

### Mengakses dari Frontend (JSON)

Karena setiap field telah dilengkapi dengan tag `json`, data ini akan otomatis terkirim ke frontend jika Anda menggunakan WebSocket atau API yang mengembalikan struct `Bot`.

**Contoh Struktur JSON:**
```json
{
  "name": "BotName",
  "local": {
    "gem_count": 1000,
    "level": 15,
    "pos": { "x": 1024, "y": 512 },
    "items": [
      { "id": 2, "count": 200, "is_active": false }
    ]
  }
}
```

---

## Update Terbaru (Fixes)

1.  **`handler_game_packet.go`**:
    - Memperbaiki syntax error pada `handleGamePacket_CallFunction` (masalah `else` yang terpisah dari block `if`).
    - Mengimplementasikan logic `OnSuperMainStartAcceptLogonHrdxs...` untuk mengirim packet `enter_game` dan `APP_CHECK_RESPONSE` secara otomatis saat login berhasil.
2.  **`structs.go`**:
    - Penambahan `Vector2`, `Players`, `Inventory`, `World`, `ServerLocal`, dan `Local`.
3.  **`bot.go`**:
    - Menambahkan field `Local` ke dalam struct `Bot` agar data mudah diakses.
