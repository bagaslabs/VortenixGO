package bot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// Network Message Types
const (
	NET_MESSAGE_UNKNOWN             uint32 = 0
	NET_MESSAGE_SERVER_HELLO        uint32 = 1
	NET_MESSAGE_GENERIC_TEXT        uint32 = 2
	NET_MESSAGE_GAME_MESSAGE        uint32 = 3
	NET_MESSAGE_GAME_PACKET         uint32 = 4
	NET_MESSAGE_ERROR               uint32 = 5
	NET_MESSAGE_TRACK               uint32 = 6
	NET_MESSAGE_CLIENT_LOG_REQUEST  uint32 = 7
	NET_MESSAGE_CLIENT_LOG_RESPONSE uint32 = 8
	NET_MESSAGE_MAX                 uint32 = 9
)

type ETankPacketType uint8

const (
	NET_GAME_PACKET_STATE                            ETankPacketType = 0
	NET_GAME_PACKET_CALL_FUNCTION                    ETankPacketType = 1
	NET_GAME_PACKET_UPDATE_STATUS                    ETankPacketType = 2
	NET_GAME_PACKET_TILE_CHANGE_REQUEST              ETankPacketType = 3
	NET_GAME_PACKET_SEND_MAP_DATA                    ETankPacketType = 4
	NET_GAME_PACKET_SEND_TILE_UPDATE_DATA            ETankPacketType = 5
	NET_GAME_PACKET_SEND_TILE_UPDATE_DATA_MULTIPLE   ETankPacketType = 6
	NET_GAME_PACKET_TILE_ACTIVATE_REQUEST            ETankPacketType = 7
	NET_GAME_PACKET_TILE_APPLY_DAMAGE                ETankPacketType = 8
	NET_GAME_PACKET_SEND_INVENTORY_STATE             ETankPacketType = 9
	NET_GAME_PACKET_ITEM_ACTIVATE_REQUEST            ETankPacketType = 10
	NET_GAME_PACKET_ITEM_ACTIVATE_OBJECT_REQUEST     ETankPacketType = 11
	NET_GAME_PACKET_SEND_TILE_TREE_STATE             ETankPacketType = 12
	NET_GAME_PACKET_MODIFY_ITEM_INVENTORY            ETankPacketType = 13
	NET_GAME_PACKET_ITEM_CHANGE_OBJECT               ETankPacketType = 14
	NET_GAME_PACKET_SEND_LOCK                        ETankPacketType = 15
	NET_GAME_PACKET_SEND_ITEM_DATABASE_DATA          ETankPacketType = 16
	NET_GAME_PACKET_SEND_PARTICLE_EFFECT             ETankPacketType = 17
	NET_GAME_PACKET_SET_ICON_STATE                   ETankPacketType = 18
	NET_GAME_PACKET_ITEM_EFFECT                      ETankPacketType = 19
	NET_GAME_PACKET_SET_CHARACTER_STATE              ETankPacketType = 20
	NET_GAME_PACKET_PING_REPLY                       ETankPacketType = 21
	NET_GAME_PACKET_PING_REQUEST                     ETankPacketType = 22
	NET_GAME_PACKET_GOT_PUNCHED                      ETankPacketType = 23
	NET_GAME_PACKET_APP_CHECK_RESPONSE               ETankPacketType = 24
	NET_GAME_PACKET_APP_INTEGRITY_FAIL               ETankPacketType = 25
	NET_GAME_PACKET_DISCONNECT                       ETankPacketType = 26
	NET_GAME_PACKET_BATTLE_JOIN                      ETankPacketType = 27
	NET_GAME_PACKET_BATTLE_EVENT                     ETankPacketType = 28
	NET_GAME_PACKET_USE_DOOR                         ETankPacketType = 29
	NET_GAME_PACKET_SEND_PARENTAL                    ETankPacketType = 30
	NET_GAME_PACKET_GONE_FISHIN                      ETankPacketType = 31
	NET_GAME_PACKET_STEAM                            ETankPacketType = 32
	NET_GAME_PACKET_PET_BATTLE                       ETankPacketType = 33
	NET_GAME_PACKET_NPC                              ETankPacketType = 34
	NET_GAME_PACKET_SPECIAL                          ETankPacketType = 35
	NET_GAME_PACKET_SEND_PARTICLE_EFFECT_V2          ETankPacketType = 36
	NET_GAME_PACKET_ACTIVATE_ARROW_TO_ITEM           ETankPacketType = 37
	NET_GAME_PACKET_SELECT_TILE_INDEX                ETankPacketType = 38
	NET_GAME_PACKET_SEND_PLAYER_TRIBUTE_DATA         ETankPacketType = 39
	NET_GAME_PACKET_FTUE_SET_ITEM_TO_QUICK_INVENTORY ETankPacketType = 40
	NET_GAME_PACKET_PVE_NPC                          ETankPacketType = 41
	NET_GAME_PACKET_PVP_CARD_BATTLE                  ETankPacketType = 42
	NET_GAME_PACKET_PVE_APPLY_PLAYER_DAMAGE          ETankPacketType = 43
	NET_GAME_PACKET_PVE_NPC_POSITION_UPDATE          ETankPacketType = 44
	NET_GAME_PACKET_SET_EXTRA_MODS                   ETankPacketType = 45
	NET_GAME_PACKET_ON_STEP_TILE_MOD                 ETankPacketType = 46
)

