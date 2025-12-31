package main

import (
	"fmt"
	"log"
	"net/http"
	"vortenixgo/bot"
	"vortenixgo/database"
	"vortenixgo/network/ws"
)

func main() {
	// Initialize Bot Manager
	_ = bot.BotManager // Ensures init() runs

	// Initialize Item Database
	log.Println("[Startup] Loading item database...")
	if err := database.InitializeItemDatabase("items.dat"); err != nil {
		log.Printf("[Startup] Warning: Failed to load items.dat: %v", err)
		log.Println("[Startup] Database features will be disabled")
	} else {
		db := database.GetGlobalItemDB()
		log.Printf("[Startup] Item database loaded successfully: %d items (version %d)", db.ItemCount, db.Version)
	}

	// Initialize WS Hub
	hub := ws.NewHub()
	hub.OnConnect = HandleBotConnect
	hub.OnDisconnect = HandleBotDisconnect
	go hub.Run()

	// Serve Static Files
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	// WebSocket Endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(hub, w, r)
	})

	port := "8080"
	fmt.Printf("VortenixGO Server started at http://localhost:%s\n", port)

	// Start Server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
