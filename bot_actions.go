package main

import (
	"log"
	"vortenixgo/bot"
	"vortenixgo/network"
	"vortenixgo/network/ws"
)

// Orchestrator handles high-level bot actions to avoid circular imports
func HandleBotConnect(b *bot.Bot, hub *ws.Hub) {
	go func() {
		b.Lock()
		if b.Connected {
			b.Unlock()
			b.Disconnect() // Stop previous connection if any
		} else {
			b.Unlock()
		}

		b.Lock()
		b.Status = "Getting Server Address"
		// Reset previous server data to ensure fresh start
		b.Server.Enet.ServerIP = ""
		b.Server.Enet.ServerPort = 0
		b.Login.Meta = ""
		b.Unlock()
		hub.BroadcastBotUpdate()

		// 1. Prepare Identity & Hashes (Preserves existing MAC/RID/WK if provided)
		generator := &bot.GenerateLoginData{}
		generator.GenerateAllLoginData(b)

		// 2. Get Meta
		handler := network.NewHTTPHandler(b.Proxy)
		err := handler.GetMeta(b)
		if err != nil {
			b.Lock()
			// Ensure it's set to HTTP_BLOCK if not already set by GetMeta
			if b.Status != "HTTP_BLOCK" {
				b.Status = "HTTP_BLOCK"
			}
			b.Unlock()
			hub.BroadcastBotUpdate()
			return
		}

		// 4. Create Login Packet
		generator.CreateLoginPacket(b)

		// 5. Token Management (CheckToken)
		b.Lock()
		ltoken := b.Server.HTTPS.LToken
		b.Unlock()

		if ltoken != "" {
			b.Lock()
			b.Status = "Checking Token..."
			b.Unlock()
			hub.BroadcastBotUpdate()

			newToken, err := handler.CheckToken(b)
			if err != nil {
				log.Printf("[Orchestrator][%s] CheckToken error: %v", b.Name, err)
				hub.BroadcastBotUpdate()

				// Jalankan Glog Flow HANYA jika token memang tidak valid.
				// Jika error lain (Bad Gateway, Too Many People, dll), langsung berhenti.
				b.Lock()
				isInvalid := b.Status == "Invalid Token"
				b.Unlock()

				if isInvalid {
					handleGlogFlow(b, handler, hub)
				}
				return
			}

			log.Printf("[Orchestrator][%s] Token Validated: %s", b.Name, newToken)
			hub.BroadcastBotUpdate()
		} else {
			// No token: Get Dashboard/Form URL first
			handleGlogFlow(b, handler, hub)

			// If it's legacy, it might continue to connect,
			// but for Gmail/Apple it usually stops at "Awaiting Glog"
			if b.Type == bot.BotTypeGmail || b.Type == bot.BotTypeApple {
				return
			}
		}

		// 6. Final Connection State
		b.Connect()
		hub.BroadcastBotUpdate()

		// 3. Start ENet event loop
		b.StartEventLoop()
	}()
}

func handleGlogFlow(b *bot.Bot, handler *network.HTTPHandler, hub *ws.Hub) {
	b.Lock()
	b.Status = "Getting Login Form..."
	b.Unlock()
	hub.BroadcastBotUpdate()

	err := handler.GetDashboard(b)
	if err != nil {
		b.Lock()
		// Only set generic error if status wasn't already updated to something specific like HTTP_BLOCK
		if b.Status == "Getting Login Form..." {
			b.Status = "Login Form Error"
		}
		b.Unlock()
		hub.BroadcastBotUpdate()
		log.Printf("[Orchestrator][%s] GetDashboard failed: %v", b.Name, err)
		return
	}

	// GetCookies step: Only for Legacy bots
	// Gmail/Apple bots already got cookies from GetDashboard and don't need form token from this page.
	b.Lock()
	isLegacy := b.Type == bot.BotTypeLegacy
	b.Unlock()

	if isLegacy {
		b.Lock()
		b.Status = "Getting Cookies..."
		b.Unlock()
		hub.BroadcastBotUpdate()

		err = handler.GetCookies(b)
		if err != nil {
			b.Lock()
			// If status wasn't set inside GetCookies (e.g. general network error), set default
			if b.Status == "Getting Cookies..." {
				b.Status = "Cookies Not Found"
			}
			b.Unlock()
			hub.BroadcastBotUpdate()
			log.Printf("[Orchestrator][%s] GetCookies failed: %v", b.Name, err)
			return
		}
	} else {
		log.Printf("[Orchestrator][%s] Skipping GetCookies (Cookies obtained from Dashboard)", b.Name)
	}

	b.Lock()
	b.Status = "Getting Token..."
	b.Unlock()
	hub.BroadcastBotUpdate()

	err = handler.GetToken(b)
	if err != nil {
		b.Lock()
		if b.Status == "Getting Token..." {
			b.Status = "GetToken Failed"
		}
		b.Unlock()
		hub.BroadcastBotUpdate()
		log.Printf("[Orchestrator][%s] GetToken failed: %v", b.Name, err)
		return
	}

	b.Lock()
	b.Status = "Token Obtained"
	// Optional: You might want to auto-connect to ENet here if needed,
	// or return so the main routine picks it up (but main routine is likely finished/waiting).
	// Since handleGlogFlow is called from Orchestrator, usually we want to trigger Connect().
	b.Unlock()

	// If getting token succeeded, we should probably proceed to connect to game server (ENet).
	// Calls Connect() to update status to "Connecting..." and trigger any UI/logic needed.
	b.Connect()
	b.StartEventLoop()
	hub.BroadcastBotUpdate()
}