type TankPacketStruct struct {
	Type               uint8   // 0
	ObjectType         uint8   // 1
	JumpCount          uint8   // 2
	AnimationType      uint8   // 3
	NetID              int32   // 4
	TargetNetID        int32   // 8
	Flags              uint32  // 12
	FloatVariable      float32 // 16
	Value              uint32  // 20
	VectorX            float32 // 24
	VectorY            float32 // 28
	VectorX2           float32 // 32
	VectorY2           float32 // 36
	ParticleRotation   float32 // 40
	IntX               int32   // 44
	IntY               int32   // 48
	ExtendedDataLength uint32  // 52
}

// Serialize converts the struct to a byte slice
func (p *TankPacketStruct) Serialize() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, p)
	return buf.Bytes()
}

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

type VariantList struct {
	Variants []interface{}
}

func (vl *VariantList) Parse(data []byte) {
	if len(data) < 1 {
		return
	}
	count := int(data[0])
	offset := 1
	for i := 0; i < count; i++ {
		if offset+2 > len(data) {
			break
		}
		// index := data[offset]
		vType := data[offset+1]
		offset += 2

		switch vType {
		case 1: // Float
			if offset+4 <= len(data) {
				bits := binary.LittleEndian.Uint32(data[offset : offset+4])
				vl.Variants = append(vl.Variants, math.Float32frombits(bits))
				offset += 4
			}
		case 3: // Vector2
			if offset+8 <= len(data) {
				xBits := binary.LittleEndian.Uint32(data[offset : offset+4])
				yBits := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
				vl.Variants = append(vl.Variants, Vector2{
					X: math.Float32frombits(xBits),
					Y: math.Float32frombits(yBits),
				})
				offset += 8
			}
		case 4: // Vector3
			if offset+12 <= len(data) {
				xBits := binary.LittleEndian.Uint32(data[offset : offset+4])
				yBits := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
				zBits := binary.LittleEndian.Uint32(data[offset+8 : offset+12])
				vl.Variants = append(vl.Variants, Vector3{
					X: math.Float32frombits(xBits),
					Y: math.Float32frombits(yBits),
					Z: math.Float32frombits(zBits),
				})
				offset += 12
			}
		case 2: // String
			if offset+4 <= len(data) {
				strLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
				offset += 4
				if offset+strLen <= len(data) {
					vl.Variants = append(vl.Variants, string(data[offset:offset+strLen]))
					offset += strLen
				}
			}
		case 5: // uint32
			if offset+4 <= len(data) {
				vl.Variants = append(vl.Variants, binary.LittleEndian.Uint32(data[offset:offset+4]))
				offset += 4
			}
		case 9: // int32
			if offset+4 <= len(data) {
				vl.Variants = append(vl.Variants, int32(binary.LittleEndian.Uint32(data[offset:offset+4])))
				offset += 4
			}
		default:
			// Unhandled type
			vl.Variants = append(vl.Variants, nil)
		}
	}
}

func (vl *VariantList) String() string {
	var sb strings.Builder
	for i, v := range vl.Variants {
		sb.WriteString(fmt.Sprintf("[%d] %v\n", i, v))
	}
	return sb.String()
}

