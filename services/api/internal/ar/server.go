package ar

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"tale-backend/internal/realtime"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now (adjust for production)
	},
}

// Server manages WebSocket connections to AR frontends
type Server struct {
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex
	broadcast  chan realtime.ChatAndARResponse
}

// NewServer creates a new AR server
func NewServer() *Server {
	return &Server{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan realtime.ChatAndARResponse, 100),
	}
}

// Start starts the AR server on the given port
func (s *Server) Start(port int) error {
	// WebSocket endpoint
	http.HandleFunc("/ws", s.handleWebSocket)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Start broadcast goroutine
	go s.broadcastLoop()

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("AR Server starting on %s\n", addr)
	fmt.Printf("WebSocket endpoint: ws://localhost%s/ws\n", addr)

	return http.ListenAndServe(addr, nil)
}

// handleWebSocket handles WebSocket connections from AR frontends
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	s.clientsMux.Lock()
	s.clients[conn] = true
	s.clientsMux.Unlock()

	fmt.Println("AR client connected")

	// Handle disconnection
	defer func() {
		s.clientsMux.Lock()
		delete(s.clients, conn)
		s.clientsMux.Unlock()
		conn.Close()
		fmt.Println("AR client disconnected")
	}()

	// Read messages from client (if needed)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// Handle client messages if needed
	}
}

// broadcastLoop broadcasts AR responses to all connected clients
func (s *Server) broadcastLoop() {
	for response := range s.broadcast {
		s.clientsMux.RLock()
		for client := range s.clients {
			err := client.WriteJSON(response)
			if err != nil {
				log.Printf("Error broadcasting to client: %v", err)
				client.Close()
				s.clientsMux.Lock()
				delete(s.clients, client)
				s.clientsMux.Unlock()
			}
		}
		s.clientsMux.RUnlock()
	}
}

// SendResponse sends an AR response to all connected clients
func (s *Server) SendResponse(response realtime.ChatAndARResponse) {
	select {
	case s.broadcast <- response:
	default:
		log.Println("Broadcast channel full, dropping response")
	}
}

// ConnectRealtimeClient connects a Realtime client to the AR server
func (s *Server) ConnectRealtimeClient(client *realtime.Client) {
	go func() {
		for response := range client.GetARResponseChan() {
			s.SendResponse(response)
		}
	}()
}

// ConnectGeminiClient connects a Gemini client to the AR server
func (s *Server) ConnectGeminiClient(client *realtime.GeminiClient) {
	go func() {
		for response := range client.GetARResponseChan() {
			s.SendResponse(response)
		}
	}()
}
