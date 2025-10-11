package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func main() {
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/deployment", serveDeployment)
	http.HandleFunc("/start", startDeployment)
	http.HandleFunc("/events", sseHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

func serveDeployment(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/deployment.html")
}

func startDeployment(w http.ResponseWriter, r *http.Request) {
	// Create or truncate messages.txt
	f, err := os.Create("messages.txt")
	if err != nil {
		http.Error(w, "Unable to start deployment", 500)
		return
	}
	defer f.Close()
	f.WriteString("Deployment started...\n")
	go func() {
		for i := 1; i <= 1000; i++ {
			time.Sleep(100 * time.Millisecond)
			msg := fmt.Sprintf("Step %d completed: %d\n", i, rand.Intn(1000))
			appendMessage(msg)
		}
		appendMessage("Deployment finished!\n")
	}()
	http.Redirect(w, r, "/deployment", http.StatusSeeOther)
}

func appendMessage(msg string) {
	f, err := os.OpenFile("messages.txt", os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(msg)
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	/**
	These are the 3 headers which are required for Server-Sent Events (SSE) to work properly:
	1. Content-Type: This should be set to text/event-stream to indicate that the response is an event stream.

	These 2 are kind of optional:
	2. Cache-Control: This should be set to no-cache to prevent the browser from caching the event stream.
	3. Connection: This should be set to keep-alive to keep the connection open for continuous data flow.
	**/
	w.Header().Set("Content-Type", "text/event-stream")

	lastLine := 0
	for {
		file, err := os.Open("messages.txt")
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		scanner := bufio.NewScanner(file)
		lines := []string{}
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		file.Close()

		for i := lastLine; i < len(lines); i++ {
			fmt.Fprintf(w, "data: %s\n\n", lines[i])
			flusher, ok := w.(http.Flusher)
			if ok {
				flusher.Flush()
			}
			lastLine++
		}
		time.Sleep(5 * time.Second)
		fmt.Println("Checking for new messages. Last line:", lastLine)
	}
}