func (vl *VariantList) GetInt(index int) int32 {
	if index < 0 || index >= len(vl.Variants) || vl.Variants[index] == nil {
		return 0
	}
	switch v := vl.Variants[index].(type) {
	case int32:
		return v
	case uint32:
		return int32(v)
	case float32:
		return int32(v)
	}
	return 0
}

func (vl *VariantList) GetUint(index int) uint32 {
	if index < 0 || index >= len(vl.Variants) || vl.Variants[index] == nil {
		return 0
	}
	switch v := vl.Variants[index].(type) {
	case uint32:
		return v
	case int32:
		return uint32(v)
	case float32:
		return uint32(v)
	}
	return 0
}

func (vl *VariantList) GetString(index int) string {
	if index < 0 || index >= len(vl.Variants) || vl.Variants[index] == nil {
		return ""
	}
	if s, ok := vl.Variants[index].(string); ok {
		return s
	}
	return fmt.Sprintf("%v", vl.Variants[index])
}

func (vl *VariantList) GetVector2(index int) Vector2 {
	if index < 0 || index >= len(vl.Variants) || vl.Variants[index] == nil {
		return Vector2{}
	}
	if v, ok := vl.Variants[index].(Vector2); ok {
		return v
	}
	return Vector2{}
}

func (vl *VariantList) GetVector3(index int) Vector3 {
	if index < 0 || index >= len(vl.Variants) || vl.Variants[index] == nil {
		return Vector3{}
	}
	if v, ok := vl.Variants[index].(Vector3); ok {
		return v
	}
	return Vector3{}
}

func ProtonHash(data []byte) int32 {
	var hash uint32 = 0x55555555
	for _, b := range data {
		hash = (hash >> 27) + (hash << 5) + uint32(b)
	}
	return int32(hash)
}

func (b *Bot) convertTankPacketType(t uint8) string {
	switch ETankPacketType(t) {
	case NET_GAME_PACKET_STATE:
		return "STATE"
	case NET_GAME_PACKET_CALL_FUNCTION:
		return "CALL_FUNCTION"
	case NET_GAME_PACKET_UPDATE_STATUS:
		return "UPDATE_STATUS"
	case NET_GAME_PACKET_TILE_CHANGE_REQUEST:
		return "TILE_CHANGE_REQUEST"
	case NET_GAME_PACKET_SEND_MAP_DATA:
		return "SEND_MAP_DATA"
	case NET_GAME_PACKET_PING_REQUEST:
		return "PING_REQUEST"
	case NET_GAME_PACKET_PING_REPLY:
		return "PING_REPLY"
	case NET_GAME_PACKET_DISCONNECT:
		return "DISCONNECT"
	default:
		return fmt.Sprintf("UNKNOWN (%d)", t)
	}
}

type Vector2 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type Players struct {
	Spawn       string  `json:"spawn"`
	Name        string  `json:"name"`
	Country     string  `json:"country"`
	UserID      int     `json:"userid"`
	NetID       int     `json:"netid"`
	EID         string  `json:"eid"`
	IP          string  `json:"ip"`
	PosX        float32 `json:"pos_x"`
	PosY        float32 `json:"pos_y"`
	SpeedX      float32 `json:"speed_x"`
	SpeedY      float32 `json:"speed_y"`
	Invisible   uint8   `json:"invisible"`
	Mstate      uint8   `json:"mstate"`
	Smstate     uint8   `json:"smstate"`
	OnlineId    string  `json:"online_id"`
	IsLocal     bool    `json:"is_local"`
	ActiveItems []int   `json:"active_items"`
	Mod         bool    `json:"mod"`
}

type Inventory struct {
	Name       string `json:"name"`
	ID         int16  `json:"id"`
	Count      uint8  `json:"count"`
	IsFavorite bool   `json:"is_favorite"`
	IsActive   bool   `json:"is_active"`
}

// WeatherType constants
type WeatherType uint16

