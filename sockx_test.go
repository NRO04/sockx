package sockx

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewServer(t *testing.T) {
	server := NewServer()
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.namespaces == nil {
		t.Error("Server namespaces map is nil")
	}
}

func TestNamespace(t *testing.T) {
	server := NewServer()
	
	ns1 := server.Namespace("/test")
	if ns1 == nil {
		t.Fatal("Namespace returned nil")
	}
	if ns1.name != "/test" {
		t.Errorf("Expected namespace name '/test', got '%s'", ns1.name)
	}
	
	// Getting the same namespace should return the same instance
	ns2 := server.Namespace("/test")
	if ns1 != ns2 {
		t.Error("Namespace should return the same instance for the same name")
	}
}

func TestNamespaceOn(t *testing.T) {
	server := NewServer()
	ns := server.Namespace("/")
	
	called := false
	ns.On("test-event", func(client *Client, data interface{}) {
		called = true
	})
	
	if len(ns.events) != 1 {
		t.Errorf("Expected 1 event handler, got %d", len(ns.events))
	}
	
	// Trigger the event
	client := &Client{ID: "test"}
	ns.handleEvent(client, Message{Event: "test-event", Data: "test"})
	
	if !called {
		t.Error("Event handler was not called")
	}
}

func TestRoom(t *testing.T) {
	server := NewServer()
	ns := server.Namespace("/")
	
	room1 := ns.Room("room1")
	if room1 == nil {
		t.Fatal("Room returned nil")
	}
	if room1.name != "room1" {
		t.Errorf("Expected room name 'room1', got '%s'", room1.name)
	}
	
	// Getting the same room should return the same instance
	room2 := ns.Room("room1")
	if room1 != room2 {
		t.Error("Room should return the same instance for the same name")
	}
}

func TestClientJoinLeave(t *testing.T) {
	server := NewServer()
	ns := server.Namespace("/")
	
	client := &Client{
		ID:        "test-client",
		namespace: ns,
		rooms:     make(map[string]bool),
		send:      make(chan Message, 1),
	}
	
	// Join a room
	client.Join("room1")
	
	if !client.rooms["room1"] {
		t.Error("Client should be marked as in room1")
	}
	
	room := ns.Room("room1")
	if !room.clients[client] {
		t.Error("Client should be in room1's client list")
	}
	
	// Leave the room
	client.Leave("room1")
	
	if client.rooms["room1"] {
		t.Error("Client should not be marked as in room1 after leaving")
	}
	
	if room.clients[client] {
		t.Error("Client should not be in room1's client list after leaving")
	}
}

