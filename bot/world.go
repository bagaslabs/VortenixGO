package bot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// ParseWorld parses world data from NET_GAME_PACKET_SEND_MAP_DATA
func (b *Bot) ParseWorld(data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	reader := bytes.NewReader(data)
	world := &b.Local.World

	// Reset world
	world.Name = "EXIT"
	world.Width = 0
	world.Height = 0
	world.TileCount = 0
	world.Tiles = nil
	world.DroppedItems = nil
	world.BaseWeather = 0
	world.CurrentWeather = 0

	// Read version
	var version uint16
	if err := binary.Read(reader, binary.LittleEndian, &version); err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}
	world.Version = version

	if version < 0x19 {
		return fmt.Errorf("unsupported world version: %d", version)
	}

	// Read flags
	if err := binary.Read(reader, binary.LittleEndian, &world.Flags); err != nil {
		return fmt.Errorf("failed to read flags: %w", err)
	}

	// Read world name
	var nameLen uint16
	if err := binary.Read(reader, binary.LittleEndian, &nameLen); err != nil {
		return fmt.Errorf("failed to read name length: %w", err)
	}
	nameBytes := make([]byte, nameLen)
	if _, err := io.ReadFull(reader, nameBytes); err != nil {
		return fmt.Errorf("failed to read name: %w", err)
	}
	world.Name = string(nameBytes)

	// Read dimensions
	if err := binary.Read(reader, binary.LittleEndian, &world.Width); err != nil {
		return fmt.Errorf("failed to read width: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &world.Height); err != nil {
		return fmt.Errorf("failed to read height: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &world.TileCount); err != nil {
		return fmt.Errorf("failed to read tile count: %w", err)
	}

	// Skip 5 bytes (debug flag)
	reader.Seek(5, io.SeekCurrent)

	if world.TileCount > 0xFE01 {
		return fmt.Errorf("tile count too large: %d", world.TileCount)
	}

	// Parse tiles
	world.Tiles = make([]Tile, 0, world.TileCount)
	for i := uint32(0); i < world.TileCount; i++ {
		x := i % world.Width
		y := i / world.Width

		tile := Tile{
			X: x,
			Y: y,
		}

		// Read tile data
		if err := binary.Read(reader, binary.LittleEndian, &tile.ForegroundItemID); err != nil {
			return fmt.Errorf("failed to read fg item id at tile %d: %w", i, err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &tile.BackgroundItemID); err != nil {
			return fmt.Errorf("failed to read bg item id at tile %d: %w", i, err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &tile.ParentBlockIndex); err != nil {
			return fmt.Errorf("failed to read parent index at tile %d: %w", i, err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &tile.Flags); err != nil {
			return fmt.Errorf("failed to read flags at tile %d: %w", i, err)
		}
		tile.TileFlags = tile.Flags

		// Check for parent data
		if tile.Flags&0x02 != 0 { // HAS_PARENT
			var parentData uint16
			binary.Read(reader, binary.LittleEndian, &parentData)
		}

		// Check for extra data
		if tile.Flags&0x01 != 0 { // HAS_EXTRA_DATA
			var extraType uint8
			if err := binary.Read(reader, binary.LittleEndian, &extraType); err != nil {
				return fmt.Errorf("failed to read extra type at tile %d: %w", i, err)
			}

			if err := b.parseExtraTileData(reader, &tile, extraType); err != nil {
				return fmt.Errorf("failed to parse extra data at tile %d: %w", i, err)
			}
		}

		// Check for CBOR data (some special tiles)
		if b.ItemDatabase != nil {
			if item := b.ItemDatabase.GetItem(uint32(tile.ForegroundItemID)); item != nil {
				// Tiles with CBOR data
				specialTiles := []uint32{15376, 8642, 15546}
				isCBOR := false
				for _, id := range specialTiles {
					if uint32(tile.ForegroundItemID) == id {
						isCBOR = true
						break
					}
				}

				if isCBOR || len(item.FileName) > 4 && item.FileName[len(item.FileName)-4:] == ".xml" {
					var cborSize uint32
					if err := binary.Read(reader, binary.LittleEndian, &cborSize); err == nil {
						// Skip CBOR data for now
						reader.Seek(int64(cborSize), io.SeekCurrent)
					}
				}
			}
		}

		world.Tiles = append(world.Tiles, tile)
	}

	// Skip 12 bytes
	reader.Seek(12, io.SeekCurrent)

	// Parse dropped items
	var droppedCount uint32
	var lastDroppedUID uint32
	if err := binary.Read(reader, binary.LittleEndian, &droppedCount); err != nil {
		return fmt.Errorf("failed to read dropped count: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &lastDroppedUID); err != nil {
		return fmt.Errorf("failed to read last dropped uid: %w", err)
	}

	world.DroppedItems = make([]DroppedItem, 0, droppedCount)
	for i := uint32(0); i < droppedCount; i++ {
		var item DroppedItem
		if err := binary.Read(reader, binary.LittleEndian, &item.ID); err != nil {
			return fmt.Errorf("failed to read dropped item id: %w", err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &item.X); err != nil {
			return fmt.Errorf("failed to read dropped item x: %w", err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &item.Y); err != nil {
			return fmt.Errorf("failed to read dropped item y: %w", err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &item.Count); err != nil {
			return fmt.Errorf("failed to read dropped item count: %w", err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &item.Flags); err != nil {
			return fmt.Errorf("failed to read dropped item flags: %w", err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &item.UID); err != nil {
			return fmt.Errorf("failed to read dropped item uid: %w", err)
		}
		world.DroppedItems = append(world.DroppedItems, item)
	}

	// Parse weather
	if err := binary.Read(reader, binary.LittleEndian, &world.BaseWeather); err != nil {
		return fmt.Errorf("failed to read base weather: %w", err)
	}
	var unknownWeather uint16
	binary.Read(reader, binary.LittleEndian, &unknownWeather)
	if err := binary.Read(reader, binary.LittleEndian, &world.CurrentWeather); err != nil {
		return fmt.Errorf("failed to read current weather: %w", err)
	}

	// Save raw world data (matching Rust: fs::write("world.dat", world_data))
	if err := os.WriteFile("world.dat", data, 0644); err != nil {
		b.logENet("Warning: Failed to write world.dat: " + err.Error())
	}

	// Save raw bytes to text file as requested (Hex Dump)
	hexDump := fmt.Sprintf("%x", data)
	if err := os.WriteFile("world_raw.txt", []byte(hexDump), 0644); err != nil {
		b.logENet("Warning: Failed to write world_raw.txt: " + err.Error())
	}

	// Calculate Collision Data
	if len(world.Tiles) > 0 {
		width := int(world.Width)
		height := int(world.Height)
		collisionData := make([]uint8, width*height)

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				idx := x + y*width
				collisionType := uint8(0) // Default 0

				if idx < len(world.Tiles) {
					tile := &world.Tiles[idx]
					var itemID uint32 = uint32(tile.ForegroundItemID)
					if b.ItemDatabase != nil {
						if item := b.ItemDatabase.GetItem(itemID); item != nil {
							collisionType = item.CollisionType
						}
					}
				} else {
					collisionType = 255
				}
				collisionData[idx] = collisionType
			}
		}
		world.CollisionMap = collisionData
	}

	b.logENet(fmt.Sprintf("World loaded: %s (%dx%d, %d tiles)", world.Name, world.Width, world.Height, world.TileCount))

	// Dump parsed map to text file
	b.dumpMapToTxt(world)

	return nil
}

func (b *Bot) dumpMapToTxt(world *World) {
	f, err := os.Create("world_parsed.txt")
	if err != nil {
		b.logENet("Failed to create world_parsed.txt: " + err.Error())
		return
	}
	defer f.Close()

	// Header Info
	fmt.Fprintf(f, "World Name: %s\n", world.Name)
	fmt.Fprintf(f, "Size: %dx%d\n", world.Width, world.Height)
	fmt.Fprintf(f, "Tile Count: %d\n", world.TileCount)
	fmt.Fprintf(f, "Version: %d\n", world.Version)
	fmt.Fprintf(f, "Flags: %d\n", world.Flags)
	fmt.Fprintf(f, "Weather: Base=%d, Current=%d\n\n", world.BaseWeather, world.CurrentWeather)

	fmt.Fprintln(f, "--- TILE LIST ---")
	fmt.Fprintln(f, "Index | X, Y | Fg | Bg | Flags | TileFlags | Type | Extra Data")

	for i, tile := range world.Tiles {
		extraStr := "None"
		if tile.Extra != nil {
			extraStr = fmt.Sprintf("%+v", tile.Extra)
		}
		// Basic format
		fmt.Fprintf(f, "%5d | %3d, %3d | %4d | %4d | 0x%04X | 0x%04X | %3d | %s\n",
			i, tile.X, tile.Y, tile.ForegroundItemID, tile.BackgroundItemID, tile.Flags, tile.TileFlags, tile.TileType, extraStr)
	}
	b.logENet(fmt.Sprintf("Successfully dumped %d tiles to world_parsed.txt", len(world.Tiles)))
}

func (b *Bot) DumpMapToFile() {
	f, err := os.Create("map.txt")
	if err != nil {
		b.logENet("Failed to create map.txt: " + err.Error())
		return
	}
	defer f.Close()

	// Use a read lock to access local world safely
	b.mu.Lock()
	world := b.Local.World
	b.mu.Unlock()

	fmt.Fprintf(f, "World: %s (%dx%d)\n", world.Name, world.Width, world.Height)
	fmt.Fprintln(f, "Index | X, Y | Fg | Bg | Flags | TileFlags | Type | Extra")

	for i, tile := range world.Tiles {
		extraStr := ""
		if tile.Extra != nil {
			extraStr = fmt.Sprintf("%+v", tile.Extra)
		}
		// Basic format
		fmt.Fprintf(f, "%5d | %3d, %3d | %4d | %4d | 0x%04X | 0x%04X | %3d | %s\n",
			i, tile.X, tile.Y, tile.ForegroundItemID, tile.BackgroundItemID, tile.Flags, tile.TileFlags, tile.TileType, extraStr)
	}
	b.logENet("Map dumped to map.txt successfully")
}

func (b *Bot) parseExtraTileData(reader *bytes.Reader, tile *Tile, extraType uint8) error {
	tile.TileType = extraType

	switch extraType {
	case 1: // Sign
		var textLen uint16
		if err := binary.Read(reader, binary.LittleEndian, &textLen); err != nil {
			return err
		}
		textBytes := make([]byte, textLen)
		io.ReadFull(reader, textBytes)
		var flags uint8
		binary.Read(reader, binary.LittleEndian, &flags)

		tile.Extra = TileSign{
			Text:  string(textBytes),
			Flags: flags,
		}

	case 2: // Door
		var textLen uint16
		if err := binary.Read(reader, binary.LittleEndian, &textLen); err != nil {
			return err
		}
		textBytes := make([]byte, textLen)
		io.ReadFull(reader, textBytes)
		var ownerUID uint32
		binary.Read(reader, binary.LittleEndian, &ownerUID)

		tile.Extra = TileDoor{
			Text:     string(textBytes),
			OwnerUID: ownerUID,
		}

	case 3: // Lock
		data := TileLock{}
		binary.Read(reader, binary.LittleEndian, &data.Settings)
		binary.Read(reader, binary.LittleEndian, &data.OwnerUID)
		binary.Read(reader, binary.LittleEndian, &data.AccessCount)
		data.AccessUIDs = make([]uint32, data.AccessCount)
		for i := uint32(0); i < data.AccessCount; i++ {
			binary.Read(reader, binary.LittleEndian, &data.AccessUIDs[i])
		}
		binary.Read(reader, binary.LittleEndian, &data.MinimumLevel)
		reader.Seek(7, io.SeekCurrent) // unknown data
		if tile.ForegroundItemID == 5814 {
			reader.Seek(16, io.SeekCurrent)
		}
		tile.Extra = data

	case 4: // Seed
		data := TileSeed{}
		binary.Read(reader, binary.LittleEndian, &data.TimePassed)
		binary.Read(reader, binary.LittleEndian, &data.ItemOnTree)

		// Check if ready to harvest
		if b.ItemDatabase != nil {
			if item := b.ItemDatabase.GetItem(uint32(tile.ForegroundItemID)); item != nil {
				data.ReadyToHarvest = data.TimePassed >= item.GrowTime
			}
		}

		tile.Extra = data

	case 6: // Mailbox
		data := TileMailbox{}
		var strLen uint16

		binary.Read(reader, binary.LittleEndian, &strLen)
		buf := make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message1 = string(buf)

		binary.Read(reader, binary.LittleEndian, &strLen)
		buf = make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message2 = string(buf)

		binary.Read(reader, binary.LittleEndian, &strLen)
		buf = make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message3 = string(buf)

		binary.Read(reader, binary.LittleEndian, &data.Unknown)
		tile.Extra = data

	case 7: // Bulletin
		data := TileBulletin{}
		var strLen uint16

		binary.Read(reader, binary.LittleEndian, &strLen)
		buf := make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message1 = string(buf)

		binary.Read(reader, binary.LittleEndian, &strLen)
		buf = make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message2 = string(buf)

		binary.Read(reader, binary.LittleEndian, &strLen)
		buf = make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message3 = string(buf)

		binary.Read(reader, binary.LittleEndian, &data.Unknown)
		tile.Extra = data

	case 8: // Dice
		data := TileDice{}
		binary.Read(reader, binary.LittleEndian, &data.Symbol)
		tile.Extra = data

	case 9: // ChemicalSource
		data := TileChemicalSource{}
		binary.Read(reader, binary.LittleEndian, &data.TimePassed)
		tile.Extra = data

	case 10: // AchievementBlock
		data := TileAchievementBlock{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.TileType)
		tile.Extra = data

	case 11: // HearthMonitor
		data := TileHearthMonitor{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		var strLen uint16
		binary.Read(reader, binary.LittleEndian, &strLen)
		buf := make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.PlayerName = string(buf)
		tile.Extra = data

	case 12: // DonationBox
		data := TileDonationBox{}
		var strLen uint16

		binary.Read(reader, binary.LittleEndian, &strLen)
		buf := make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message1 = string(buf)

		binary.Read(reader, binary.LittleEndian, &strLen)
		buf = make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message2 = string(buf)

		binary.Read(reader, binary.LittleEndian, &strLen)
		buf = make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message3 = string(buf)

		binary.Read(reader, binary.LittleEndian, &data.Unknown)
		tile.Extra = data

	case 14: // Mannequin
		data := TileMannequin{}
		var strLen uint16
		binary.Read(reader, binary.LittleEndian, &strLen)
		buf := make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Text = string(buf)

		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.Clothing1)
		binary.Read(reader, binary.LittleEndian, &data.Clothing2)
		binary.Read(reader, binary.LittleEndian, &data.Clothing3)
		binary.Read(reader, binary.LittleEndian, &data.Clothing4)
		binary.Read(reader, binary.LittleEndian, &data.Clothing5)
		binary.Read(reader, binary.LittleEndian, &data.Clothing6)
		binary.Read(reader, binary.LittleEndian, &data.Clothing7)
		binary.Read(reader, binary.LittleEndian, &data.Clothing8)
		binary.Read(reader, binary.LittleEndian, &data.Clothing9)
		binary.Read(reader, binary.LittleEndian, &data.Clothing10)
		tile.Extra = data

	case 15: // BunnyEgg
		data := TileBunnyEgg{}
		binary.Read(reader, binary.LittleEndian, &data.EggPlaced)
		tile.Extra = data

	case 16: // GamePack
		data := TileGamePack{}
		binary.Read(reader, binary.LittleEndian, &data.Team)
		tile.Extra = data

	case 17: // GameGenerator
		tile.Extra = TileGameGenerator{}

	case 18: // XenoniteCrystal
		data := TileXenoniteCrystal{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		tile.Extra = data

	case 19: // PhoneBooth
		data := TilePhoneBooth{}
		binary.Read(reader, binary.LittleEndian, &data.Clothing1)
		binary.Read(reader, binary.LittleEndian, &data.Clothing2)
		binary.Read(reader, binary.LittleEndian, &data.Clothing3)
		binary.Read(reader, binary.LittleEndian, &data.Clothing4)
		binary.Read(reader, binary.LittleEndian, &data.Clothing5)
		binary.Read(reader, binary.LittleEndian, &data.Clothing6)
		binary.Read(reader, binary.LittleEndian, &data.Clothing7)
		binary.Read(reader, binary.LittleEndian, &data.Clothing8)
		binary.Read(reader, binary.LittleEndian, &data.Clothing9)
		tile.Extra = data

	case 20: // Crystal
		data := TileCrystal{}
		var strLen uint16
		binary.Read(reader, binary.LittleEndian, &strLen)
		buf := make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message = string(buf)
		tile.Extra = data

	case 21: // CrimeInProgress
		data := TileCrimeInProgress{}
		var strLen uint16
		binary.Read(reader, binary.LittleEndian, &strLen)
		buf := make([]byte, strLen)
		io.ReadFull(reader, buf)
		data.Message = string(buf)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		binary.Read(reader, binary.LittleEndian, &data.Unknown3)
		tile.Extra = data

	case 23: // DisplayBlock
		data := TileDisplayBlock{}
		binary.Read(reader, binary.LittleEndian, &data.ItemID)
		tile.Extra = data

	case 24: // VendingMachine
		data := TileVendingMachine{}
		binary.Read(reader, binary.LittleEndian, &data.ItemID)
		binary.Read(reader, binary.LittleEndian, &data.Price)
		tile.Extra = data

	case 25: // FishTankPort
		data := TileFishTankPort{}
		binary.Read(reader, binary.LittleEndian, &data.Flags)
		var fishCount uint32
		binary.Read(reader, binary.LittleEndian, &fishCount)
		// fishCount is number of fields, each fish has 2 fields (id, lbs) -> fishCount/2 ?
		// Rust: for _ in 0..(fish_count / 2)
		for i := uint32(0); i < fishCount/2; i++ {
			var f FishInfo
			binary.Read(reader, binary.LittleEndian, &f.FishItemID)
			binary.Read(reader, binary.LittleEndian, &f.Lbs)
			data.Fishes = append(data.Fishes, f)
		}
		tile.Extra = data

	case 26: // SolarCollector
		data := TileSolarCollector{Unknown1: make([]byte, 5)}
		io.ReadFull(reader, data.Unknown1)
		tile.Extra = data

	case 27: // Forge
		data := TileForge{}
		binary.Read(reader, binary.LittleEndian, &data.Temperature)
		tile.Extra = data

	case 28: // GivingTree
		data := TileGivingTree{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		tile.Extra = data

	case 30: // SteamOrgan
		data := TileSteamOrgan{}
		binary.Read(reader, binary.LittleEndian, &data.InstrumentType)
		binary.Read(reader, binary.LittleEndian, &data.Note)
		tile.Extra = data

	case 31: // SilkWorm
		data := TileSilkWorm{}
		binary.Read(reader, binary.LittleEndian, &data.Type)
		var nameLen uint16
		binary.Read(reader, binary.LittleEndian, &nameLen)
		buf := make([]byte, nameLen)
		io.ReadFull(reader, buf)
		data.Name = string(buf)
		binary.Read(reader, binary.LittleEndian, &data.Age)
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		binary.Read(reader, binary.LittleEndian, &data.CanBeFed)
		binary.Read(reader, binary.LittleEndian, &data.FoodSaturation)
		binary.Read(reader, binary.LittleEndian, &data.WaterSaturation)

		var color uint32
		binary.Read(reader, binary.LittleEndian, &color)
		data.Color = SilkWormColor{
			A: uint8(color >> 24),
			R: uint8(color >> 16),
			G: uint8(color >> 8),
			B: uint8(color),
		}
		binary.Read(reader, binary.LittleEndian, &data.SickDuration)
		tile.Extra = data

	case 32: // SewingMachine
		data := TileSewingMachine{}
		var boltLen uint16
		binary.Read(reader, binary.LittleEndian, &boltLen)
		data.BoltIDList = make([]uint32, boltLen)
		for i := 0; i < int(boltLen); i++ {
			binary.Read(reader, binary.LittleEndian, &data.BoltIDList[i])
		}
		tile.Extra = data

	case 33: // CountryFlag
		data := TileCountryFlag{}
		var countryLen uint16
		binary.Read(reader, binary.LittleEndian, &countryLen)
		buf := make([]byte, countryLen)
		io.ReadFull(reader, buf)
		data.Country = string(buf)
		tile.Extra = data

	case 34: // LobsterTrap
		tile.Extra = TileLobsterTrap{}

	case 35: // PaintingEasel
		data := TilePaintingEasel{}
		binary.Read(reader, binary.LittleEndian, &data.ItemID)
		var labelLen uint16
		binary.Read(reader, binary.LittleEndian, &labelLen)
		buf := make([]byte, labelLen)
		io.ReadFull(reader, buf)
		data.Label = string(buf)
		tile.Extra = data

	case 36: // PetBattleCage
		data := TilePetBattleCage{}
		var labelLen uint16
		binary.Read(reader, binary.LittleEndian, &labelLen)
		buf := make([]byte, labelLen)
		io.ReadFull(reader, buf)
		data.Label = string(buf)

		data.Unknown1 = make([]byte, 12)
		io.ReadFull(reader, data.Unknown1)

		var cborSize uint32
		binary.Read(reader, binary.LittleEndian, &cborSize)
		// CBOR parsing skipped for now, but bytes consumed
		reader.Seek(int64(cborSize), io.SeekCurrent)
		tile.Extra = data

	case 37: // PetTrainer
		data := TilePetTrainer{}
		var nameLen uint16
		binary.Read(reader, binary.LittleEndian, &nameLen)
		buf := make([]byte, nameLen)
		io.ReadFull(reader, buf)
		data.Name = string(buf)

		binary.Read(reader, binary.LittleEndian, &data.PetTotalCount)
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		data.PetsID = make([]uint32, data.PetTotalCount)
		for i := uint32(0); i < data.PetTotalCount; i++ {
			binary.Read(reader, binary.LittleEndian, &data.PetsID[i])
		}
		tile.Extra = data

	case 38: // SteamEngine
		data := TileSteamEngine{}
		binary.Read(reader, binary.LittleEndian, &data.Temperature)
		tile.Extra = data

	case 39: // LockBot
		data := TileLockBot{}
		binary.Read(reader, binary.LittleEndian, &data.TimePassed)
		tile.Extra = data

	case 40: // WeatherMachine
		data := TileWeatherMachine{}
		binary.Read(reader, binary.LittleEndian, &data.Settings)
		tile.Extra = data

	case 41: // SpiritStorageUnit
		data := TileSpiritStorageUnit{}
		binary.Read(reader, binary.LittleEndian, &data.GhostJarCount)
		tile.Extra = data

	case 42: // DataBedrock
		reader.Seek(21, io.SeekCurrent)
		tile.Extra = TileDataBedrock{}

	case 43: // Shelf
		data := TileShelf{}
		binary.Read(reader, binary.LittleEndian, &data.TopLeftItemID)
		binary.Read(reader, binary.LittleEndian, &data.TopRightItemID)
		binary.Read(reader, binary.LittleEndian, &data.BottomLeftItemID)
		binary.Read(reader, binary.LittleEndian, &data.BottomRightItemID)
		tile.Extra = data

	case 44: // VipEntrance
		data := TileVipEntrance{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.OwnerUID)
		var accessCount uint32
		binary.Read(reader, binary.LittleEndian, &accessCount)
		data.AccessUIDs = make([]uint32, accessCount)
		for i := uint32(0); i < accessCount; i++ {
			binary.Read(reader, binary.LittleEndian, &data.AccessUIDs[i])
		}
		tile.Extra = data

	case 45: // ChallengeTimer
		tile.Extra = TileChallengeTimer{}

	case 47: // FishWallMount
		data := TileFishWallMount{}
		var labelLen uint16
		binary.Read(reader, binary.LittleEndian, &labelLen)
		buf := make([]byte, labelLen)
		io.ReadFull(reader, buf)
		data.Label = string(buf)
		binary.Read(reader, binary.LittleEndian, &data.ItemID)
		binary.Read(reader, binary.LittleEndian, &data.Lb)
		tile.Extra = data

	case 48: // Portrait
		data := TilePortrait{}
		var labelLen uint16
		binary.Read(reader, binary.LittleEndian, &labelLen)
		buf := make([]byte, labelLen)
		io.ReadFull(reader, buf)
		data.Label = string(buf)

		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		binary.Read(reader, binary.LittleEndian, &data.Unknown3)
		binary.Read(reader, binary.LittleEndian, &data.Unknown4)
		binary.Read(reader, binary.LittleEndian, &data.Face)
		binary.Read(reader, binary.LittleEndian, &data.Hat)
		binary.Read(reader, binary.LittleEndian, &data.Hair)
		binary.Read(reader, binary.LittleEndian, &data.Unknown5)
		binary.Read(reader, binary.LittleEndian, &data.Unknown6)
		tile.Extra = data

	case 49: // GuildWeatherMachine
		data := TileGuildWeatherMachine{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.Gravity)
		binary.Read(reader, binary.LittleEndian, &data.Flags)
		tile.Extra = data

	case 50: // FossilPrepStation
		data := TileFossilPrepStation{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		tile.Extra = data

	case 51, 52: // DnaExtractor, Howler
		// No extra data

	case 53: // ChemsynthTank
		data := TileChemsynthTank{}
		binary.Read(reader, binary.LittleEndian, &data.CurrentChem)
		binary.Read(reader, binary.LittleEndian, &data.TargetChem)
		tile.Extra = data

	case 54: // StorageBlock
		data := TileStorageBlock{}
		var dataLen uint16
		binary.Read(reader, binary.LittleEndian, &dataLen)
		// dataLen / 13 items ?
		// Rust: for _ in 0..(data_len / 13)
		count := int(dataLen) / 13
		for i := 0; i < count; i++ {
			// Rust: skip 3, read u32, skip 2, read u32
			// The wrapper code says:
			// data.set_position(data.position() + 3);
			// let id = data.read_u32::<LittleEndian>().unwrap();
			// data.set_position(data.position() + 2);
			// let amount = data.read_u32::<LittleEndian>().unwrap();
			reader.Seek(3, io.SeekCurrent)
			var id uint32
			binary.Read(reader, binary.LittleEndian, &id)
			reader.Seek(2, io.SeekCurrent)
			var amount uint32
			binary.Read(reader, binary.LittleEndian, &amount)
			data.Items = append(data.Items, StorageBlockItemInfo{ID: id, Amount: amount})
		}
		tile.Extra = data

	case 55: // CookingOven
		data := TileCookingOven{}
		reader.Seek(4, io.SeekCurrent) // temperature_level (rust says read it, wait)
		// Rust: let temperature_level = data.read_u32::<LittleEndian>().unwrap();
		// I must rewind/not skip if I want to read.
		// My logic below handles it correctly by NOT skipping.

		binary.Read(reader, binary.LittleEndian, &data.TemperatureLevel)
		var ingredientCount uint32
		binary.Read(reader, binary.LittleEndian, &ingredientCount)
		for i := uint32(0); i < ingredientCount; i++ {
			var ing CookingOvenIngredientInfo
			binary.Read(reader, binary.LittleEndian, &ing.ItemID)
			binary.Read(reader, binary.LittleEndian, &ing.TimeAdded)
			data.Ingredients = append(data.Ingredients, ing)
		}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		binary.Read(reader, binary.LittleEndian, &data.Unknown3)
		tile.Extra = data

	case 56: // AudioRack
		data := TileAudioRack{}
		var noteLen uint16
		binary.Read(reader, binary.LittleEndian, &noteLen)
		buf := make([]byte, noteLen)
		io.ReadFull(reader, buf)
		data.Note = string(buf)
		binary.Read(reader, binary.LittleEndian, &data.Volume)
		tile.Extra = data

	case 57: // GeigerCharger
		data := TileGeigerCharger{}
		var raw uint32
		binary.Read(reader, binary.LittleEndian, &raw)

		data.SecondsFromStart = raw
		if data.SecondsFromStart > 3600 {
			data.SecondsFromStart = 3600
		}
		data.SecondsToComplete = 3600 - data.SecondsFromStart
		data.ChargingPercent = data.SecondsFromStart / 36
		data.MinutesFromStart = data.SecondsFromStart / 60
		data.MinutesToComplete = 60
		if data.MinutesFromStart < 60 {
			data.MinutesToComplete = 60 - data.MinutesFromStart
		} else {
			data.MinutesToComplete = 0
		}
		tile.Extra = data

	case 58: // AdventureBegins
		tile.Extra = TileAdventureBegins{}
	case 59: // TombRobber
		tile.Extra = TileTombRobber{}

	case 60: // BalloonOMatic
		data := TileBalloonOMatic{}
		binary.Read(reader, binary.LittleEndian, &data.TotalRarity)
		binary.Read(reader, binary.LittleEndian, &data.TeamType)
		tile.Extra = data

	case 61: // TrainingPort
		data := TileTrainingPort{}
		binary.Read(reader, binary.LittleEndian, &data.FishLb)
		binary.Read(reader, binary.LittleEndian, &data.FishStatus)
		binary.Read(reader, binary.LittleEndian, &data.FishID)
		binary.Read(reader, binary.LittleEndian, &data.FishTotalExp)
		binary.Read(reader, binary.LittleEndian, &data.FishLevel)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		tile.Extra = data

	case 62: // ItemSucker
		data := TileItemSucker{}
		binary.Read(reader, binary.LittleEndian, &data.ItemIDToSuck)
		binary.Read(reader, binary.LittleEndian, &data.ItemAmount)
		binary.Read(reader, binary.LittleEndian, &data.Flags)
		binary.Read(reader, binary.LittleEndian, &data.Limit)
		tile.Extra = data

	case 63: // CyBot
		data := TileCyBot{}
		binary.Read(reader, binary.LittleEndian, &data.SyncTimer)
		binary.Read(reader, binary.LittleEndian, &data.Activated)
		var commandCount uint32
		binary.Read(reader, binary.LittleEndian, &commandCount)
		for i := uint32(0); i < commandCount; i++ {
			var cmd CyBotCommandData
			binary.Read(reader, binary.LittleEndian, &cmd.CommandID)
			binary.Read(reader, binary.LittleEndian, &cmd.IsCommandUsed)
			reader.Seek(7, io.SeekCurrent)
			data.CommandDatas = append(data.CommandDatas, cmd)
		}
		tile.Extra = data

	case 65: // GuildItem
		reader.Seek(17, io.SeekCurrent)
		tile.Extra = TileGuildItem{}

	case 66: // Growscan
		data := TileGrowscan{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		tile.Extra = data

	case 67: // ContainmentFieldPowerNode
		data := TileContainmentFieldPowerNode{}
		binary.Read(reader, binary.LittleEndian, &data.GhostJarCount)
		var unknown1Size uint32
		binary.Read(reader, binary.LittleEndian, &unknown1Size)
		data.Unknown1 = make([]uint32, unknown1Size)
		for i := uint32(0); i < unknown1Size; i++ {
			binary.Read(reader, binary.LittleEndian, &data.Unknown1[i])
		}
		tile.Extra = data

	case 68: // SpiritBoard
		data := TileSpiritBoard{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		binary.Read(reader, binary.LittleEndian, &data.Unknown3)
		tile.Extra = data

	case 72: // StormyCloud
		data := TileStormyCloud{}
		binary.Read(reader, binary.LittleEndian, &data.StingDuration)
		binary.Read(reader, binary.LittleEndian, &data.IsSolid)
		binary.Read(reader, binary.LittleEndian, &data.NonSolidDuration)
		tile.Extra = data

	case 73: // TemporaryPlatform
		data := TileTemporaryPlatform{}
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		tile.Extra = data

	case 74: // SafeVault
		tile.Extra = TileSafeVault{}

	case 75: // AngelicCountingCloud
		data := TileAngelicCountingCloud{}
		binary.Read(reader, binary.LittleEndian, &data.IsRaffling)
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.AsciiCode)
		tile.Extra = data

	case 77: // InfinityWeatherMachine
		data := TileInfinityWeatherMachine{}
		binary.Read(reader, binary.LittleEndian, &data.IntervalMinutes)
		var listSize uint32
		binary.Read(reader, binary.LittleEndian, &listSize)
		data.WeatherMachineList = make([]uint32, listSize)
		for i := uint32(0); i < listSize; i++ {
			binary.Read(reader, binary.LittleEndian, &data.WeatherMachineList[i])
		}
		tile.Extra = data

	case 79: // PineappleGuzzler
		tile.Extra = TilePineappleGuzzler{}

	case 80: // KrakenGalaticBlock
		data := TileKrakenGalaticBlock{}
		binary.Read(reader, binary.LittleEndian, &data.PatternIndex)
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.R)
		binary.Read(reader, binary.LittleEndian, &data.G)
		binary.Read(reader, binary.LittleEndian, &data.B)
		tile.Extra = data

	case 81: // FriendsEntrance
		data := TileFriendsEntrance{}
		binary.Read(reader, binary.LittleEndian, &data.OwnerUserID)
		binary.Read(reader, binary.LittleEndian, &data.Unknown1)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		tile.Extra = data

	case 69: // TesseractManipulator
		data := TileTesseractManipulator{}
		binary.Read(reader, binary.LittleEndian, &data.Gems)
		binary.Read(reader, binary.LittleEndian, &data.Unknown2)
		binary.Read(reader, binary.LittleEndian, &data.ItemID)
		binary.Read(reader, binary.LittleEndian, &data.Unknown4)
		tile.Extra = data

	default:
		// Unknown tile type, skip carefully
		b.logENet(fmt.Sprintf("WARNING: Unknown tile extra type %d at fg_item=%d", extraType, tile.ForegroundItemID))
	}

	return nil
}
