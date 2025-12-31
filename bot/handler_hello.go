package bot

import "fmt"

func (b *Bot) handleHelloPacket() {
	b.logENet("[SYSTEM]: RECEIVED HELLO LOGIN REQUEST")

	generator := &GenerateLoginData{}
	b.mu.Lock()
	uuidToken := b.Login.UUIDToken
	protocol := b.Login.Protocol
	ltoken := b.Server.HTTPS.LToken
	platformID := b.Login.PlatformID
	b.mu.Unlock()

	if uuidToken == "" {
		logToken := fmt.Sprintf("protocol|%s\nltoken|%s\nplatformID|%s", protocol, ltoken, platformID)
		b.SendPacket(logToken, NET_MESSAGE_GENERIC_TEXT)
		b.logENet("[SYSTEM]: SENDING REQUEST\n" + logToken)
	} else {
		loginPacket := generator.CreateLoginPacket(b)
		b.SendPacket(loginPacket, NET_MESSAGE_GENERIC_TEXT)
		b.logENet("[SYSTEM]: SENDING REQUEST:\n" + loginPacket)
	}
}
