package bot

import (
	"fmt"
	"regexp"
	"strings"
)

func (b *Bot) handleGameMessage(data []byte) {
	message := string(data)
	message = strings.TrimRight(message, "\x00")

	b.logENet("[SYSTEM]: RECEIVED GAME MESSAGE:\n" + message)
	fmt.Println(message)

	b.mu.Lock()
	defer b.mu.Unlock()

	if strings.Contains(message, "logon_fail") {
		b.Status = "login_fail"
		b.Server.Enet.NowConnectedIP = b.Server.Enet.ServerIP
		b.Server.Enet.NowConnectedPort = b.Server.Enet.ServerPort
		go func() {
			b.DisconnectClient()
			b.Connect()
		}()
	} else if strings.Contains(message, "currently banned") {
		b.Status = "banned"
		b.Connected = false
	} else if strings.Contains(message, "View GT Twitter") {
		b.Status = "maintenance"
		b.Connected = false
	} else if strings.Contains(message, "password is wrong") {
		b.Status = "wrong password"
		b.Connected = false
	} else if strings.Contains(message, "Advanced Account Protection") {
		b.Status = "AAP"
		b.Connected = false
	} else if strings.Contains(message, "temporarily suspended") {
		b.Status = "temporarily suspended"
		b.Connected = false
	} else if strings.Contains(message, "has been suspended") {
		b.Status = "suspended"
		b.Connected = false
	} else if strings.Contains(message, "Growtopia is not quite ready for users") {
		b.Status = "Server issues"
		b.Connected = false
	} else if strings.Contains(message, "UPDATE REQUIRED") {
		b.Status = "UPDATE"
		re := regexp.MustCompile(`\$V([\d\.]+)`)
		match := re.FindStringSubmatch(message)
		if len(match) > 1 {
			newVersion := match[1]
			b.logENet("[SYSTEM]: Detected update version: " + newVersion)
		}
		b.Connected = false
	}
}
