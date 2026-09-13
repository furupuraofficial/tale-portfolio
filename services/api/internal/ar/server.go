package ar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"tale-backend/internal/realtime"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: websocketOriginAllowed,
}

// Server manages WebSocket connections to AR frontends
type Server struct {
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex
	broadcast  chan realtime.ChatAndARResponse
	serverMux  sync.RWMutex
	httpServer *http.Server
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
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Start broadcast goroutine
	go s.broadcastLoop()

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("AR Server starting on %s\n", addr)
	fmt.Printf("WebSocket endpoint: ws://localhost%s/ws\n", addr)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(http.DefaultServeMux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	s.serverMux.Lock()
	s.httpServer = httpServer
	s.serverMux.Unlock()

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown stops accepting requests and closes active WebSocket connections.
func (s *Server) Shutdown(ctx context.Context) error {
	s.clientsMux.Lock()
	for client := range s.clients {
		_ = client.Close()
		delete(s.clients, client)
	}
	s.clientsMux.Unlock()

	s.serverMux.RLock()
	httpServer := s.httpServer
	s.serverMux.RUnlock()
	if httpServer == nil {
		return nil
	}
	return httpServer.Shutdown(ctx)
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
		clients := make([]*websocket.Conn, 0, len(s.clients))
		for client := range s.clients {
			clients = append(clients, client)
		}
		s.clientsMux.RUnlock()

		for _, client := range clients {
			err := client.WriteJSON(response)
			if err != nil {
				log.Printf("Error broadcasting to client: %v", err)
				_ = client.Close()
				s.clientsMux.Lock()
				delete(s.clients, client)
				s.clientsMux.Unlock()
			}
		}
	}
}

func websocketOriginAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		// Native clients such as the iOS app do not send a browser Origin header.
		return true
	}

	normalized, ok := normalizeOrigin(origin)
	if !ok {
		return false
	}
	originURL, _ := url.Parse(normalized)
	if strings.EqualFold(originURL.Host, r.Host) {
		return true
	}

	for _, allowed := range strings.Split(os.Getenv("TALE_ALLOWED_ORIGINS"), ",") {
		if candidate, valid := normalizeOrigin(allowed); valid && candidate == normalized {
			return true
		}
	}
	return false
}

func normalizeOrigin(value string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	if parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), true
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
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
