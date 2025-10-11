// ...existing code...
package main

import (
	"log"
	"net/http"

	socketio "github.com/googollee/go-socket.io"
)

func main() {
	server := socketio.NewServer(nil)

	// NOTE: SetAllowRequest is not available in this version of the library.
	// If you need to allow cross-origin requests, wrap the handler with CORS middleware
	// (see corsWrapper below).

	server.OnConnect("/", func(s socketio.Conn) error {
		log.Println("connected:", s.ID())
		return nil
	})

	server.OnEvent("/", "echo", func(s socketio.Conn, msg string) {
		log.Println("received:", msg)
		s.Emit("echo", msg)
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("disconnected:", reason)
	})

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	// Wrap the socket.io handler with simple CORS support (optional)
	http.Handle("/socket.io/", corsWrapper(server))
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Println("Serving at :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func corsWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// allow any origin (use more restrictive policy in production)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// ...existing code...
