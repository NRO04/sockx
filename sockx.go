// Package sockx provides a WebSocket server similar to Socket.IO with support for
// namespaces, rooms, custom events, and broadcasts.
package sockx

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// EventHandler is a function type for handling events.
type EventHandler func(*Client, interface{})

// Message represents a sockx message.
type Message struct {
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
	Namespace string      `json:"namespace,omitempty"`
	Room      string      `json:"room,omitempty"`
}

// Server represents the main sockx server.
type Server struct {
	namespaces map[string]*Namespace
	upgrader   websocket.Upgrader
	mu         sync.RWMutex
}

// Namespace represents a namespace that groups clients.
type Namespace struct {
	name    string
	clients map[*Client]bool
	rooms   map[string]*Room
	events  map[string]EventHandler
	server  *Server
	mu      sync.RWMutex
}

// Room represents a room within a namespace.
type Room struct {
	name      string
	clients   map[*Client]bool
	namespace *Namespace
	mu        sync.RWMutex
}

// Client represents a connected client.
type Client struct {
	ID        string
	conn      *websocket.Conn
	server    *Server
	namespace *Namespace
	rooms     map[string]bool
	send      chan Message
	mu        sync.RWMutex
}

// NewServer creates a new sockx server.
func NewServer() *Server {
	return &Server{
		namespaces: make(map[string]*Namespace),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins by default
			},
		},
	}
}

// Namespace retrieves or creates a namespace.
func (s *Server) Namespace(name string) *Namespace {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ns, exists := s.namespaces[name]; exists {
		return ns
	}

	ns := &Namespace{
		name:    name,
		clients: make(map[*Client]bool),
		rooms:   make(map[string]*Room),
		events:  make(map[string]EventHandler),
		server:  s,
	}
	s.namespaces[name] = ns
	return ns
}

// On registers an event handler for the namespace.
func (ns *Namespace) On(event string, handler EventHandler) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.events[event] = handler
}

// Emit sends an event to all clients in the namespace.
func (ns *Namespace) Emit(event string, data interface{}) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	msg := Message{
		Event:     event,
		Data:      data,
		Namespace: ns.name,
	}

	for client := range ns.clients {
		select {
		case client.send <- msg:
		default:
			// Client send channel is full, skip
		}
	}
}

// Room retrieves or creates a room in the namespace.
func (ns *Namespace) Room(name string) *Room {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if room, exists := ns.rooms[name]; exists {
		return room
	}

	room := &Room{
		name:      name,
		clients:   make(map[*Client]bool),
		namespace: ns,
	}
	ns.rooms[name] = room
	return room
}

// addClient adds a client to the namespace.
func (ns *Namespace) addClient(client *Client) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.clients[client] = true
}

// removeClient removes a client from the namespace.
func (ns *Namespace) removeClient(client *Client) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	delete(ns.clients, client)
}

// handleEvent processes an event for the namespace.
func (ns *Namespace) handleEvent(client *Client, msg Message) {
	ns.mu.RLock()
	handler, exists := ns.events[msg.Event]
	ns.mu.RUnlock()

	if exists {
		handler(client, msg.Data)
	}
}

// Emit sends an event to all clients in the room.
func (r *Room) Emit(event string, data interface{}) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	msg := Message{
		Event:     event,
		Data:      data,
		Namespace: r.namespace.name,
		Room:      r.name,
	}

	for client := range r.clients {
		select {
		case client.send <- msg:
		default:
			// Client send channel is full, skip
		}
	}
}

// addClient adds a client to the room.
func (r *Room) addClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[client] = true
}

// removeClient removes a client from the room.
func (r *Room) removeClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, client)
}

// Join adds the client to a room.
func (c *Client) Join(roomName string) {
	c.mu.Lock()
	c.rooms[roomName] = true
	c.mu.Unlock()

	room := c.namespace.Room(roomName)
	room.addClient(c)
}

// Leave removes the client from a room.
func (c *Client) Leave(roomName string) {
	c.mu.Lock()
	delete(c.rooms, roomName)
	c.mu.Unlock()

	c.namespace.mu.RLock()
	room, exists := c.namespace.rooms[roomName]
	c.namespace.mu.RUnlock()

	if exists {
		room.removeClient(c)
	}
}

// Emit sends an event to the client.
func (c *Client) Emit(event string, data interface{}) {
	msg := Message{
		Event:     event,
		Data:      data,
		Namespace: c.namespace.name,
	}

	select {
	case c.send <- msg:
	default:
		// Client send channel is full, skip
	}
}

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump() {
	defer func() {
		c.namespace.removeClient(c)

		// Leave all rooms
		c.mu.RLock()
		rooms := make([]string, 0, len(c.rooms))
		for room := range c.rooms {
			rooms = append(rooms, room)
		}
		c.mu.RUnlock()

		for _, room := range rooms {
			c.Leave(room)
		}

		c.conn.Close()
	}()

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		c.namespace.handleEvent(c, msg)
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	defer c.conn.Close()

	for msg := range c.send {
		err := c.conn.WriteJSON(msg)
		if err != nil {
			break
		}
	}
}

// ServeWebSocket handles WebSocket upgrade and client connection.
func (s *Server) ServeWebSocket(namespaceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		namespace := s.Namespace(namespaceName)

		client := &Client{
			ID:        generateID(),
			conn:      conn,
			server:    s,
			namespace: namespace,
			rooms:     make(map[string]bool),
			send:      make(chan Message, 256),
		}

		namespace.addClient(client)

		// Start client goroutines
		go client.writePump()
		go client.readPump()
	}
}

// generateID generates a unique client ID.
func generateID() string {
	// Simple ID generation - in production, use UUID or similar
	return randomString(16)
}

// randomString generates a random string of specified length.
func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}
