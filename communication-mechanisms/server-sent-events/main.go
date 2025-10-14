package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", serveIndex)
	router.HandleFunc("/deployment/{deploymentId}", serveDeployment)
	router.HandleFunc("/start", startDeployment)
	router.HandleFunc("/events/{deploymentId}", sseHandler)
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	log.Println("Server started at :8080")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving index.html")
	http.ServeFile(w, r, "./static/index.html")
	log.Println("Served index.html")
}

func serveDeployment(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/deployment.html")
}

func startDeployment(w http.ResponseWriter, r *http.Request) {
	// Create or truncate messages.txt
	uuid, _ := uuid.NewUUID()
	uuidStr := uuid.String()
	fmt.Println("Starting deployment with UUID:", uuidStr)
	f, err := os.Create(fmt.Sprintf("deployments/%s.txt", uuid))
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
			appendMessage(msg, uuidStr)
		}
		appendMessage("Deployment finished!\n", uuidStr)
	}()
	http.Redirect(w, r, fmt.Sprintf("/deployment/%s", uuidStr), http.StatusSeeOther)
}

func appendMessage(msg string, fileName string) {
	f, err := os.OpenFile(fmt.Sprintf("deployments/%s.txt", fileName), os.O_APPEND|os.O_WRONLY, 0644)
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

	deploymentId := mux.Vars(r)["deploymentId"]
	if deploymentId == "" {
		http.Error(w, "Deployment ID is required", http.StatusBadRequest)
		return
	}

	filePath := fmt.Sprintf("deployments/%s.txt", deploymentId)

	w.Header().Set("Content-Type", "text/event-stream")

	lastLine := 0
	for {
		file, err := os.Open(filePath)
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
		time.Sleep(1 * time.Second)
		fmt.Println("Checking for new messages. Last line:", lastLine)
	}
}
