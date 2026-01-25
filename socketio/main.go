package main

import (
	"fmt"
	"net"
	"syscall"
)

/***
This example demonstrates how to create a simple TCP server using raw syscalls in Go.
It covers the essential steps: creating a socket, binding it to an address and port,
listening for incoming connections, and accepting those connections.

Note: This example is for educational purposes only. In production code, it's recommended
to use the higher-level "net" package for better safety and convenience.
***/

func main() {
	// 1. SOCKET: Create the phone (Get a File Descriptor)
	// AF_INET = IPv4, SOCK_STREAM = TCP, 0 = default protocol
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	defer syscall.Close(fd)
	fmt.Printf("Socket created. FD: %d\n", fd)

	// 2. BIND: Assign a number to the phone
	// We bind to port 8080 on localhost
	sa := &syscall.SockaddrInet4{Port: 8080}
	copy(sa.Addr[:], net.ParseIP("127.0.0.1").To4())

	err := syscall.Bind(fd, sa)
	if err != nil {
		panic(err)
	}
	fmt.Println("Bind successful. We own 127.0.0.1:8080")

	// 3. LISTEN: Turn on the ringer (Wait for calls)
	// 128 is the backlog (queue size for pending connections)
	err = syscall.Listen(fd, 128)
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening for incoming connections...")

	for {
		// 4. ACCEPT: Answer a call
		// BLOCKS until a client connects.
		// Returns a NEW file descriptor (clientFD) for this specific conversation.
		clientFD, _, err := syscall.Accept(fd)
		if err != nil {
			continue
		}

		fmt.Printf("Accepted connection! New FD: %d\n", clientFD)

		// Simple write to the new FD (standard I/O)
		msg := []byte("Hello from raw syscalls!\n")
		syscall.Write(clientFD, msg)

		// Use lsof -i:8080 to see the file descriptors in action.

		// Hang up on this client (close THEIR fd, not the main one)
		syscall.Close(clientFD)
	}
}
