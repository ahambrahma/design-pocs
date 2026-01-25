package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func Listen() {
	// 1. Listen on a TCP port
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Server listening on :8080 (Single Threaded)...")

	// 2. Infinite loop to accept connections
	for {
		// This call BLOCKS until a client connects
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err)
			continue
		}

		// 3. Handle connection DIRECTLY (Blocking the main loop)
		// No 'go handleConnection(conn)' here!
		fmt.Printf("Client connected: %s\n", conn.RemoteAddr())

		handleConnection(conn)

		fmt.Printf("Client disconnected: %s\n", conn.RemoteAddr())
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close() // Ensure connection is closed when function exits

	reader := bufio.NewReader(conn)

	for {
		// Read data until newline
		message, err := reader.ReadString('\n')
		if err != nil {
			// EOF or error means client disconnected
			return
		}

		// Process message
		message = strings.TrimSpace(message)
		fmt.Printf("Received: %s\n", message)

		// Respond (Echo)
		response := fmt.Sprintf("Echo: %s\n", message)
		conn.Write([]byte(response))

		if message == "QUIT" {
			return
		}
	}
}
