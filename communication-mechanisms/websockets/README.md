# WebSockets Demo with Go and Socket.IO

This project demonstrates a basic real-time messaging app using Go as the backend and Socket.IO for WebSocket communication.

## Features

- Real-time communication between server and browser clients using Socket.IO.
- Echo messages: Clients can send messages and receive them back.
- Server-initiated messages: The server can broadcast messages to all connected clients.
- Simple HTML/JavaScript frontend.

## How to Run

1. **Start the server:**
   ```sh
   go run main.go
   ```

2. **Open the client:**
   Visit [http://localhost:8080](http://localhost:8080) in your browser.

3. **Send messages:**
   - Type a message and click "Send" to see it echoed back.
   - The server periodically sends messages to all clients.
   - You can trigger a broadcast from the server using:
     ```sh
     curl "http://localhost:8080/broadcast?msg=hello"
     ```

## File Structure

- `main.go` — Go server with Socket.IO integration.
- `static/index.html` — Simple web client.
- `README.md` — Project summary.

## Requirements

- Go 1.18