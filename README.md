# Sockx

A Go WebSocket server library similar to Socket.IO, built on top of `net/http` and Gorilla WebSocket. Sockx provides an idiomatic, concurrent-safe API for building real-time applications with support for namespaces, rooms, custom events, and broadcasts.

## Features

- **WebSocket-based**: Built on standard `net/http` and Gorilla WebSocket
- **Namespaces**: Organize connections into separate channels
- **Rooms**: Group clients within namespaces for targeted broadcasting
- **Custom Events**: Register and emit custom events with typed handlers
- **Broadcasts**: Send messages to all clients in a namespace or room
- **Concurrent-safe**: All operations are protected with mutexes for safe concurrent access
- **Clean API**: Simple, intuitive On/Emit pattern similar to Socket.IO
- **Minimal Dependencies**: Only depends on Gorilla WebSocket

## Installation

```bash
go get github.com/NRO04/sockx
```

## Quick Start

### Server Setup

```go
package main

import (
    "log"
    "net/http"
    "github.com/NRO04/sockx"
)

func main() {
    // Create a new sockx server
    server := sockx.NewServer()
    
    // Get the default namespace
    ns := server.Namespace("/")
    
    // Register event handlers
    ns.On("message", func(client *sockx.Client, data interface{}) {
        log.Printf("Received message: %v", data)
        
        // Broadcast to all clients
        ns.Emit("message", data)
    })
    
    // Setup WebSocket endpoint
    http.HandleFunc("/ws", server.ServeWebSocket("/"))
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Client (JavaScript)

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function() {
    console.log('Connected');
    
    // Send a message
    ws.send(JSON.stringify({
        event: 'message',
        data: 'Hello, server!'
    }));
};

ws.onmessage = function(event) {
    const msg = JSON.parse(event.data);
    console.log('Event:', msg.event, 'Data:', msg.data);
};
```

## Core Concepts

### Server

The `Server` is the main entry point. It manages namespaces and handles WebSocket upgrades.

```go
server := sockx.NewServer()
```

### Namespaces

Namespaces allow you to separate concerns and create different channels of communication.

```go
// Get or create a namespace
ns := server.Namespace("/chat")

// Register event handlers
ns.On("message", func(client *sockx.Client, data interface{}) {
    // Handle message event
})

// Broadcast to all clients in the namespace
ns.Emit("notification", "Server message")
```

### Rooms

Rooms let you group clients within a namespace for targeted broadcasts.

```go
// Client joins a room
client.Join("room1")

// Broadcast to all clients in a room
room := ns.Room("room1")
room.Emit("room-message", "Message for room1")

// Client leaves a room
client.Leave("room1")
```

### Events

Events are the core communication mechanism. Register handlers with `On()` and trigger with `Emit()`.

```go
// Register an event handler
ns.On("join-room", func(client *sockx.Client, data interface{}) {
    roomName := data.(string)
    client.Join(roomName)
    
    // Notify room members
    room := ns.Room(roomName)
    room.Emit("user-joined", client.ID)
})

// Emit to a specific client
client.Emit("welcome", "Welcome to the server!")

// Emit to all clients in namespace
ns.Emit("announcement", "Server announcement")

// Emit to all clients in a room
room.Emit("room-update", "Room notification")
```

### Message Format

All messages follow this JSON structure:

```json
{
    "event": "event-name",
    "data": "any-json-data",
    "namespace": "/optional-namespace",
    "room": "optional-room"
}
```

## Examples

### Chat Application

See the full example in `examples/chat/main.go` for a complete chat application with:
- Real-time messaging
- Room-based conversations
- User join/leave notifications
- HTML/JavaScript client

Run the example:

```bash
cd examples/chat
go run main.go
```

Then open http://localhost:8080 in your browser.

### Room-based Broadcasting

```go
server := sockx.NewServer()
ns := server.Namespace("/")

ns.On("join-room", func(client *sockx.Client, data interface{}) {
    roomName := data.(string)
    client.Join(roomName)
    
    room := ns.Room(roomName)
    room.Emit("user-joined", map[string]interface{}{
        "userId": client.ID,
        "room": roomName,
    })
})

ns.On("room-message", func(client *sockx.Client, data interface{}) {
    msg := data.(map[string]interface{})
    roomName := msg["room"].(string)
    text := msg["text"]
    
    room := ns.Room(roomName)
    room.Emit("message", map[string]interface{}{
        "from": client.ID,
        "text": text,
    })
})
```

### Multiple Namespaces

```go
server := sockx.NewServer()

// Chat namespace
chat := server.Namespace("/chat")
chat.On("message", func(client *sockx.Client, data interface{}) {
    chat.Emit("message", data)
})

// Notifications namespace
notifications := server.Namespace("/notifications")
notifications.On("subscribe", func(client *sockx.Client, data interface{}) {
    // Handle subscription
})

http.HandleFunc("/ws/chat", server.ServeWebSocket("/chat"))
http.HandleFunc("/ws/notifications", server.ServeWebSocket("/notifications"))
```

## API Reference

### Server

- `NewServer() *Server` - Create a new server
- `Namespace(name string) *Namespace` - Get or create a namespace
- `ServeWebSocket(namespaceName string) http.HandlerFunc` - HTTP handler for WebSocket upgrade

### Namespace

- `On(event string, handler EventHandler)` - Register an event handler
- `Emit(event string, data interface{})` - Broadcast to all clients
- `Room(name string) *Room` - Get or create a room

### Room

- `Emit(event string, data interface{})` - Broadcast to all clients in the room

### Client

- `Join(roomName string)` - Join a room
- `Leave(roomName string)` - Leave a room
- `Emit(event string, data interface{})` - Send a message to this client
- `ID` - Unique client identifier

## Thread Safety

All operations in sockx are concurrent-safe. The library uses `sync.RWMutex` to protect shared state, allowing safe concurrent access from multiple goroutines.

## Testing

Run the test suite:

```bash
go test -v
```

Run tests with race detection:

```bash
go test -race -v
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.