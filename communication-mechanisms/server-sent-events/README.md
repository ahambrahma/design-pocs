# Server-Sent Events (SSE) Deployment Demo

This project demonstrates a basic deployment progress tracker using Go and Server-Sent Events (SSE).  
Each deployment creates a unique file and streams its progress to the browser in real time.

## Features

- **Start a deployment:** Each click creates a new deployment with a unique ID (UUID).
- **Progress streaming:** The server streams deployment progress using SSE.
- **Multiple deployments:** Each deployment has its own progress file and URL.
- **Simple HTML frontend.**

## How It Works

1. **Homepage (`/`):**  
   Shows a "Deployment" button.

2. **Start Deployment:**  
   Clicking the button sends a POST request to `/start`.  
   The server creates a new deployment file under `deployments/` and redirects to `/deployment/{deploymentId}`.

3. **Deployment Progress (`/deployment/{deploymentId}`):**  
   The browser loads the deployment page and connects to `/events/{deploymentId}` to receive live progress updates.

## Endpoints

- `/` — Homepage with deployment button.
- `/start` — Starts a new deployment and redirects to its progress page.
- `/deployment/{deploymentId}` — Shows progress for a specific deployment.
- `/events/{deploymentId}` — SSE endpoint streaming progress for the given deployment.

## File Structure

- `main.go` — Go server with SSE and deployment logic.
- `static/index.html` — Homepage.
- `static/deployment.html` — Deployment progress page.
- `deployments/` — Folder containing per-deployment progress files.
- `README.md` — Project summary.

## How to Run

1. **Start the server:**
   ```sh
   go run main.go
   ```

2. **Open your browser:**  
   Visit [http://localhost:8080](http://localhost:8080)

3. **Start a deployment:**  
   Click the "Deployment" button.  
   You will be redirected to `/deployment/{deploymentId}` and see live progress.

---

```<!-- filepath: /Users/shubhamsharma/projects/go/design-pocs/communication-mechanisms/server-sent-events/README.md -->

# Server-Sent Events (SSE) Deployment Demo

This project demonstrates a basic deployment progress tracker using Go and Server-Sent Events (SSE).  
Each deployment creates a unique file and streams its progress to the browser in real time.

## Features

- **Start a deployment:** Each click creates a new deployment with a unique ID (UUID).
- **Progress streaming:** The server streams deployment progress using SSE.
- **Multiple deployments:** Each deployment has its own progress file and URL.
- **Simple HTML frontend.**

## How It Works

1. **Homepage (`/`):**  
   Shows a "Deployment" button.

2. **Start Deployment:**  
   Clicking the button sends a POST request to `/start`.  
   The server creates a new deployment file under `deployments/` and redirects to `/deployment/{deploymentId}`.

3. **Deployment Progress (`/deployment/{deploymentId}`):**  
   The browser loads the deployment page and connects to `/events/{deploymentId}` to receive live progress updates.

## Endpoints

- `/` — Homepage with deployment button.
- `/start` — Starts a new deployment and redirects to its progress page.
- `/deployment/{deploymentId}` — Shows progress for a specific deployment.
- `/events/{deploymentId}` — SSE endpoint streaming progress for the given deployment.

## File Structure

- `main.go` — Go server with SSE and deployment logic.
- `static/index.html` — Homepage.
- `static/deployment.html` — Deployment progress page.
- `deployments/` — Folder containing per-deployment progress files.
- `README.md` — Project summary.

## How to Run

1. **Start the server:**
   ```sh
   go run main.go
   ```

2. **Open your browser:**  
   Visit [http://localhost:8080](http://localhost:8080)

3. **Start a deployment:**  
   Click the "Deployment" button.  
   You will be redirected to `/deployment/{deploymentId}` and see live progress.

## Key Concepts

- **Server-Sent Events (SSE):**  
  Allows the server to push updates to the browser over a single HTTP connection.
- **Go HTTP Server:**  
  Streams lines from a file to the client using the SSE protocol.

## Notes

- No CSS or frameworks are used; the UI is intentionally minimal.
- Messages are stored in `messages.txt` and streamed as they



# PS: Entire summary was generated using GPT, so there can be mistakes