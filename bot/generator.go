package bot

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// GenerateLoginData handles the creation of bot identity and login packets
type GenerateLoginData struct{}

var genSource = rand.NewSource(time.Now().UnixNano())
var genRand = rand.New(genSource)

// ProtonHash64 matches the C++ 64-bit behavior provided in the snippet
func (g *GenerateLoginData) ProtonHash64(data string) int64 {
	var hash int64 = 0x55555555
	for i := 0; i < len(data); i++ {
		charVal := int64(data[i])
		hash = charVal + (int64(uint64(hash) >> 27)) + (hash << 5)
	}
	return hash
}

func (g *GenerateLoginData) MD5(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (g *GenerateLoginData) SHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (g *GenerateLoginData) GenerateRid() string {
	bytes := make([]byte, 16)
	genRand.Read(bytes)
	return strings.ToUpper(hex.EncodeToString(bytes))
}

func (g *GenerateLoginData) GenerateWk() string {
	bytes := make([]byte, 16)
	genRand.Read(bytes)
	return strings.ToUpper(hex.EncodeToString(bytes))
}

func (g *GenerateLoginData) GenerateMac() string {
	bytes := make([]byte, 6)
	genRand.Read(bytes)
	// Locally administered address bit
	bytes[0] = (bytes[0] & 0xFE) | 0x02
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5])
}

func (g *GenerateLoginData) RandomHex(length int, upper bool) string {
	bytes := make([]byte, length/2)
	genRand.Read(bytes)
	res := hex.EncodeToString(bytes)
	if upper {
		return strings.ToUpper(res)
	}
	return res
}

func (g *GenerateLoginData) GenerateKlv(protocol, version, rid string) string {
	salts := []string{
		"e9fc40ec08f9ea6393f59c65e37f750aacddf68490c4f92d0d2523a5bc02ea63",
		"c85df9056ee603b849a93e1ebab5dd5f66e1fb8b2f4a8caef8d13b9f9e013fa4",
		"3ca373dffbf463bb337e0fd768a2f395b8e417475438916506c721551f32038d",
		"73eff5914c61a20a71ada81a6fc7780700fb1c0285659b4899bc172a24c14fc1",
	}

	cv0 := g.SHA256(g.MD5(g.SHA256(protocol)))
	cv1 := g.SHA256(g.SHA256(version))
	cv2 := g.SHA256(g.SHA256(protocol) + salts[3])

	ridHash := g.SHA256(g.MD5(g.SHA256(rid)))

	return g.SHA256(cv0 + salts[0] + cv1 + salts[1] + ridHash + salts[2] + cv2)
}

func (g *GenerateLoginData) GenerateAllLoginData(b *Bot) {
	b.Lock()
	defer b.Unlock()

	if b.Login.Wk == "" {
		b.Login.Wk = g.GenerateWk()
	}
	if b.Login.Mac == "" {
		b.Login.Mac = g.GenerateMac()
	}
	if b.Login.Rid == "" {
		b.Login.Rid = g.GenerateRid()
	}

	b.Login.Klv = g.GenerateKlv(b.Login.Protocol, b.Login.GameVersion, b.Login.Rid)

	hashVal := g.ProtonHash64(b.Login.Mac + "RT")
	b.Login.Hash = fmt.Sprintf("%d", hashVal)

	hash2Val := g.ProtonHash64(g.RandomHex(16, true) + "RT")
	b.Login.Hash2 = fmt.Sprintf("%d", hash2Val)
}

func (g *GenerateLoginData) CreateLoginPacket(b *Bot) string {
	b.Lock()
	defer b.Unlock()

	var lp strings.Builder
	lp.WriteString("tankIDName|\n")    // Force empty as requested
	lp.WriteString("tankIDPass|\n")    // Force empty as requested
	lp.WriteString("requestedName|\n") // Force empty as requested
	lp.WriteString(fmt.Sprintf("f|%s\n", b.Login.F))
	lp.WriteString(fmt.Sprintf("protocol|%s\n", b.Login.Protocol))
	lp.WriteString(fmt.Sprintf("game_version|%s\n", b.Login.GameVersion))
	lp.WriteString(fmt.Sprintf("fz|%s\n", b.Login.Fz))
	lp.WriteString(fmt.Sprintf("cbits|%s\n", b.Login.Cbits))
	lp.WriteString(fmt.Sprintf("player_age|%s\n", b.Login.PlayerAge))
	lp.WriteString(fmt.Sprintf("GDPR|%s\n", b.Login.GDPR))
	lp.WriteString(fmt.Sprintf("FCMToken|%s\n", b.Login.FCMToken))
	lp.WriteString(fmt.Sprintf("category|%s\n", b.Login.Category))
	lp.WriteString(fmt.Sprintf("totalPlaytime|%s\n", b.Login.TotalPlaytime))
	lp.WriteString(fmt.Sprintf("klv|%s\n", b.Login.Klv))
	lp.WriteString(fmt.Sprintf("hash2|%s\n", b.Login.Hash2))
	lp.WriteString(fmt.Sprintf("meta|%s\n", b.Login.Meta))
	lp.WriteString(fmt.Sprintf("fhash|%s\n", b.Login.FHash))
	lp.WriteString(fmt.Sprintf("rid|%s\n", b.Login.Rid))
	lp.WriteString(fmt.Sprintf("platformID|%s\n", b.Login.PlatformID))
	lp.WriteString(fmt.Sprintf("deviceVersion|%s\n", b.Login.DeviceVersion))
	lp.WriteString(fmt.Sprintf("country|%s\n", b.Login.Country))
	lp.WriteString(fmt.Sprintf("hash|%s\n", b.Login.Hash))
	lp.WriteString(fmt.Sprintf("mac|%s\n", b.Login.Mac))
	lp.WriteString(fmt.Sprintf("wk|%s\n", b.Login.Wk))
	lp.WriteString(fmt.Sprintf("zf|%s\n", b.Login.Zf))
	lp.WriteString(fmt.Sprintf("lmode|%s\n", b.Login.LMode))

	if b.Login.UUIDToken != "" {
		lp.WriteString(fmt.Sprintf("user|%s\n", b.Login.User))
		lp.WriteString(fmt.Sprintf("token|%s\n", b.Login.UserToken))
		lp.WriteString(fmt.Sprintf("UUIDToken|%s\n", b.Login.UUIDToken))
		if b.Login.DoorID != "" {
			lp.WriteString(fmt.Sprintf("doorID|%s\n", b.Login.DoorID))
		}
		lp.WriteString(fmt.Sprintf("aat|%s", b.Login.Aat))
	}

	packet := lp.String()
	b.Login.LoginPkt = packet
	return packet
}