const (
	WeatherDefault                  WeatherType = 0
	WeatherSunset                   WeatherType = 1
	WeatherNight                    WeatherType = 2
	WeatherDesert                   WeatherType = 3
	WeatherSunny                    WeatherType = 4
	WeatherRainyCity                WeatherType = 5
	WeatherHarvest                  WeatherType = 6
	WeatherMars                     WeatherType = 7
	WeatherSpooky                   WeatherType = 8
	WeatherMaw                      WeatherType = 9
	WeatherBlank                    WeatherType = 10
	WeatherSnowy                    WeatherType = 11
	WeatherGrowch                   WeatherType = 12
	WeatherGrowchHappy              WeatherType = 13
	WeatherUndersea                 WeatherType = 14
	WeatherWarp                     WeatherType = 15
	WeatherComet                    WeatherType = 16
	WeatherComet2                   WeatherType = 17
	WeatherParty                    WeatherType = 18
	WeatherPineapple                WeatherType = 19
	WeatherSnowyNight               WeatherType = 20
	WeatherSpring                   WeatherType = 21
	WeatherWolf                     WeatherType = 22
	WeatherNotInitialized           WeatherType = 23
	WeatherPurpleHaze               WeatherType = 24
	WeatherFireHaze                 WeatherType = 25
	WeatherGreenHaze                WeatherType = 26
	WeatherAquaHaze                 WeatherType = 27
	WeatherCustomHaze               WeatherType = 28
	WeatherCustomItems              WeatherType = 29
	WeatherPagoda                   WeatherType = 30
	WeatherApocalypse               WeatherType = 31
	WeatherJungle                   WeatherType = 32
	WeatherBalloonWarz              WeatherType = 33
	WeatherBackground               WeatherType = 34
	WeatherAutumn                   WeatherType = 35
	WeatherHearth                   WeatherType = 36
	WeatherStPatricks               WeatherType = 37
	WeatherIceAge                   WeatherType = 38
	WeatherVolcano                  WeatherType = 39
	WeatherFloatingIslands          WeatherType = 40
	WeatherMascot                   WeatherType = 41
	WeatherDigitalRain              WeatherType = 42
	WeatherMonoChrome               WeatherType = 43
	WeatherTreasure                 WeatherType = 44
	WeatherSurgery                  WeatherType = 45
	WeatherBountiful                WeatherType = 46
	WeatherMeteor                   WeatherType = 47
	WeatherStars                    WeatherType = 48
	WeatherAscended                 WeatherType = 49
	WeatherDestroyed                WeatherType = 50
	WeatherGrowtopiaSign            WeatherType = 51
	WeatherDungeon                  WeatherType = 52
	WeatherLegendaryCity            WeatherType = 53
	WeatherBloodDragon              WeatherType = 54
	WeatherPopCity                  WeatherType = 55
	WeatherAnzu                     WeatherType = 56
	WeatherTmntCity                 WeatherType = 57
	WeatherRadCity                  WeatherType = 58
	WeatherPlaze                    WeatherType = 59
	WeatherNebula                   WeatherType = 60
	WeatherProtoStar                WeatherType = 61
	WeatherDarkMountains            WeatherType = 62
	WeatherAc15                     WeatherType = 63
	WeatherMountGrowMore            WeatherType = 64
	WeatherCrackInReality           WeatherType = 65
	WeatherLnyNian                  WeatherType = 66
	WeatherRaymanLock               WeatherType = 67
	WeatherSteampunk                WeatherType = 68
	WeatherRealmOfSpirits           WeatherType = 69
	WeatherBlackhole                WeatherType = 70
	WeatherGems                     WeatherType = 71
	WeatherHolidayHaven             WeatherType = 72
	WeatherFenyxLock                WeatherType = 73
	WeatherEnchantedLock            WeatherType = 74
	WeatherRoyalEnchantedLock       WeatherType = 75
	WeatherNeptunesAtlantis         WeatherType = 76
	WeatherPinuskiPetalPerfectHaven WeatherType = 77
	WeatherCandyland                WeatherType = 78
)

// World represents the game world/map
type World struct {
	Name           string        `json:"name"`
	Width          uint32        `json:"width"`
	Height         uint32        `json:"height"`
	TileCount      uint32        `json:"tile_count"`
	Tiles          []Tile        `json:"tiles"`
	DroppedItems   []DroppedItem `json:"dropped_items"`
	BaseWeather    WeatherType   `json:"base_weather"`
	CurrentWeather WeatherType   `json:"current_weather"`
	Version        uint16        `json:"version"`
	Flags          uint32        `json:"flags"`
	CollisionMap   []uint8       `json:"-"`
}

