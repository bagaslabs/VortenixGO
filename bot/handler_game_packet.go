package bot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"strings"
	"vortenixgo/database"
)

// SendPacketRaw is now defined in enet_client.go

func (b *Bot) handleGamePacket(ptr []byte) {
	if len(ptr) < 56 {
		return
	}
	var p TankPacketStruct
	binary.Read(bytes.NewReader(ptr), binary.LittleEndian, &p)

	logMsg := "[SYSTEM]: [RECEIVED PACKET RAW]:\n\n"
	logMsg += fmt.Sprintf("  TYPE: %s\n", b.convertTankPacketType(p.Type))
	logMsg += fmt.Sprintf("  OBJECT_TYPE: %d\n", p.ObjectType)
	logMsg += fmt.Sprintf("  NETID: %d\n", p.NetID)
	logMsg += fmt.Sprintf("  TARGET_NETID: %d\n", p.TargetNetID)
	logMsg += fmt.Sprintf("  FLOAT_VARIABLE: %f\n", p.FloatVariable)
	logMsg += fmt.Sprintf("  VALUE: %d\n", p.Value)
	logMsg += fmt.Sprintf("  VECTOR_X: %f\n", p.VectorX)
	logMsg += fmt.Sprintf("  VECTOR_Y: %f\n", p.VectorY)
	logMsg += fmt.Sprintf("  VECTOR_X2: %f\n", p.VectorX2)
	logMsg += fmt.Sprintf("  VECTOR_Y2: %f\n", p.VectorY2)
	logMsg += fmt.Sprintf("  INT_X: %d\n", p.IntX)
	logMsg += fmt.Sprintf("  INT_Y: %d\n", p.IntY)
	logMsg += fmt.Sprintf("  EXT_DATA_SIZE: %d\n", p.ExtendedDataLength)
	b.logENet(logMsg)

	switch ETankPacketType(p.Type) {
	case NET_GAME_PACKET_STATE:
		b.mu.Lock()
		defer b.mu.Unlock()
		for i := 0; i < len(b.Local.Players); i++ {
			player := &b.Local.Players[i]
			if int32(player.NetID) == p.NetID {
				player.PosX = p.VectorX
				player.PosY = p.VectorY
				player.SpeedX = p.VectorX2
				player.SpeedY = p.VectorY2
				if player.NetID == b.Local.NetID {
					b.Local.PosX = p.VectorX
					b.Local.PosY = p.VectorY
					b.Local.SpeedX = p.VectorX2
					b.Local.SpeedY = p.VectorY2
				}

				break
			}
		}

	case NET_GAME_PACKET_CALL_FUNCTION:
		b.logENet("[SYSTEM]: RECEIVED GAME PACKET: " + b.convertTankPacketType(uint8(NET_GAME_PACKET_CALL_FUNCTION)))
		b.handleGamePacket_CallFunction(ptr, &p)
	case NET_GAME_PACKET_SET_CHARACTER_STATE:
		b.logENet("[SYSTEM]: RECEIVED GAME PACKET: " + b.convertTankPacketType(uint8(NET_GAME_PACKET_SET_CHARACTER_STATE)))
		b.mu.Lock()
		b.Local.HackType = int32(p.Value)
		b.Local.BuildLength = int(p.JumpCount) - 126
		b.Local.PunchLength = int(p.AnimationType) - 126
		b.Local.Gravity = p.VectorX2
		b.Local.Velocity = p.VectorY2
		b.mu.Unlock()

	case NET_GAME_PACKET_PING_REQUEST:
		b.mu.Lock()
		b.Status = "online"
		buildLen := b.Local.BuildLength
		punchLen := b.Local.PunchLength
		hackType := b.Local.HackType
		velocity := b.Local.Velocity
		gravity := b.Local.Gravity
		worldName := b.World
		b.mu.Unlock()

		var tank TankPacketStruct
		tank.Type = uint8(NET_GAME_PACKET_PING_REPLY)
		tank.TargetNetID = ProtonHash([]byte(strconv.FormatInt(int64(p.Value), 10)))
		tank.Value = b.GetElapsedMS()

		if buildLen == 0 {
			tank.VectorX = 64.0 // 2.0 * 32.0
		} else {
			tank.VectorX = float32(buildLen) * 32.0
		}

		if punchLen == 0 {
			tank.VectorY = 64.0 // 2.0 * 32.0
		} else {
			tank.VectorY = float32(punchLen) * 32.0
		}

		if worldName != "EXIT" && worldName != "" {
			tank.NetID = int32(hackType)
			tank.VectorX2 = velocity
			tank.VectorY2 = gravity
		}
		b.SendPacketRaw(&tank)
	case NET_GAME_PACKET_SEND_ITEM_DATABASE_DATA:
		// Manual update selected, ignoring automatic data
		b.logENet("[SYSTEM]: Received items.dat packet, but ignoring due to manual update mode.")
	case NET_GAME_PACKET_SEND_INVENTORY_STATE:
		b.handleInventoryState(ptr)
	case NET_GAME_PACKET_MODIFY_ITEM_INVENTORY:
		b.handleModifyInventory(&p)
	case NET_GAME_PACKET_SEND_MAP_DATA:
		b.logENet("[SYSTEM]: Received world map data")
		if p.ExtendedDataLength > 0 && len(ptr) >= 56 {
			mapData := ptr[56:]
			if err := b.ParseWorld(mapData); err != nil {
				b.logENet(fmt.Sprintf("Failed to parse world: %v", err))
			} else {
				b.mu.Lock()
				b.Status = "In World"
				b.mu.Unlock()
				if b.OnUpdate != nil {
					b.OnUpdate()
				}
			}
		}
	case NET_GAME_PACKET_ITEM_CHANGE_OBJECT:
		// UDP::ChangeItemObjects(ptr, bot);
	case NET_GAME_PACKET_TILE_CHANGE_REQUEST:
		// UDP::OnTileChangeRequest(packet, ptr, bot);
	case NET_GAME_PACKET_SEND_TILE_TREE_STATE:
		// UDP::OnTileTreeState(packet, bot);
	case NET_GAME_PACKET_SEND_TILE_UPDATE_DATA:
		// UDP::OnTileUpdateData(packet, ptr + 56, bot);
	case NET_GAME_PACKET_NPC:
		// Silently ignore for now
	case NET_GAME_PACKET_PET_BATTLE:
		// Silently ignore for now
	default:
		log.Printf("Unknown packet type: %s (%d)\n", b.convertTankPacketType(p.Type), p.Type)
	}
}

