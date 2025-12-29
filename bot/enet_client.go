package bot

import (
	"fmt"
	"log"
	"os"
	"vortenixgo/network/enet"
)

func (b *Bot) DisconnectClient() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Peer != nil {
		b.Peer.Disconnect(0)
		if b.Client != nil {
			b.Client.Flush()
		}
		b.Peer.Reset()
		b.Peer = nil
	}
	if b.Client != nil {
		b.Client.Destroy()
		b.Client = nil
		enet.Deinitialize()
	}
	// UDP::resetData(bot) - placeholder, translated as resetting specific bot data if needed.
	// bot.local.world.reset();
	b.World = ""
	b.Connected = false
}

func (b *Bot) GetPing() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Peer != nil && !b.Peer.IsNil() {
		b.Ping = b.Peer.GetRoundTripTime()
	} else {
		b.Status = "offline"
		b.Ping = 500
	}
}

func (b *Bot) ConnectClient() {
	// Note: Locking inside methods that modify state, but ConnectClient performs long operations.
	// Ideally we lock only when modifying struct fields, not during network calls if they are blocking.
	// However, C++ logic is sequential.

	// Pre-lock to set status/reset
	b.mu.Lock()
	b.World = ""
	b.Status = "Located Server"
	targetIP := b.Server.Enet.NowConnectedIP
	targetPort := b.Server.Enet.NowConnectedPort
	b.logENet("Connecting to Server...")
	b.mu.Unlock()

	if enet.Initialize() != 0 {
		log.Println("\n[FAILED TO INITIALIZE ENET]")
		b.logENet("Failed to initialize ENet")
		os.Exit(1)
	}

	client := enet.CreateHost(nil, 1, 2, 0, 0)
	if client == nil {
		log.Println("\n[FAILED TO CREATE CLIENT]")
		b.logENet("Failed to create client")
		os.Exit(1)
	}

	client.SetUsingNewPacket(true)
	client.SetChecksum()
	client.CompressWithRangeCoder()

	address, err := enet.NewAddress(targetIP, targetPort)
	if err != nil {
		log.Printf("\n[FAILED TO CONNECT TO SERVER]:%s", targetIP)
		client.Destroy()
		b.logENet("Failed to connect to server")
		os.Exit(1)
	}

	// address.port = bot.local.server.EnetNowPort; (Handled in NewAddress)

	peer := client.Connect(address, 2, 0)
	if peer == nil {
		log.Println("\n[NO PEER CONNECTED]")
		client.Destroy()
		b.logENet("No peer connected")
		os.Exit(1)
	}

	client.Flush()

	// Update bot state
	b.mu.Lock()
	b.Client = client
	b.Peer = peer
	b.mu.Unlock()
	// Logger equivalent
	log.Printf("[SYSTEM]: CONNECTION TO SERVER ESTABLISHED: %s:%d", targetIP, targetPort)
}

func (b *Bot) logENet(msg string) {
	if b.ShowENet {
		// log.Println("[ENET]: " + msg) // Optional: keep console clean or log it? User asked for "debug", usually implies visibility.
		if b.OnDebug != nil {
			b.OnDebug("ENET", msg, false)
		}
	}
}

func (b *Bot) EventListener() {
	b.logENet("Starting EventListener...")
	for {
		b.mu.Lock()
		shouldRun := b.Connected
		b.mu.Unlock()

		if !shouldRun {
			break
		}

		select {
		case <-b.stop:
			return
		default:
		}

		// Ensure Ping is updated regularly if connected
		b.GetPing()

		event, err := b.Client.Service(100) // Lower timeout for more responsive loop? 500 is fine too.
		if err != nil {
			b.logENet(fmt.Sprintf("Error in ENet service: %v", err))
			continue
		}

		if event != nil {
			switch event.Type {
			case enet.EventConnect:
				b.GetPing()
				ip := event.Peer.GetRemoteIP()
				port := event.Peer.GetRemotePort()
				msg := fmt.Sprintf("CONNECTED TO SERVER | IP: %s | PORT: %d", ip, port)
				fmt.Println("\n[ENET] " + msg)
				b.logENet(msg)

				b.mu.Lock()
				b.Status = "online"
				b.mu.Unlock()

			case enet.EventDisconnect:
				ip := event.Peer.GetRemoteIP()
				port := event.Peer.GetRemotePort()
				msg := fmt.Sprintf("DISCONNECTED FROM SERVER | IP: %s | PORT: %d", ip, port)
				fmt.Println("\n[ENET] " + msg)
				b.logENet(msg)

				b.DisconnectClient()
				b.mu.Lock()
				b.Status = "offline"
				b.mu.Unlock()
				return

			case enet.EventReceive:
				b.GetPing()
				// Only log receive if debug is on
				if b.ShowENet {
					b.logENet(fmt.Sprintf("Packet received. Length: %d", event.Packet.GetLength()))
				}

				if event.Packet != nil {
					event.Packet.Destroy()
				}
			}
		}
	}
}