// Tile represents a single tile in the world
type Tile struct {
	ForegroundItemID uint16      `json:"fg_id"`
	BackgroundItemID uint16      `json:"bg_id"`
	ParentBlockIndex uint16      `json:"parent_idx"`
	Flags            uint16      `json:"flags"` // FlagsNumber
	TileFlags        uint16      `json:"tile_flags"`
	X                uint32      `json:"x"`
	Y                uint32      `json:"y"`
	TileType         uint8       `json:"tile_type"`
	Extra            interface{} `json:"extra,omitempty"`
}

// TileFlag constants
const (
	TileFlagHasExtraData      uint16 = 0x01
	TileFlagHasParent         uint16 = 0x02
	TileFlagWasSpliced        uint16 = 0x04
	TileFlagWillSpawnSeedsToo uint16 = 0x08
	TileFlagIsSeedling        uint16 = 0x10
	TileFlagFlippedX          uint16 = 0x20
	TileFlagIsOn              uint16 = 0x40
	TileFlagIsOpenToPublic    uint16 = 0x80
	TileFlagBgIsOn            uint16 = 0x100
	TileFlagFgAltMode         uint16 = 0x200
	TileFlagIsWet             uint16 = 0x400
	TileFlagGlued             uint16 = 0x800
	TileFlagOnFire            uint16 = 0x1000
	TileFlagPaintedRed        uint16 = 0x2000
	TileFlagPaintedGreen      uint16 = 0x4000
	TileFlagPaintedBlue       uint16 = 0x8000
)

// DroppedItem represents an item dropped on the ground
type DroppedItem struct {
	ID    uint16  `json:"id"`
	X     float32 `json:"x"`
	Y     float32 `json:"y"`
	Count uint8   `json:"count"`
	Flags uint8   `json:"flags"`
	UID   uint32  `json:"uid"`
}

// Tile Extra Data Structs

type TileDoor struct {
	Text     string `json:"text"`
	OwnerUID uint32 `json:"owner_uid"`
}

type TileSign struct {
	Text  string `json:"text"`
	Flags uint8  `json:"flags"`
}

type TileLock struct {
	Settings     uint8    `json:"settings"`
	OwnerUID     uint32   `json:"owner_uid"`
	AccessCount  uint32   `json:"access_count"`
	AccessUIDs   []uint32 `json:"access_uids"`
	MinimumLevel uint8    `json:"minimum_level"`
}

type TileSeed struct {
	TimePassed     uint32 `json:"time_passed"`
	ItemOnTree     uint8  `json:"item_on_tree"`
	ReadyToHarvest bool   `json:"ready_to_harvest"`
}

type TileMailbox struct {
	Message1 string `json:"message_1"`
	Message2 string `json:"message_2"`
	Message3 string `json:"message_3"`
	Unknown  uint8  `json:"unknown"`
}

type TileBulletin struct {
	Message1 string `json:"message_1"`
	Message2 string `json:"message_2"`
	Message3 string `json:"message_3"`
	Unknown  uint8  `json:"unknown"`
}

type TileDice struct {
	Symbol uint8 `json:"symbol"`
}

type TileChemicalSource struct {
	TimePassed uint32 `json:"time_passed"`
}

type TileAchievementBlock struct {
	Unknown1 uint32 `json:"unknown_1"`
	TileType uint8  `json:"tile_type"`
}

type TileHearthMonitor struct {
	Unknown1   uint32 `json:"unknown_1"`
	PlayerName string `json:"player_name"`
}

type TileDonationBox struct {
	Message1 string `json:"message_1"`
	Message2 string `json:"message_2"`
	Message3 string `json:"message_3"`
	Unknown  uint8  `json:"unknown"`
}

type TileMannequin struct {
	Text       string `json:"text"`
	Unknown1   uint8  `json:"unknown_1"`
	Clothing1  uint32 `json:"clothing_1"`
	Clothing2  uint16 `json:"clothing_2"`
	Clothing3  uint16 `json:"clothing_3"`
	Clothing4  uint16 `json:"clothing_4"`
	Clothing5  uint16 `json:"clothing_5"`
	Clothing6  uint16 `json:"clothing_6"`
	Clothing7  uint16 `json:"clothing_7"`
	Clothing8  uint16 `json:"clothing_8"`
	Clothing9  uint16 `json:"clothing_9"`
	Clothing10 uint16 `json:"clothing_10"`
}

