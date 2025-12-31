package bot

import (
	"fmt"
	"log"
	"sort"
	"sync"
)

// Manager handles the lifecycle of multiple bots
type Manager struct {
	Bots map[string]*Bot
	mu   sync.RWMutex
}

// Global instance
var BotManager *Manager

func init() {
	BotManager = NewManager()
}

func NewManager() *Manager {
	return &Manager{
		Bots: make(map[string]*Bot),
	}
}

// AddBot creates and registers a new bot
func (m *Manager) AddBot(botType BotType, name string, password string, glog string, proxy string) (*Bot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("bot_%s", name) // Simple ID generation
	if _, exists := m.Bots[id]; exists {
		return nil, fmt.Errorf("bot with name %s already exists", name)
	}

	bot := NewBot(id, botType, name, password, glog)
	bot.Proxy = proxy
	m.Bots[id] = bot
	log.Printf("[BotManager] Added bot ID: %s. Total bots: %d", id, len(m.Bots))
	return bot, nil
}

// RemoveBot stops and removes a bot
func (m *Manager) RemoveBot(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bot, exists := m.Bots[id]
	if !exists {
		return fmt.Errorf("bot not found")
	}

	bot.DisconnectClient() // Stop ENet and EventListener
	bot.Disconnect()       // Stop general bot loop
	delete(m.Bots, id)
	log.Printf("[BotManager] Removed bot ID: %s. Remaining: %d", id, len(m.Bots))
	return nil
}

// GetBot returns a bot by ID
func (m *Manager) GetBot(id string) (*Bot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bot, ok := m.Bots[id]
	return bot, ok
}

// GetAllBots returns a list of all bots sorted by ID
func (m *Manager) GetAllBots() []*Bot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bots := make([]*Bot, 0, len(m.Bots))
	for _, b := range m.Bots {
		bots = append(bots, b)
	}

	// Sort to maintain consistent order
	sort.Slice(bots, func(i, j int) bool {
		return bots[i].ID < bots[j].ID
	})

	return bots
}
