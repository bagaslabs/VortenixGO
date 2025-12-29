package bot

import (
	"fmt"
	"strings"
	"sync"
	"vortenixgo/network/enet"
)

// BotType defines the authentication method
type BotType string

const (
	BotTypeLegacy BotType = "legacy" // Uses Name & Password
	BotTypeGmail  BotType = "gmail"  // Uses Token / Glog
	BotTypeApple  BotType = "apple"  // Uses Token
)

// HttpsConfig corresponds to Server_Info::Https
type HttpsConfig struct {
	LoginFormURL       string `json:"login_url"`
	FormToken          string `json:"form_token"`
	CookieVisit        string `json:"cookie_visit"`
	CookieActivity     string `json:"cookie_activity"`
	CookieAWSALBTG     string `json:"cookie_aws_albtg"`
	CookieAWSALBTGCORS string `json:"cookie_aws_albtg_cors"`
	CookieAWSALB       string `json:"cookie_aws_alb"`
	CookieAWSALBCORS   string `json:"cookie_aws_alb_cors"`
	CookieXSRF         string `json:"cookie_xsrf"`
	CookieGameSession  string `json:"cookie_game_session"`
	StatusToken        string `json:"status_token"`
	LToken             string `json:"ltoken"`
}

// EnetConfig corresponds to Server_Info::Enet
type EnetConfig struct {
	ServerIP         string `json:"server_ip"`
	ServerPort       int    `json:"server_port"`
	SubServerIP      string `json:"sub_server_ip"`
	SubServerPort    int    `json:"sub_server_port"`
	NowConnectedIP   string `json:"connected_ip"`
	NowConnectedPort int    `json:"connected_port"`
}

// ServerInfo wraps HTTPS and Enet configurations
type ServerInfo struct {
	HTTPS HttpsConfig `json:"https"`
	Enet  EnetConfig  `json:"enet"`
}

// LoginPacket corresponds to the massive Login_Packet struct
type LoginPacket struct {
	TankIDName    string `json:"tank_id_name"`
	TankIDPass    string `json:"tank_id_pass"`
	RequestedName string `json:"requested_name"`
	F             string `json:"f" default:"1"`
	Protocol      string `json:"protocol"`
	GameVersion   string `json:"game_version"`
	Fz            string `json:"fz"`
	Cbits         string `json:"cbits" default:"1024"`
	PlayerAge     string `json:"player_age" default:"17"`
	GDPR          string `json:"gdpr" default:"2"`
	FCMToken      string `json:"fcm_token"`
	Category      string `json:"category" default:"_-5100"`
	TotalPlaytime string `json:"total_playtime" default:"0"`
	Klv           string `json:"klv"`
	Hash2         string `json:"hash2"`
	Meta          string `json:"meta"`
	FHash         string `json:"fhash" default:"-716928004"`
	Rid           string `json:"rid"`
	PlatformID    string `json:"platform_id" default:"0,1,1"`
	DeviceVersion string `json:"device_version" default:"0"`
	Country       string `json:"country" default:"id"`
	Hash          string `json:"hash"`
	Mac           string `json:"mac"`
	Wk            string `json:"wk"`
	Zf            string `json:"zf"`
	LMode         string `json:"lmode" default:"1"`

	// Subserver / Session fields
	User      string `json:"user"`
	UserToken string `json:"user_token"`
	UUIDToken string `json:"uuid_token"`
	DoorID    string `json:"door_id"`
	Aat       string `json:"aat" default:"2"`
	LoginPkt  string `json:"login_packet"` // Raw String if needed
}

// ExternalAuth holds 3rd party auth info
type ExternalAuth struct {
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	AccessKey    string `json:"access_key"`
	UseForGoogle bool   `json:"use_for_google" default:"true"`
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
	Login        LoginPacket  `json:"login_data"`
	Server       ServerInfo   `json:"server_info"`
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

	// Concurrency control
	mu   sync.Mutex
	stop chan struct{}

	// Callbacks
	OnDebug func(category, message string, isError bool) `json:"-"`
}

func (b *Bot) Lock() {
	b.mu.Lock()
}

func (b *Bot) Unlock() {
	b.mu.Unlock()
}

func NewBot(id string, botType BotType, name string, password string, glog string) *Bot {
	bot := &Bot{
		ID:     id,
		Type:   botType,
		Status: "Idle",
		Glog:   glog,
		stop:   make(chan struct{}),
	}

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
	bot.Login = LoginPacket{
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
		bot.Login.RequestedName = "" // As requested, empty for legacy too? Or just Gmail? User said "Requested Name empty... also tank_id_pass empty".
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

	if !b.Connected {
		return
	}

	b.Status = "Idle"
	b.Connected = false
	close(b.stop) // Signal the event loop to stop
	// Re-create the channel for next use, or handle differently
	b.stop = make(chan struct{})
}