func (b *Bot) handleGamePacket_CallFunction(ptr []byte, p *TankPacketStruct) {
	if len(ptr) < 56 {
		return
	}
	varList := &VariantList{}
	varList.Parse(ptr[56:])

	b.logENet("[SYSTEM]: RECEIVED VARIANT LIST:\n" + varList.String())

	if len(varList.Variants) > 0 {
		headVar, _ := varList.Variants[0].(string)
		if headVar == "OnSendToServer" {
			b.mu.Lock()
			b.Login.User = varList.GetString(3)
			if varList.GetString(2) != "-1" {
				b.Login.UserToken = varList.GetString(2)
			}
			port := varList.GetInt(1)
			b.Server.Enet.SubServerPort = int(port)
			b.Login.TankIDName = varList.GetString(6)
			b.Name = b.Login.TankIDName
			b.Status = "Redirecting"

			ipPortDoor, _ := varList.Variants[4].(string)
			v := strings.Split(ipPortDoor, "|")
			if len(v) >= 1 {
				b.Server.Enet.SubServerIP = v[0]
			}
			if len(v) >= 2 {
				b.Login.DoorID = v[1]
			}
			if len(v) >= 3 && v[2] != "-1" && v[2] != "0" && v[2] != "" {
				if v[2] != "-1" {
					b.Login.UUIDToken = v[2]
				}
			}
			lmode := varList.GetInt(5)
			b.Login.LMode = fmt.Sprintf("%d", lmode)

			b.Server.Enet.NowConnectedIP = b.Server.Enet.SubServerIP
			b.Server.Enet.NowConnectedPort = b.Server.Enet.SubServerPort
			b.mu.Unlock()

			var tank TankPacketStruct
			tank.Type = uint8(NET_GAME_PACKET_DISCONNECT)
			tank.NetID = 0
			b.SendPacketRaw(&tank)

			generator := &GenerateLoginData{}
			generator.CreateLoginPacket(b)
			go func() {
				b.DisconnectClient()
				b.ConnectClient()
			}()
		} else if headVar == "OnSuperMainStartAcceptLogonHrdxs47254722215a" {
			serverHash := varList.GetUint(1)
			b.mu.Lock()
			b.Local.ServerHash = int(serverHash)
			b.mu.Unlock()

			b.SendPacket("action|enter_game\n", NET_MESSAGE_GENERIC_TEXT)

			var tank TankPacketStruct
			tank.Type = uint8(NET_GAME_PACKET_APP_CHECK_RESPONSE)
			tank.NetID = -1
			tank.Value = 1125432991
			b.SendPacketRaw(&tank)
		} else if headVar == "OnSendFavItemsList" {
			raw := varList.GetString(1) // "242,1796,3722,148,3062,"
			parts := strings.Split(raw, ",")
			favItems := make([]int, 0)
			for _, p := range parts {
				if p == "" {
					continue
				}
				v, err := strconv.Atoi(p)
				if err == nil {
					favItems = append(favItems, v)
				}
			}
			b.mu.Lock()
			b.Local.FavoriteItems = favItems
			b.Local.FavoriteItemsSlot = int(varList.GetInt(2))
			b.mu.Unlock()
		} else if headVar == "SetHasGrowID" {
			b.mu.Lock()
			b.Local.Name = varList.GetString(2)
			b.Local.Password = varList.GetString(3)
			b.mu.Unlock()
		} else if headVar == "OnSpawn" {
			raw := varList.GetString(1)
			lines := strings.Split(raw, "\n")
			p := Players{}
			for _, line := range lines {
				parts := strings.Split(line, "|")
				if len(parts) < 2 {
					continue
				}
				key := parts[0]
				switch key {
				case "spawn":
					p.Spawn = parts[1]
				case "netID":
					p.NetID, _ = strconv.Atoi(parts[1])
				case "userID":
					p.UserID, _ = strconv.Atoi(parts[1])
				case "name":
					p.Name = strings.ReplaceAll(parts[1], "`", "")
				case "country":
					p.Country = parts[1]
				case "eid":
					p.EID = parts[1]
				case "ip":
					p.IP = parts[1]
				case "invis":
					v, _ := strconv.Atoi(parts[1])
					p.Invisible = uint8(v)
				case "mstate":
					v, _ := strconv.Atoi(parts[1])
					p.Mstate = uint8(v)
				case "smstate":
					v, _ := strconv.Atoi(parts[1])
					p.Smstate = uint8(v)
				case "onlineID":
					p.OnlineId = parts[1]
				case "posXY":
					if len(parts) >= 3 {
						x, _ := strconv.ParseFloat(parts[1], 32)
						y, _ := strconv.ParseFloat(parts[2], 32)
						p.PosX = float32(x)
						p.PosY = float32(y)
					}
				case "type":
					p.IsLocal = parts[1] == "local"
				}
			}
			p.Mod = false
			//How to check mod? player.m_state == 1 || player.invisible != 0
			if p.Mstate == 1 || p.Invisible != 0 {
				b.logENet("[SYSTEM]: Player " + p.Name + " is modded.")
				p.Mod = true
			}
			b.mu.Lock()
			b.Local.Players = append(b.Local.Players, p)
			if p.IsLocal {
				b.Local.NetID = p.NetID
				b.Local.UserID = p.UserID
				b.Local.Name = p.Name
				b.Local.PosX = p.PosX
				b.Local.PosY = p.PosY
			}
			b.mu.Unlock()
		} else if headVar == "OnRemove" {
			raw, _ := varList.Variants[1].(string)

			// ambil netID
			netIDRemove := -1
			lines := strings.Split(raw, "\n")

			for _, line := range lines {
				parts := strings.Split(line, "|")
				if len(parts) >= 2 && parts[0] == "netID" {
					netIDRemove, _ = strconv.Atoi(parts[1])
					break
				}
			}
			if netIDRemove == -1 {
				return
			}
			b.mu.Lock()
			defer b.mu.Unlock()

			for i := 0; i < len(b.Local.Players); i++ {
				if b.Local.Players[i].NetID == netIDRemove {

					// hapus player
					b.Local.Players = append(
						b.Local.Players[:i],
						b.Local.Players[i+1:]...,
					)
					// kalau player lokal
					if netIDRemove == b.Local.NetID {
						b.Local.NetID = -1
					}
					break
				}
			}
		} else if headVar == "OnSetPos" {
			v := varList.GetVector2(1)
			b.mu.Lock()
			defer b.mu.Unlock()
			for i := 0; i < len(b.Local.Players); i++ {
				player := &b.Local.Players[i]
				if int32(player.NetID) == p.NetID {
					player.PosX = v.X
					player.PosY = v.Y
					if player.NetID == b.Local.NetID {
						b.Local.PosX = v.X
						b.Local.PosY = v.Y
					}
					break
				}
			}
		} else if headVar == "OnSetBux" {
			switch v := varList.Variants[1].(type) {
			case int32:
				b.Local.GemCount = int(v)
			case uint32:
				b.Local.GemCount = int(v)
			case float32:
				b.Local.GemCount = int(v)
			}
		} else if headVar == "OnSetClothing" {
			b.mu.Lock()
			defer b.mu.Unlock()
			// Try to find player in list to verify if it's local
			var targetPlayer *Players
			for i := 0; i < len(b.Local.Players); i++ {
				if int32(b.Local.Players[i].NetID) == p.NetID {
					targetPlayer = &b.Local.Players[i]
					break
				}
			}

			// If we know this is us (either by NetID match or we found our player struct and it says IsLocal)
			isLocal := false
			if targetPlayer != nil {
				if targetPlayer.IsLocal {
					isLocal = true
					// Ensure b.Local.NetID is synced if it wasn't
					if b.Local.NetID == -1 || b.Local.NetID == 0 {
						b.Local.NetID = targetPlayer.NetID
					}
				} else if p.NetID == int32(b.Local.NetID) {
					isLocal = true
				}
			} else if p.NetID == int32(b.Local.NetID) {
				isLocal = true
			}

			b.logENet(fmt.Sprintf("[DEBUG] OnSetClothing for NetID %d. IsLocal: %v", p.NetID, isLocal))

			// Update active items list
			newActiveItems := []int{}

			// Parse variants
			for j := 1; j <= 5; j++ {
				if len(varList.Variants) > j {
					v3, ok := varList.Variants[j].(Vector3)
					if ok {
						for _, val := range []float32{v3.X, v3.Y, v3.Z} {
							if val != 0 {
								newActiveItems = append(newActiveItems, int(val))
							}
						}
					}
				}
			}

			// Apply to player struct if found
			if targetPlayer != nil {
				targetPlayer.ActiveItems = newActiveItems
			}

			// Apply to local bot if local
			if isLocal {
				b.Local.ActiveItems = newActiveItems
				// Sync Inventory IsActive states
				for k := 0; k < len(b.Local.Inventory); k++ {
					// Reset first
					b.Local.Inventory[k].IsActive = false
					// Check if active
					for _, activeID := range newActiveItems {
						if int(b.Local.Inventory[k].ID) == activeID {
							b.Local.Inventory[k].IsActive = true
							break
						}
					}
				}
			}
		}
		if b.OnUpdate != nil {
			b.OnUpdate()
		}
	}
}

