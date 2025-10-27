package main

import (
	"log"
	"net/http"

	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

func main() {
	// Create Socket.IO server with permissive transport options
	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: func(r *http.Request) bool {
					log.Printf("Polling CheckOrigin called from: %s", r.RemoteAddr)
					return true
				},
			},
			&websocket.Transport{
				CheckOrigin: func(r *http.Request) bool {
					log.Printf("WebSocket CheckOrigin called from: %s", r.RemoteAddr)
					return true
				},
			},
		},
	})

	const chatRoom = "chat"

	// Handle connection
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		log.Printf("✅ Client connected: ID=%s", s.ID())

		// Join the chat room
		s.Join(chatRoom)
		log.Printf("Client %s joined room: %s", s.ID(), chatRoom)

		return nil
	})

	// Handle message events
	server.OnEvent("/", "message", func(s socketio.Conn, msg Message) {
		log.Printf("📨 Message from %s (ID: %s): %s", msg.Username, s.ID(), msg.Message)

		// Broadcast to all clients in the room
		server.BroadcastToRoom("", chatRoom, "message", msg)
		log.Printf("📤 Broadcasted message to room: %s", chatRoom)
	})

	// Handle errors
	server.OnError("/", func(s socketio.Conn, e error) {
		log.Printf("❌ Socket.IO error for client %s: %v", s.ID(), e)
	})

	// Handle disconnection
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Printf("🔌 Client disconnected: ID=%s, Reason=%s", s.ID(), reason)
	})

	// Start Socket.IO server
	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("Socket.IO server error: %v\n", err)
		}
	}()
	defer server.Close()

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Socket.IO endpoint
	mux.Handle("/socket.io/", server)

	// Static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	// Add logging middleware
	loggedMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📍 %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Add CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		mux.ServeHTTP(w, r)
	})

	port := ":8080"
	log.Printf("🚀 Server starting on http://localhost%s", port)
	log.Printf("📁 Serving static files from ./static/")
	log.Printf("🔌 Socket.IO endpoint: http://localhost%s/socket.io/", port)

	if err := http.ListenAndServe(port, loggedMux); err != nil {
		log.Fatal("❌ Server error: ", err)
	}
}