type TileBunnyEgg struct {
	EggPlaced uint32 `json:"egg_placed"`
}

type TileGamePack struct {
	Team uint8 `json:"team"`
}

type TileGameGenerator struct{}

type TileXenoniteCrystal struct {
	Unknown1 uint8  `json:"unknown_1"`
	Unknown2 uint32 `json:"unknown_2"`
}

type TilePhoneBooth struct {
	Clothing1 uint16 `json:"clothing_1"`
	Clothing2 uint16 `json:"clothing_2"`
	Clothing3 uint16 `json:"clothing_3"`
	Clothing4 uint16 `json:"clothing_4"`
	Clothing5 uint16 `json:"clothing_5"`
	Clothing6 uint16 `json:"clothing_6"`
	Clothing7 uint16 `json:"clothing_7"`
	Clothing8 uint16 `json:"clothing_8"`
	Clothing9 uint16 `json:"clothing_9"`
}

type TileCrystal struct {
	Message string `json:"message"`
}

type TileCrimeInProgress struct {
	Message  string `json:"message"`
	Unknown2 uint32 `json:"unknown_2"`
	Unknown3 uint8  `json:"unknown_3"`
}

type TileDisplayBlock struct {
	ItemID uint32 `json:"item_id"`
}

type TileVendingMachine struct {
	ItemID uint32 `json:"item_id"`
	Price  int32  `json:"price"`
}

type FishInfo struct {
	FishItemID uint32 `json:"fish_item_id"`
	Lbs        uint32 `json:"lbs"`
}

type TileFishTankPort struct {
	Flags  uint8      `json:"flags"`
	Fishes []FishInfo `json:"fishes"`
}

type TileSolarCollector struct {
	Unknown1 []uint8 `json:"unknown_1"`
}

type TileForge struct {
	Temperature uint32 `json:"temperature"`
}

type TileGivingTree struct {
	Unknown1 uint16 `json:"unknown_1"`
	Unknown2 uint32 `json:"unknown_2"`
}

type TileSteamOrgan struct {
	InstrumentType uint8  `json:"instrument_type"`
	Note           uint32 `json:"note"`
}