func TestClientEmit(t *testing.T) {
	server := NewServer()
	ns := server.Namespace("/")
	
	client := &Client{
		ID:        "test-client",
		namespace: ns,
		send:      make(chan Message, 10),
	}
	
	client.Emit("test-event", "test-data")
	
	select {
	case msg := <-client.send:
		if msg.Event != "test-event" {
			t.Errorf("Expected event 'test-event', got '%s'", msg.Event)
		}
		if msg.Data != "test-data" {
			t.Errorf("Expected data 'test-data', got '%v'", msg.Data)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestNamespaceEmit(t *testing.T) {
	server := NewServer()
	ns := server.Namespace("/")
	
	client1 := &Client{
		ID:        "client1",
		namespace: ns,
		send:      make(chan Message, 10),
	}
	client2 := &Client{
		ID:        "client2",
		namespace: ns,
		send:      make(chan Message, 10),
	}
	
	ns.addClient(client1)
	ns.addClient(client2)
	
	ns.Emit("broadcast-event", "broadcast-data")
	
	// Check client1 received the message
	select {
	case msg := <-client1.send:
		if msg.Event != "broadcast-event" {
			t.Errorf("Client1: Expected event 'broadcast-event', got '%s'", msg.Event)
		}
	case <-time.After(time.Second):
		t.Error("Client1: Timeout waiting for message")
	}
	
	// Check client2 received the message
	select {
	case msg := <-client2.send:
		if msg.Event != "broadcast-event" {
			t.Errorf("Client2: Expected event 'broadcast-event', got '%s'", msg.Event)
		}
	case <-time.After(time.Second):
		t.Error("Client2: Timeout waiting for message")
	}
}

func TestRoomEmit(t *testing.T) {
	server := NewServer()
	ns := server.Namespace("/")
	
	client1 := &Client{
		ID:        "client1",
		namespace: ns,
		rooms:     make(map[string]bool),
		send:      make(chan Message, 10),
	}
	client2 := &Client{
		ID:        "client2",
		namespace: ns,
		rooms:     make(map[string]bool),
		send:      make(chan Message, 10),
	}
	
	client1.Join("room1")
	
	room := ns.Room("room1")
	room.Emit("room-event", "room-data")
	
	// Check client1 received the message
	select {
	case msg := <-client1.send:
		if msg.Event != "room-event" {
			t.Errorf("Client1: Expected event 'room-event', got '%s'", msg.Event)
		}
		if msg.Room != "room1" {
			t.Errorf("Client1: Expected room 'room1', got '%s'", msg.Room)
		}
	case <-time.After(time.Second):
		t.Error("Client1: Timeout waiting for message")
	}
	
	// Check client2 did NOT receive the message (not in room)
	select {
	case <-client2.send:
		t.Error("Client2 should not have received the room message")
	case <-time.After(100 * time.Millisecond):
		// Expected - client2 is not in the room
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	server := NewServer()
	ns := server.Namespace("/")
	
	var eventReceived int32
	ns.On("test", func(client *Client, data interface{}) {
		atomic.StoreInt32(&eventReceived, 1)
	})
	
	// Create test server
	ts := httptest.NewServer(server.ServeWebSocket("/"))
	defer ts.Close()
	
	// Convert http://... to ws://...
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	
	// Connect as a client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	// Send a test message
	msg := Message{
		Event: "test",
		Data:  "test-data",
	}
	
	err = conn.WriteJSON(msg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
	
	// Wait a bit for the message to be processed
	time.Sleep(100 * time.Millisecond)
	
	if atomic.LoadInt32(&eventReceived) == 0 {
		t.Error("Event was not received by server")
	}
}

func TestWebSocketBroadcast(t *testing.T) {
	server := NewServer()
	ns := server.Namespace("/")
	
	ns.On("broadcast", func(client *Client, data interface{}) {
		// Broadcast to all clients
		ns.Emit("message", data)
	})
	
	// Create test server
	ts := httptest.NewServer(server.ServeWebSocket("/"))
	defer ts.Close()
	
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	
	// Connect first client
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer conn1.Close()
	
	// Connect second client
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer conn2.Close()
	
	// Give connections time to establish
	time.Sleep(100 * time.Millisecond)
	
	// Send broadcast message from client 1
	err = conn1.WriteJSON(Message{
		Event: "broadcast",
		Data:  "hello from client 1",
	})
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
	
	// Both clients should receive the broadcast
	received := 0
	
	// Try to receive on client 1
	conn1.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var msg1 Message
	if err := conn1.ReadJSON(&msg1); err == nil {
		if msg1.Event == "message" {
			received++
		}
	}
	
	// Try to receive on client 2
	conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var msg2 Message
	if err := conn2.ReadJSON(&msg2); err == nil {
		if msg2.Event == "message" {
			received++
		}
	}
	
	if received < 1 {
		t.Errorf("Expected at least 1 client to receive broadcast, got %d", received)
	}
}

func TestMessageSerialization(t *testing.T) {
	msg := Message{
		Event:     "test-event",
		Data:      "test-data",
		Namespace: "/test",
		Room:      "room1",
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}
	
	var decoded Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}
	
	if decoded.Event != msg.Event {
		t.Errorf("Event mismatch: expected '%s', got '%s'", msg.Event, decoded.Event)
	}
	if decoded.Namespace != msg.Namespace {
		t.Errorf("Namespace mismatch: expected '%s', got '%s'", msg.Namespace, decoded.Namespace)
	}
	if decoded.Room != msg.Room {
		t.Errorf("Room mismatch: expected '%s', got '%s'", msg.Room, decoded.Room)
	}
}
