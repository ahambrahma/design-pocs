# Server-Sent Events (SSE) Deployment Demo

This is a simple Go prototype demonstrating **Server-Sent Events (SSE)** using a deployment simulation. It streams deployment progress messages from the server to the browser in real time.

## How It Works

- **Homepage (`/`)**: Shows a "Deployment" button.
- **On Button Click**: Starts a simulated deployment, creates/clears `messages.txt`, and appends progress messages.
- **Deployment Page (`/deployment`)**: Displays messages as they arrive using SSE.
- **SSE Endpoint (`/events`)**: Streams new lines from `messages.txt` to the browser.

## File Structure

```
communication-mechanisms/
├── main.go
├── messages.txt
├── static/
│   ├── index.html
│   └── deployment.html
└── README.md
```

## How to Run

1. **Start the server:**
   ```sh
   go run main.go
   ```

2. **Open your browser:**  
   Visit [http://localhost:8080](http://localhost:8080)

3. **Click "Deployment":**  
   You will be redirected to the deployment page and see progress messages appear one by one.

## Key Concepts

- **Server-Sent Events (SSE):**  
  Allows the server to push updates to the browser over a single HTTP connection.
- **Go HTTP Server:**  
  Streams lines from a file to the client using the SSE protocol.

## Notes

- No CSS or frameworks are used; the UI is intentionally minimal.
- Messages are stored in `messages.txt` and streamed as they are appended.

---
```<!-- filepath: /Users/shubhamsharma/projects/go/design-pocs/communication-mechanisms/README.md -->
# Server-Sent Events (SSE) Deployment Demo

This is a simple Go prototype demonstrating **Server-Sent Events (SSE)** using a deployment simulation. It streams deployment progress messages from the server to the browser in real time.

## How It Works

- **Homepage (`/`)**: Shows a "Deployment" button.
- **On Button Click**: Starts a simulated deployment, creates/clears `messages.txt`, and appends progress messages.
- **Deployment Page (`/deployment`)**: Displays messages as they arrive using SSE.
- **SSE Endpoint (`/events`)**: Streams new lines from `messages.txt` to the browser.

## File Structure

```
communication-mechanisms/
├── main.go
├── messages.txt
├── static/
│   ├── index.html
│   └── deployment.html
└── README.md
```

## How to Run

1. **Start the server:**
   ```sh
   go run main.go
   ```

2. **Open your browser:**  
   Visit [http://localhost:8080](http://localhost:8080)

3. **Click "Deployment":**  
   You will be redirected to the deployment page and see progress messages appear one by one.

## Key Concepts

- **Server-Sent Events (SSE):**  
  Allows the server to push updates to the browser over a single HTTP connection.
- **Go HTTP Server:**  
  Streams lines from a file to the client using the SSE protocol.

## Notes

- No CSS or frameworks are used; the UI is intentionally minimal.
- Messages are stored in `messages.txt` and streamed as they



# PS: Entire summary was generated using GPT, so there can be mistakes