package bot

import (
	"fmt"
)

// WearItem sends a packet to wear an item from inventory
func (b *Bot) WearItem(itemID int32) {
	var tank TankPacketStruct
	tank.Type = uint8(NET_GAME_PACKET_ITEM_ACTIVATE_REQUEST)
	tank.Value = uint32(itemID)
	tank.NetID = -1
	b.SendPacketRaw(&tank)
	b.logENet(fmt.Sprintf("[SYSTEM]: [WEAR ITEM]: ID %d", itemID))
}

// UnwearItem is usually the same as Wear (toggle) in Growtopia
func (b *Bot) UnwearItem(itemID int32) {
	b.WearItem(itemID)
}

// DropItem initiates a drop for a specific item
func (b *Bot) DropItem(itemID int32) {
	// In Growtopia, dropping typically requires opening the dialog first or sending a specific action
	// For simplicity, we'll send the action packet that most servers expect
	b.SendPacket(fmt.Sprintf("action|drop\nitemID|%d", itemID), NET_MESSAGE_GENERIC_TEXT)
	b.logENet(fmt.Sprintf("[SYSTEM]: [DROP ITEM]: ID %d", itemID))
}

// TrashItem initiates a trash for a specific item
func (b *Bot) TrashItem(itemID int32) {
	b.SendPacket(fmt.Sprintf("action|trash\nitemID|%d", itemID), NET_MESSAGE_GENERIC_TEXT)
	b.logENet(fmt.Sprintf("[SYSTEM]: [TRASH ITEM]: ID %d", itemID))
}
