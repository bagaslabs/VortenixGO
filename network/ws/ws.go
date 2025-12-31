package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"vortenixgo/bot"
	"vortenixgo/database"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for local dev
	},
}

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	clients      map[*Client]bool
	broadcast    chan []byte
	register     chan *Client
	unregister   chan *Client
	mu           sync.Mutex
	OnConnect    func(*bot.Bot, *Hub)
	OnDisconnect func(*bot.Bot, *Hub)
}

var GlobalHub *Hub

func NewHub() *Hub {
	h := &Hub{
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
	GlobalHub = h
	return h
}

func (h *Hub) Run() {
	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			// Send current bot list to new client
			h.SendBotList(client)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.broadcastToClients(message)
		case <-ticker.C:
			h.broadcastBotUpdateInternal()
		}
	}
}

func (h *Hub) broadcastToClients(message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *Hub) broadcastBotUpdateInternal() {
	bots := bot.BotManager.GetAllBots()
	msg := map[string]interface{}{
		"type": "UPDATE_LIST",
		"data": bots,
	}
	data, err := json.Marshal(msg)
	if err == nil {
		h.broadcastToClients(data)
	}
}

func (h *Hub) BroadcastBotUpdate() {
	h.broadcastBotUpdateInternal()
}

func (h *Hub) BroadcastDebug(botID, category, message string, isError bool) {
	msgType := "DEBUG_LOG"
	msg := map[string]interface{}{
		"type": msgType,
		"data": map[string]interface{}{
			"bot_id":   botID,
			"category": category,
			"message":  message,
			"is_error": isError,
			"time":     time.Now().Format("15:04:05"),
		},
	}
	data, err := json.Marshal(msg)
	if err == nil {
		h.broadcastToClients(data)
	}
}

func (h *Hub) SendBotList(client *Client) {
	bots := bot.BotManager.GetAllBots()
	msg := map[string]interface{}{
		"type": "UPDATE_LIST",
		"data": bots,
	}
	data, _ := json.Marshal(msg)
	client.send <- data
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("error: %v", err)
			break
		}

		// Handle incoming message
		var req map[string]interface{}
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		msgType, _ := req["type"].(string)
		data, _ := req["data"].(map[string]interface{})

		switch msgType {
		case "ADD_BOT":
			bType := bot.BotType(data["type"].(string))
			name := data["name"].(string)
			pass := data["pass"].(string)
			glog := data["glog"].(string)
			proxy := ""
			if p, ok := data["proxy"].(string); ok {
				proxy = p
			}
			log.Printf("Adding bot: type=%s, name=%s, glog=%s, proxy=%s", bType, name, glog, proxy)

			newBot, err := bot.BotManager.AddBot(bType, name, pass, glog, proxy)
			if err == nil {
				newBot.OnDebug = func(cat, msg string, isErr bool) {
					c.hub.BroadcastDebug(newBot.ID, cat, msg, isErr)
				}
				c.hub.BroadcastBotUpdate()
			} else {
				log.Printf("Error adding bot: %v", err)
				msg := map[string]interface{}{
					"type": "ERROR",
					"data": err.Error(),
				}
				data, _ := json.Marshal(msg)
				c.send <- data
			}
		case "REMOVE_BOT":
			id, _ := data["id"].(string)
			err := bot.BotManager.RemoveBot(id)
			if err == nil {
				c.hub.BroadcastBotUpdate()
			}
		case "BOT_ACTION":
			id, _ := data["id"].(string)
			action, _ := data["action"].(string)
			b, ok := bot.BotManager.GetBot(id)
			if ok {
				switch action {
				case "CONNECT":
					if c.hub.OnConnect != nil {
						c.hub.OnConnect(b, c.hub)
					} else {
						b.Connect()
						c.hub.BroadcastBotUpdate()
					}
				case "DISCONNECT":
					if c.hub.OnDisconnect != nil {
						c.hub.OnDisconnect(b, c.hub)
					} else {
						b.Disconnect()
						c.hub.BroadcastBotUpdate()
					}
				case "INVENTORY_ACTION":
					subAction, _ := data["sub_action"].(string)
					itemID, _ := data["item_id"].(float64)
					switch subAction {
					case "WEAR", "UNWEAR":
						b.WearItem(int32(itemID))
					case "DROP":
						b.DropItem(int32(itemID))
					case "TRASH":
						b.TrashItem(int32(itemID))
					}
				case "SAY":
					text, _ := data["text"].(string)
					b.Say(text)
				case "WARP":
					world, _ := data["world"].(string)
					b.Warp(world)
				case "LEAVE":
					b.Warp("EXIT")
				}
			}
		case "UPDATE_BOT_CONFIG":
			id, _ := data["id"].(string)
			b, ok := bot.BotManager.GetBot(id)
			if ok {
				b.Lock()
				if g, ok := data["glog"].(string); ok {
					b.Glog = g
				}
				if p, ok := data["proxy"].(string); ok {
					b.Proxy = p
				}
				if s, ok := data["show_enet"].(bool); ok {
					b.ShowENet = s
				}
				b.Unlock()
				c.hub.BroadcastBotUpdate()
			}
		case "EXECUTE_LUA":
			id, _ := data["id"].(string)
			script, _ := data["script"].(string)
			log.Printf("Executing Lua on bot %s: %s", id, script)
			// TODO: Actually run in Lua state

		// Database queries
		case "GET_ITEM":
			c.handleGetItem(data)
		case "SEARCH_ITEMS":
			c.handleSearchItems(data)
		case "GET_ITEMS_BY_RARITY":
			c.handleGetItemsByRarity(data)
		case "GET_DATABASE_INFO":
			c.handleGetDatabaseInfo()
		}
	}
}