type SilkWormColor struct {
	A uint8 `json:"a"`
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

type TileSilkWorm struct {
	Type            uint8         `json:"type"`
	Name            string        `json:"name"`
	Age             uint32        `json:"age"`
	Unknown1        uint32        `json:"unknown_1"`
	Unknown2        uint32        `json:"unknown_2"`
	CanBeFed        uint8         `json:"can_be_fed"`
	FoodSaturation  uint32        `json:"food_saturation"`
	WaterSaturation uint32        `json:"water_saturation"`
	Color           SilkWormColor `json:"color"`
	SickDuration    uint32        `json:"sick_duration"`
}

type TileSewingMachine struct {
	BoltIDList []uint32 `json:"bolt_id_list"`
}

type TileCountryFlag struct {
	Country string `json:"country"`
}

type TileLobsterTrap struct{}

type TilePaintingEasel struct {
	ItemID uint32 `json:"item_id"`
	Label  string `json:"label"`
}

type PetBattleCageExtra struct {
	Damage uint32   `json:"damage"`
	Pet    []uint32 `json:"pet"`
}

type TilePetBattleCage struct {
	Label    string             `json:"label"`
	Unknown1 []uint8            `json:"unknown_1"`
	Extra    PetBattleCageExtra `json:"extra"`
}

type TilePetTrainer struct {
	Name          string   `json:"name"`
	PetTotalCount uint32   `json:"pet_total_count"`
	Unknown1      uint32   `json:"unknown_1"`
	PetsID        []uint32 `json:"pets_id"`
}

type TileSteamEngine struct {
	Temperature uint32 `json:"temperature"`
}

type TileLockBot struct {
	TimePassed uint32 `json:"time_passed"`
}

type TileWeatherMachine struct {
	Settings uint32 `json:"settings"`
}

type TileSpiritStorageUnit struct {
	GhostJarCount uint32 `json:"ghost_jar_count"`
}

type TileDataBedrock struct{}

type TileShelf struct {
	TopLeftItemID     uint32 `json:"top_left_item_id"`
	TopRightItemID    uint32 `json:"top_right_item_id"`
	BottomLeftItemID  uint32 `json:"bottom_left_item_id"`
	BottomRightItemID uint32 `json:"bottom_right_item_id"`
}

type TileVipEntrance struct {
	Unknown1   uint8    `json:"unknown_1"`
	OwnerUID   uint32   `json:"owner_uid"`
	AccessUIDs []uint32 `json:"access_uids"`
}

type TileChallengeTimer struct{}

type TileFishWallMount struct {
	Label  string `json:"label"`
	ItemID uint32 `json:"item_id"`
	Lb     uint8  `json:"lb"`
}

type TilePortrait struct {
	Label    string `json:"label"`
	Unknown1 uint32 `json:"unknown_1"`
	Unknown2 uint32 `json:"unknown_2"`
	Unknown3 uint32 `json:"unknown_3"`
	Unknown4 uint32 `json:"unknown_4"`
	Face     uint32 `json:"face"`
	Hat      uint32 `json:"hat"`
	Hair     uint32 `json:"hair"`
	Unknown5 uint16 `json:"unknown_5"`
	Unknown6 uint16 `json:"unknown_6"`
}

type TileGuildWeatherMachine struct {
	Unknown1 uint32 `json:"unknown_1"`
	Gravity  uint32 `json:"gravity"`
	Flags    uint8  `json:"flags"`
}

type TileFossilPrepStation struct {
	Unknown1 uint32 `json:"unknown_1"`
}

type TileDnaExtractor struct{}
type TileHowler struct{}

type TileChemsynthTank struct {
	CurrentChem uint32 `json:"current_chem"`
	TargetChem  uint32 `json:"target_chem"`
}

type StorageBlockItemInfo struct {
	ID     uint32 `json:"id"`
	Amount uint32 `json:"amount"`
}

type TileStorageBlock struct {
	Items []StorageBlockItemInfo `json:"items"`
}

type CookingOvenIngredientInfo struct {
	ItemID    uint32 `json:"item_id"`
	TimeAdded uint32 `json:"time_added"`
}

type TileCookingOven struct {
	TemperatureLevel uint32                      `json:"temperature_level"`
	Ingredients      []CookingOvenIngredientInfo `json:"ingredients"`
	Unknown1         uint32                      `json:"unknown_1"`
	Unknown2         uint32                      `json:"unknown_2"`
	Unknown3         uint32                      `json:"unknown_3"`
}

type TileAudioRack struct {
	Note   string `json:"note"`
	Volume uint32 `json:"volume"`
}

type TileGeigerCharger struct {
	SecondsFromStart  uint32 `json:"seconds_from_start"`
	SecondsToComplete uint32 `json:"seconds_to_complete"`
	ChargingPercent   uint32 `json:"charging_percent"`
	MinutesFromStart  uint32 `json:"minutes_from_start"`
	MinutesToComplete uint32 `json:"minutes_to_complete"`
}

type TileAdventureBegins struct{}
type TileTombRobber struct{}

type TileBalloonOMatic struct {
	TotalRarity uint32 `json:"total_rarity"`
	TeamType    uint8  `json:"team_type"`
}

type TileTrainingPort struct {
	FishLb       uint32 `json:"fish_lb"`
	FishStatus   uint16 `json:"fish_status"`
	FishID       uint32 `json:"fish_id"`
	FishTotalExp uint32 `json:"fish_total_exp"`
	FishLevel    uint32 `json:"fish_level"`
	Unknown2     uint32 `json:"unknown_2"`
}

type TileItemSucker struct {
	ItemIDToSuck uint32 `json:"item_id_to_suck"`
	ItemAmount   uint32 `json:"item_amount"`
	Flags        uint16 `json:"flags"`
	Limit        uint32 `json:"limit"`
}

type CyBotCommandData struct {
	CommandID     uint32 `json:"command_id"`
	IsCommandUsed uint32 `json:"is_command_used"`
}

type TileCyBot struct {
	SyncTimer    uint32             `json:"sync_timer"`
	Activated    uint32             `json:"activated"`
	CommandDatas []CyBotCommandData `json:"command_datas"`
}

type TileGuildItem struct{}

type TileGrowscan struct {
	Unknown1 uint8 `json:"unknown_1"`
}

type TileContainmentFieldPowerNode struct {
	GhostJarCount uint32   `json:"ghost_jar_count"`
	Unknown1      []uint32 `json:"unknown_1"`
}

type TileSpiritBoard struct {
	Unknown1 uint32 `json:"unknown_1"`
	Unknown2 uint32 `json:"unknown_2"`
	Unknown3 uint32 `json:"unknown_3"`
}

type TileStormyCloud struct {
	StingDuration    uint32 `json:"sting_duration"`
	IsSolid          uint32 `json:"is_solid"`
	NonSolidDuration uint32 `json:"non_solid_duration"`
}

type TileTemporaryPlatform struct {
	Unknown1 uint32 `json:"unknown_1"`
}

type TileSafeVault struct{}

type TileAngelicCountingCloud struct {
	IsRaffling uint32 `json:"is_raffling"`
	Unknown1   uint16 `json:"unknown_1"`
	AsciiCode  uint8  `json:"ascii_code"`
}

type TileInfinityWeatherMachine struct {
	IntervalMinutes    uint32   `json:"interval_minutes"`
	WeatherMachineList []uint32 `json:"weather_machine_list"`
}

type TilePineappleGuzzler struct{}

type TileKrakenGalaticBlock struct {
	PatternIndex uint8  `json:"pattern_index"`
	Unknown1     uint32 `json:"unknown_1"`
	R            uint8  `json:"r"`
	G            uint8  `json:"g"`
	B            uint8  `json:"b"`
}

type TileFriendsEntrance struct {
	OwnerUserID uint32 `json:"owner_user_id"`
	Unknown1    uint16 `json:"unknown_1"`
	Unknown2    uint16 `json:"unknown_2"`
}

type TileTesseractManipulator struct {
	Gems     uint32 `json:"gems"`
	Unknown2 uint32 `json:"unknown_2"`
	ItemID   uint32 `json:"item_id"`
	Unknown4 uint32 `json:"unknown_4"`
}

type ServerLocal struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type BotLoginPacket struct {
	Data string `json:"data"`
}

type Local struct {
	Name       string `json:"-"`
	Password   string `json:"password"`
	Status     string `json:"-"`
	HTTPStatus string `json:"http_status"`
	Ping       int    `json:"-"`
	Level      int    `json:"-"`
	SkinColor  uint32 `json:"skin_color"`

	//Idk
	GemCount     int `json:"gem_count"`
	PearlCount   int `json:"pearl_count"`
	VoucherCount int `json:"voucher_count"`

	//Mod Effect
	BuildLength int     `json:"build_length"`
	PunchLength int     `json:"punch_length"`
	Gravity     float32 `json:"gravity"`
	Velocity    float32 `json:"velocity"`
	HackType    int32   `json:"hack_type"`

	//Player
	Country     string  `json:"country"`
	UserID      int     `json:"userid"`
	NetID       int     `json:"netid"`
	EID         string  `json:"eid"`
	IP          string  `json:"ip"`
	PosX        float32 `json:"pos_x"`
	PosY        float32 `json:"pos_y"`
	SpeedX      float32 `json:"speed_x"`
	SpeedY      float32 `json:"speed_y"`
	ActiveItems []int   `json:"active_items"` //Clothes

	// Track
	Awesomeness       int     `json:"awesomeness"`
	GlobalPlaytime    int     `json:"global_playtime"`
	WorldLock         int     `json:"world_lock"`
	TotalPlaytime     float32 `json:"total_playtime"`
	FavoriteItems     []int   `json:"favorite_items"`
	FavoriteItemsSlot int     `json:"favorite_items_slot"`

	Players        []Players   `json:"players"`
	Inventory      []Inventory `json:"inventory"`
	InventorySlots int         `json:"inventory_slots"`
	World          World       `json:"world"`

	Server         ServerLocal    `json:"server"`
	BotLoginPacket BotLoginPacket `json:"bot_login_packet"`
	ServerHash     int            `json:"server_hash"`

	ServerInfo         ServerInfo    `json:"server_info"`
	Login              LoginPacket   `json:"login"`
	AllDebug           []string      `json:"all_debug"`
	OnGenericText      []string      `json:"on_generic_text"`
	GameMessageDebug   []string      `json:"game_message_debug"`
	TankPacketDebug    []string      `json:"tank_packet_debug"`
	OnVariantListDebug []VariantList `json:"on_variant_list_debug"`
	TrackPacketDebug   []string      `json:"track_packet_debug"`
}