func (b *Bot) handleInventoryState(ptr []byte) {
	if len(ptr) < 56+5+1 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.Local.Inventory = nil

	offset := 56 + 5 // offset = 61

	// Match C++: uint8_t size = *(ptr);
	// Offset is at 61 (56 + 5)
	size := int(ptr[offset])
	// C++ loop uses ptr += 2 immediately, so we keep offset at 'size' position initially?
	// C++: ptr points to size.
	// Loop: ptr += 2 (skips size and next byte). item.id = ...
	// So we DO NOT increment offset here if we want to mimic the relative jumps inside loop starting from 'size' pos.
	// But to be cleaner in Go, let's track absolute:
	// Ptr starts at 61.

	// We allow the loop to handle the increments relative to 'current' pointer.
	// So we keep offset = 61.

	for i := 0; i < size; i++ {
		// ptr += 2 (skips size byte and next byte)
		offset += 2
		if offset+2 > len(ptr) { // Need 2 bytes for itemID
			break
		}

		// item.id = *(short*)(ptr)
		itemID := int16(binary.LittleEndian.Uint16(ptr[offset : offset+2]))

		// ptr += 2 (skips itemID)
		offset += 2
		if offset >= len(ptr) { // Need 1 byte for count
			break
		}

		// item.count = *(ptr)
		count := ptr[offset]

		var itemName string
		db := database.GetGlobalItemDB()
		if db != nil {
			itemDef := db.GetItem(uint32(itemID))
			if itemDef != nil {
				itemName = itemDef.Name
			}
		}
		if itemName == "" {
			itemName = fmt.Sprintf("Item %d", itemID)
		}

		item := Inventory{
			Name:  itemName,
			ID:    itemID,
			Count: count,
		}
		// Check if favorite
		for _, favID := range b.Local.FavoriteItems {
			if int(item.ID) == favID {
				item.IsFavorite = true
				break
			}
		}
		for _, activeID := range b.Local.ActiveItems {
			if int(item.ID) == activeID {
				item.IsActive = true
				break
			}
		}
		b.Local.Inventory = append(b.Local.Inventory, item)
	}

	b.Local.InventorySlots = size
}

func (b *Bot) handleModifyInventory(p *TankPacketStruct) {
	b.mu.Lock()
	defer b.mu.Unlock()

	itemID := int16(p.Value)
	amountToRemove := int(p.JumpCount)

	for i := 0; i < len(b.Local.Inventory); i++ {
		if b.Local.Inventory[i].ID == itemID {
			if int(b.Local.Inventory[i].Count) <= amountToRemove {
				// Remove item
				b.Local.Inventory = append(b.Local.Inventory[:i], b.Local.Inventory[i+1:]...)
			} else {
				// Decrease count
				b.Local.Inventory[i].Count -= uint8(amountToRemove)
			}
			break
		}
	}
	b.Local.InventorySlots = len(b.Local.Inventory)
}