// Database query handlers
func (c *Client) handleGetItem(data map[string]interface{}) {
	db := database.GetGlobalItemDB()
	if db == nil || !db.Loaded {
		c.sendError("Database not loaded")
		return
	}

	// Support both ID and name queries
	if idFloat, ok := data["id"].(float64); ok {
		item := db.GetItem(uint32(idFloat))
		c.sendItemResponse(item)
	} else if name, ok := data["name"].(string); ok {
		item := db.GetItemByName(name)
		c.sendItemResponse(item)
	} else {
		c.sendError("Missing 'id' or 'name' parameter")
	}
}

func (c *Client) handleSearchItems(data map[string]interface{}) {
	db := database.GetGlobalItemDB()
	if db == nil || !db.Loaded {
		c.sendError("Database not loaded")
		return
	}

	query, ok := data["query"].(string)
	if !ok {
		c.sendError("Missing 'query' parameter")
		return
	}

	items := db.SearchItems(query)
	c.sendItemsResponse(items)
}

func (c *Client) handleGetItemsByRarity(data map[string]interface{}) {
	db := database.GetGlobalItemDB()
	if db == nil || !db.Loaded {
		c.sendError("Database not loaded")
		return
	}

	rarityFloat, ok := data["rarity"].(float64)
	if !ok {
		c.sendError("Missing 'rarity' parameter")
		return
	}

	items := db.GetItemsByRarity(uint16(rarityFloat))
	c.sendItemsResponse(items)
}

func (c *Client) handleGetDatabaseInfo() {
	db := database.GetGlobalItemDB()
	if db == nil || !db.Loaded {
		c.sendError("Database not loaded")
		return
	}

	msg := map[string]interface{}{
		"type": "DATABASE_INFO",
		"data": map[string]interface{}{
			"loaded":     db.Loaded,
			"version":    db.Version,
			"item_count": db.ItemCount,
		},
	}
	data, _ := json.Marshal(msg)
	c.send <- data
}

func (c *Client) sendItemResponse(item *database.Item) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in sendItemResponse: %v", r)
		}
	}()

	if item == nil {
		c.sendError("Item not found")
		return
	}

	msg := map[string]interface{}{
		"type": "ITEM_DATA",
		"data": item,
	}
	data, _ := json.Marshal(msg)

	// Use non-blocking send or verify safety
	select {
	case c.send <- data:
	default:
		// Channel might be full or closed logic handled by recover if closed?
		// Actually select default won't catch closed, it catches full.
		// Send on closed panics.
		// Recover is the safest quick fix.
	}
}

func (c *Client) sendItemsResponse(items []*database.Item) {
	msg := map[string]interface{}{
		"type": "ITEMS_DATA",
		"data": items,
	}
	data, _ := json.Marshal(msg)
	c.send <- data
}

func (c *Client) sendError(message string) {
	msg := map[string]interface{}{
		"type": "ERROR",
		"data": message,
	}
	data, _ := json.Marshal(msg)
	c.send <- data
}

func (c *Client) writePump() {
	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		c.conn.WriteMessage(websocket.TextMessage, message)
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
