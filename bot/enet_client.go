package bot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
	"vortenixgo/network/enet"
)

var enetOnce sync.Once

const (
// Any remaining constants if any
)

func (b *Bot) EventListener(client *enet.Host) {
	b.logENet("Starting EventListener...")
	defer close(b.enetLoopDone)

	for {
		select {
		case <-b.stop:
			b.logENet("EventListener stopping due to stop signal.")
			return
		default:
		}

		b.GetPing()

		// Verify client is still valid/owned by bot?
		// Since we pass 'client' arg, we service THIS client.
		// But if b.Client changes, we should probably stop?
		b.mu.Lock()
		currentClient := b.Client
		b.mu.Unlock()

		if currentClient != client {
			b.logENet("EventListener stopping: Client mismatch or detached.")
			return
		}

		event, err := client.Service(100)
		if err != nil {
			b.mu.Lock()
			stillConnected := b.Connected
			b.mu.Unlock()

			if stillConnected {
				msg := fmt.Sprintf("Error in ENet service: %v", err)
				b.logENet(msg)
				// If service fails, Host is likely broken. destroy and stop.
				// We should detach b.Client so next Connect creates new one.
				b.mu.Lock()
				if b.Client == client {
					b.Client = nil
				}
				b.mu.Unlock()
				break
			}
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
				b.Connected = true
				b.mu.Unlock()

				if b.OnUpdate != nil {
					b.OnUpdate()
				}

			case enet.EventDisconnect:
				ip := event.Peer.GetRemoteIP()
				port := event.Peer.GetRemotePort()
				msg := fmt.Sprintf("DISCONNECTED FROM SERVER | IP: %s | PORT: %d", ip, port)
				fmt.Println("\n[ENET] " + msg)
				b.logENet(msg)

				b.mu.Lock()
				status := b.Status
				b.mu.Unlock()

				b.DisconnectPeer()

				b.mu.Lock()
				if status != "Redirecting" {
					b.Status = "offline"
				}
				b.mu.Unlock()

				if b.OnUpdate != nil {
					b.OnUpdate()
				}
				// Do NOT return/break here. Loop continues for next connection (Redirect).

			case enet.EventReceive:
				b.GetPing()
				b.OnReceive(event)
				event.Packet.Destroy()
			}
		}
	}
}

// DisconnectPeer only disconnects the peer but keeps ENet Host alive (for Redirect).
func (b *Bot) DisconnectPeer() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Peer != nil {
		b.Peer.Disconnect(0)
		b.Peer.Reset() // Optional depending on wrapper
		b.Peer = nil
	}
	b.Connected = false
}

// StopENet fully destroys ENet Host (Hard Stop).
func (b *Bot) StopENet() {
	b.mu.Lock()
	if b.Client == nil {
		b.mu.Unlock()
		return
	}

	b.logENet("Stopping ENet Client...")
	b.Connected = false

	if b.Peer != nil {
		b.Peer.Disconnect(0)
		b.Peer = nil
	}

	b.mu.Unlock()

	// 1. Signal stop
	func() {
		defer func() { recover() }()
		close(b.stop)
	}()

	// 2. Wait for confirmation
	b.logENet("Waiting for EventListener to exit...")
	select {
	case <-b.enetLoopDone:
		b.logENet("EventListener confirmed exited.")
	case <-time.After(1500 * time.Millisecond):
		b.logENet("EventListener exit timeout. Detaching and forcing destroy.")
	}

	// 3. Destroy Client
	b.mu.Lock()
	if b.Client != nil {
		b.Client.Flush()
		b.Client.Destroy()
		b.Client = nil
	}

	// 4. IMPORTANT: Reset control channels for next fresh run
	b.stop = make(chan struct{})
	b.enetLoopDone = make(chan struct{})
	close(b.enetLoopDone) // Mark it as "ready to start a new listener"

	b.Status = "Idle"
	b.mu.Unlock()
	b.logENet("ENet resources fully cleared.")

	if b.OnUpdate != nil {
		b.OnUpdate()
	}
}

// Alias for compatibility if needed, or update call sites.
// The interface asks for DisconnectClient, so let's update it to call StopENet?
// OR user DisconnectClient for Hard Stop.
func (b *Bot) DisconnectClient() {
	b.StopENet()
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
	b.mu.Lock()
	targetIP := b.Server.Enet.NowConnectedIP
	targetPort := b.Server.Enet.NowConnectedPort
	b.logENet(fmt.Sprintf("Connecting to %s:%d...", targetIP, targetPort))
	b.Status = "Connecting..."

	client := b.Client
	var initErr error

	if client == nil {
		enetOnce.Do(func() {
			if enet.Initialize() != 0 {
				initErr = fmt.Errorf("failed to initialize enet")
			}
		})
		if initErr != nil {
			b.mu.Unlock()
			b.logENet("ENet init failed")
			return
		}

		client = enet.CreateHost(nil, 1, 2, 0, 0)
		if client == nil {
			b.mu.Unlock()
			b.logENet("Host creation failed")
			return
		}

		client.SetUsingNewPacket(true)
		client.SetChecksum()
		client.CompressWithRangeCoder()
		client.SetMaxPacketLimits(16 * 1024 * 1024)
		client.SetSocketBuffers(2048*1024, 2048*1024)

		b.Client = client
	}

	// Always apply proxy settings (handles updates/re-connects)
	proxyStr := b.Proxy
	if proxyStr != "" {
		parts := strings.Split(proxyStr, ":")
		if len(parts) == 4 {
			pPort, _ := strconv.Atoi(parts[1])
			client.SetProxy(parts[0], pPort, parts[2], parts[3])
		} else if len(parts) == 2 {
			pPort, _ := strconv.Atoi(parts[1])
			client.SetProxy(parts[0], pPort, "", "")
		}
	} else {
		// Clear proxy if setting removed?
		// client.SetProxy("", 0, "", "")
	}

	// Listener check: If it's dead or channel is closed, start a new one.
	listenerDead := false
	select {
	case <-b.enetLoopDone:
		listenerDead = true
	default:
	}

	if listenerDead {
		b.enetLoopDone = make(chan struct{})
		b.logENet("Spawning EventListener...")
		go b.EventListener(client)
	}

	b.mu.Unlock()

	address, err := enet.NewAddress(targetIP, targetPort)
	if err != nil {
		b.logENet(fmt.Sprintf("Address failed: %v", err))
		return
	}

	peer := client.Connect(address, 2, 0)
	if peer == nil {
		b.logENet("Connect() returned nil")
		b.mu.Lock()
		b.Status = "No Peer"
		b.mu.Unlock()
		return
	}

	b.mu.Lock()
	b.Peer = peer
	b.Connected = true

	b.mu.Unlock()

	go b.ProcessQueue()

	log.Printf("[SYSTEM]: CONNECTION INITIATED TO %s:%d", targetIP, targetPort)
}

