package bot

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"vortenixgo/database"
	"vortenixgo/network/enet"
)

// BotType defines the authentication method
type BotType string

const (
	BotTypeLegacy BotType = "legacy" // Uses Name & Password
	BotTypeGmail  BotType = "gmail"  // Uses Token / Glog
	BotTypeApple  BotType = "apple"  // Uses Token
)

// ExternalAuth holds 3rd party auth info
type ExternalAuth struct {
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	AccessKey    string `json:"access_key"`
	UseForGoogle bool   `json:"use_for_google" default:"true"`
}

var (
	GlobalItemDatabase *database.ItemDatabase
	GlobalItemsDatHash uint32
	itemDatabaseOnce   sync.Once
)

func init() {
	itemDatabaseOnce.Do(func() {
		GlobalItemDatabase = database.NewItemDatabase()
	})
}

type QueuedPacket struct {
	Data  []byte
	Delay time.Duration
}

// Bot represents a single bot instance
type Bot struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"` // Display name (from TankIDName or RequestedName)
	Type   BotType `json:"type"`
	Status string  `json:"status"` // Idle, Connecting, Online, etc.

	// Stats
	Level    int    `json:"level"`
	World    string `json:"world"`
	PlayTime string `json:"play_time"`
	Age      int    `json:"age"` // In days

	// Authentication & Connection Data
	Login        *LoginPacket `json:"-"`
	Server       *ServerInfo  `json:"-"`
	ExternalAuth ExternalAuth `json:"external_auth"`

	// State
	Connected        bool   `json:"connected"`
	UseBypassProxy   bool   `json:"use_bypass_proxy"`
	Glog             string `json:"glog"`              // Optional glog / format
	Email            string `json:"email"`             // 3rd party email
	ExternalPassword string `json:"external_password"` // Password for external auth (Gmail/Apple)
	InGameName       string `json:"ingame_name"`       // Name obtained from game server
	DisplayName      string `json:"display_name"`      // Computed display name (Email or InGameName)
	Proxy            string `json:"proxy"`             // Socks5 Proxy (ip:port:user:pass)

	// Enet Client
	Client *enet.Host
	Peer   *enet.Peer

	Ping     int  `json:"ping"`
	ShowENet bool `json:"show_enet"`

	Local     Local     `json:"local"`
	CreatedAt time.Time `json:"created_at"`

	LastPacketReceivedAt time.Time  `json:"-"`
	Ping500StartedAt     *time.Time `json:"-"`

	ItemDatabase *database.ItemDatabase `json:"-"`

	// Concurrency control
	mu           sync.Mutex
	stop         chan struct{}
	enetLoopDone chan struct{} // Signals that EventListener has exited
	packetQueue  chan QueuedPacket

	// Callbacks
	OnDebug  func(category, message string, isError bool) `json:"-"`
	OnUpdate func()                                       `json:"-"`
}

func (b *Bot) Lock() {
	b.mu.Lock()
}

func (b *Bot) Unlock() {
	b.mu.Unlock()
}

func (b *Bot) GetElapsedMS() uint32 {
	return uint32(time.Since(b.CreatedAt).Milliseconds())
}

func (b *Bot) ResetEnetData() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 1. Clear Network Config & Tokens
	b.Server.Enet = EnetConfig{} // Resets all fields to zero-value (empty strings, 0 ints)
	b.Login.User = ""
	b.Login.UserToken = ""
	b.Login.UUIDToken = ""
	b.Login.DoorID = ""

	// 2. Clear Local Player Attributes
	b.Local.Country = ""
	b.Local.UserID = 0
	b.Local.NetID = -1 // -1 usually indicates no ID
	b.Local.EID = ""
	b.Local.IP = ""

	// 3. Clear Position & Movement
	b.Local.PosX = 0
	b.Local.PosY = 0
	b.Local.SpeedX = 0
	b.Local.SpeedY = 0

	// 4. Clear Items & Inventory
	b.Local.ActiveItems = []int{}
	b.Local.Inventory = []Inventory{}
	b.Local.InventorySlots = 0

	// 5. Clear Currencies & Stats
	b.Local.GemCount = 0
	b.Local.PearlCount = 0
	b.Local.VoucherCount = 0
	b.Local.Awesomeness = 0
	b.Local.GlobalPlaytime = 0
	b.Local.WorldLock = 0
	b.Local.TotalPlaytime = 0
	b.Local.FavoriteItems = []int{}
	b.Local.FavoriteItemsSlot = 0

	// 6. Clear World & Others
	b.Local.World = World{}
	b.Local.Players = []Players{} // Clear player list

	b.World = ""
	b.Connected = false
	b.Status = "Idle"
}

