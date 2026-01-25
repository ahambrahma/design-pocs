package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type Server struct {
	addr  string
	ln    net.Listener
	wg    sync.WaitGroup
	mu    sync.Mutex            // Protects the conns map
	conns map[net.Conn]struct{} // Set of active connections
}

func NewServer(addr string) *Server {
	return &Server{
		addr:  addr,
		conns: make(map[net.Conn]struct{}),
	}
}

func (s *Server) Start() error {
	var err error
	s.ln, err = net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	log.Printf("Server listening on %s", s.addr)

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("Accept error: %v", err)
			continue
		}

		s.trackConn(conn, true)
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *Server) trackConn(c net.Conn, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if add {
		s.conns[c] = struct{}{}
	} else {
		delete(s.conns, c)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer s.trackConn(conn, false) // Remove from map when done
	defer conn.Close()

	log.Printf("Client connected: %s", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF && !errors.Is(err, net.ErrClosed) {
				log.Printf("Read error: %v", err)
			}
			return
		}

		// 'line' includes the newline char, so trim it
		msg := strings.TrimSpace(line)
		if msg == "exit" {
			return
		}

		// Echo back (Write is also efficient, but we can just use conn.Write)
		if _, err := conn.Write([]byte(line)); err != nil {
			return
		}
	}
}

func (s *Server) Stop() {
	// 1. Close listener to stop accepting new connections
	if s.ln != nil {
		s.ln.Close()
	}

	// 2. Close all active connections to unblock Read() calls
	s.mu.Lock()
	for conn := range s.conns {
		conn.Close()
	}
	s.mu.Unlock()

	// 3. Wait for all goroutines to finish
	s.wg.Wait()
	log.Println("Server gracefully stopped")
}

func ListenMultiThreaded() {
	server := NewServer(":8080")
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	server.Stop()
}