func (b *Bot) logENet(msg string) {
	if b.ShowENet {
		if b.OnDebug != nil {
			b.OnDebug("ENET", msg, false)
		}
	}
}

func (b *Bot) SendPacket(text string, mType uint32) {
	b.mu.Lock()
	if !b.Connected { // Check connection status
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()

	// Packet Structure: [4 bytes Type] + [Text Bytes]
	data := make([]byte, 4+len(text))

	binary.LittleEndian.PutUint32(data[:4], uint32(mType))
	copy(data[4:], []byte(text))

	// Send to queue
	b.packetQueue <- data
}

func (b *Bot) OnReceive(event *enet.Event) {
	packet := event.Packet
	if packet == nil {
		return
	}

	data := packet.GetData()
	length := len(data)

	if length < 4 {
		b.logENet("[BAD PACKET LENGTH < 4]")
		return
	}

	// Hex Debugging
	if b.ShowENet {
		maxLen := 500
		hexStr := ""
		if length > maxLen {
			var sb strings.Builder
			for i, b := range data[:maxLen] {
				if i > 0 {
					sb.WriteByte(' ')
				}
				fmt.Fprintf(&sb, "%02x", b)
			}
			hexStr = sb.String() + " ..."
			b.logENet(fmt.Sprintf("Packet received. Length: %d | Data (truncated): %s", length, hexStr))
		} else {
			var sb strings.Builder
			for i, b := range data {
				if i > 0 {
					sb.WriteByte(' ')
				}
				fmt.Fprintf(&sb, "%02x", b)
			}
			hexStr = sb.String()
			b.logENet(fmt.Sprintf("Packet received. Length: %d | Data: %s", length, hexStr))
		}
	}

	packetId := binary.LittleEndian.Uint32(data[:4])

	switch packetId {
	case NET_MESSAGE_SERVER_HELLO:
		b.handleHelloPacket()
	case NET_MESSAGE_GAME_MESSAGE:
		b.handleGameMessage(data[4:])
	case NET_MESSAGE_GAME_PACKET:
		b.handleGamePacket(data[4:])
	case NET_MESSAGE_ERROR:
		b.logENet("RECEIVED NET_MESSAGE_ERROR")
	case NET_MESSAGE_TRACK:
		b.handleTrackPacket(data)
	}
}

func (b *Bot) ProcessQueue() {
	b.logENet("Starting Packet Queue Processor...")
	for {
		b.mu.Lock()
		connected := b.Connected
		peer := b.Peer
		b.mu.Unlock()

		if !connected || peer == nil || peer.IsNil() {
			return
		}

		select {
		case <-b.stop:
			return
		case pkt := <-b.packetQueue:
			b.mu.Lock()
			currentPeer := b.Peer
			if currentPeer != nil && !currentPeer.IsNil() {
				// Unified Debugging for sent packets
				if b.ShowENet && len(pkt) >= 4 {
					pktId := binary.LittleEndian.Uint32(pkt[:4])

					if pktId == NET_MESSAGE_GAME_PACKET && len(pkt) >= 60 {
						var p TankPacketStruct
						binary.Read(bytes.NewReader(pkt[4:]), binary.LittleEndian, &p)

						logMsg := fmt.Sprintf("[SYSTEM]: SENDING GAME PACKET (%d):\n\n", pktId)
						logMsg += fmt.Sprintf("  TYPE: %s\n", b.convertTankPacketType(p.Type))
						logMsg += fmt.Sprintf("  OBJECT_TYPE: %d\n", p.ObjectType)
						logMsg += fmt.Sprintf("  JUMP_COUNT: %d\n", p.JumpCount)
						logMsg += fmt.Sprintf("  ANIMATION_TYPE: %d\n", p.AnimationType)
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
					} else if (pktId == NET_MESSAGE_GENERIC_TEXT || pktId == NET_MESSAGE_GAME_MESSAGE) && len(pkt) > 4 {
						msg := string(pkt[4:])
						b.logENet(fmt.Sprintf("[SYSTEM]: SENDING TEXT PACKET (%d):\n%s", pktId, msg))
					} else {
						b.logENet(fmt.Sprintf("[SYSTEM]: SENDING NET_MESSAGE TYPE: %d (Length: %d)", pktId, len(pkt)))
					}
				}

				currentPeer.Send(0, pkt, 1) // 1 = ENET_PACKET_FLAG_RELIABLE
			}
			b.mu.Unlock()
			// Small delay between packets to prevent spamming
			time.Sleep(1 * time.Millisecond)
		}
	}
}