func NewBot(id string, botType BotType, name string, password string, glog string) *Bot {
	bot := &Bot{
		ID:           id,
		Type:         botType,
		Status:       "Idle",
		Glog:         glog,
		stop:         make(chan struct{}),
		enetLoopDone: make(chan struct{}),
		packetQueue:  make(chan QueuedPacket, 100),
		CreatedAt:    time.Now(),
		ItemDatabase: GlobalItemDatabase,
	}
	close(bot.enetLoopDone) // Start as "dead" so ConnectClient knows to start listener

	// Parse Glog string if provided: ip:port:accessKey
	if glog != "" {
		parts := strings.Split(glog, ":")
		if len(parts) >= 3 {
			bot.ExternalAuth.IP = parts[0]
			fmt.Sscanf(parts[1], "%d", &bot.ExternalAuth.Port)
			bot.ExternalAuth.AccessKey = parts[2]
			bot.ExternalAuth.UseForGoogle = true // Default to true if glog is provided
		}
	}

	// Initialize LoginPacket with defaults
	bot.Login = &bot.Local.Login
	bot.Server = &bot.Local.ServerInfo

	*bot.Login = LoginPacket{
		F:             "1",
		Protocol:      "225",
		GameVersion:   "5.39",
		Fz:            "22647480",
		Cbits:         "1024",
		PlayerAge:     "17",
		GDPR:          "1",
		Category:      "_-5100",
		TotalPlaytime: "0",
		FHash:         "-716928004",
		PlatformID:    "0,1,1",
		DeviceVersion: "0",
		Country:       "us",
		Zf:            "-1986240656",
		LMode:         "1",
		Aat:           "2",
	}

	if (botType == BotTypeGmail || botType == BotTypeApple) && contains(password, "|") {
		// New Format: email|token|mac|rid|wk
		parts := split(password, "|")

		if len(parts) >= 1 {
			bot.Email = parts[0]
			bot.Name = parts[0]
			bot.DisplayName = parts[0] // Initial display is email
		}

		if len(parts) >= 2 {
			bot.Server.HTTPS.LToken = parts[1]
		}
		if len(parts) >= 3 {
			bot.Login.Mac = parts[2]
		}
		if len(parts) >= 4 {
			bot.Login.Rid = parts[3]
		}
		if len(parts) >= 5 {
			bot.Login.Wk = parts[4]
		}

		if len(parts) >= 6 {
			bot.ExternalPassword = parts[5]
		}

		// Ensure requestedName and password fields in Login packet are empty for token-based bots
		bot.Login.RequestedName = ""
		bot.Login.TankIDPass = "" // Password is used for external auth request but not in the game packet

	} else {
		// Legacy / Name & Pass
		bot.Name = name
		bot.InGameName = name
		bot.DisplayName = name
		bot.Login.TankIDName = name
		bot.Login.TankIDPass = password
		bot.Login.RequestedName = ""
		// Wait, if it's Legacy, we NEED tank_id_pass to login usually.
		// User said "Saya ingin requested Name nya empty... begitu juga dengan tank_id_pass empty."
		// If this applies to ALL bots, then legacy won't work without pass.
		// Assuming user means for the current context (Gmail/Apple Glog flow) or maybe Guest.
		// But let's follow instruction literally for RequestedName, but be careful with Pass for Legacy.

		// If user truly wants TankIDPass empty for ALL, legacy login will fail unless it's a guest account.
		// However, in "getTokenLegacy" we use b.Login.TankIDPass. If we clear it here, getTokenLegacy fails.
		// I will assume the user refers to the Login Packet sent to CLIENT, or specifically for the Gmail/Apple flow.
		// Let's clear RequestedName for all as it's often optional.
		// For TankIDPass, I will clear it only if it's NOT Legacy, OR if the user meant the packet sent to server?

		// User instruction: "Saya ingin requested Name nya empty tidak usah diisi apapun begitu juga dengan tank_id_pass empty."
		// Context: Discussion about Glog/Token flow.
		// Action: Set RequestedName = "" for all. Set TankIDPass = "" for Gmail/Apple.
		// For Legacy, if I set TankIDPass="", how do we login?
		// Maybe the user implies the 'LoginPacket' struct used for `CreateLoginPacket`, not the internal storage.
		// But CreateLoginPacket reads from b.Login.

		// Safe bet: Empty RequestedName for all.
		// Empty TankIDPass for Gmail/Apple (since pass is in main Bot struct/ExternalAuth payload, not needed in ENet packet).
	}

	return bot
}

func contains(s, substr string) bool {
	// Basic helper since we want to avoid too many imports if possible, but strings is usually fine
	return strings.Contains(s, substr)
}

func split(s, sep string) []string {
	return strings.Split(s, sep)
}

func (b *Bot) Connect() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Connected {
		return
	}

	b.Status = "Connecting..."
	b.Connected = true
}

func (b *Bot) Disconnect() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Status = "Idle"
	b.Connected = false

	// Safely close stop channel
	defer func() {
		if r := recover(); r != nil {
			// Ignore if already closed
		}
	}()
	close(b.stop) // Signal the event loop to stop
	// Re-create the channel for next use, or handle differently
	b.stop = make(chan struct{})
}

func (b *Bot) Say(text string) {
	pkt := fmt.Sprintf("action|input\n|text|%s\n", text)
	b.SendPacketWithDelay(pkt, NET_MESSAGE_GENERIC_TEXT, 2000*time.Millisecond)
}

func (b *Bot) Warp(world string) {
	pkt := fmt.Sprintf("action|join_request\nname|%s\ninvitedWorld|0\n", world)
	b.SendPacketWithDelay(pkt, NET_MESSAGE_GAME_MESSAGE, 4000*time.Millisecond)
}
