package main

import (
	"fmt"
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
		log.Printf("Client %s sent message: %v", client.ID, data)

		// Broadcast to all clients in the namespace
		ns.Emit("message", map[string]interface{}{
			"from": client.ID,
			"text": data,
		})
	})

	ns.On("join-room", func(client *sockx.Client, data interface{}) {
		roomName, ok := data.(string)
		if !ok {
			return
		}

		log.Printf("Client %s joining room: %s", client.ID, roomName)
		client.Join(roomName)

		// Broadcast to the room
		room := ns.Room(roomName)
		room.Emit("user-joined", map[string]interface{}{
			"user": client.ID,
			"room": roomName,
		})
	})

	ns.On("room-message", func(client *sockx.Client, data interface{}) {
		msg, ok := data.(map[string]interface{})
		if !ok {
			return
		}

		roomName, ok := msg["room"].(string)
		if !ok {
			return
		}

		text, _ := msg["text"]

		log.Printf("Client %s sent message to room %s: %v", client.ID, roomName, text)

		// Send to specific room
		room := ns.Room(roomName)
		room.Emit("room-message", map[string]interface{}{
			"from": client.ID,
			"text": text,
			"room": roomName,
		})
	})

	// Setup HTTP routes
	http.HandleFunc("/ws", server.ServeWebSocket("/"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Sockx Chat Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; }
        #messages { border: 1px solid #ccc; height: 400px; overflow-y: scroll; padding: 10px; margin-bottom: 10px; }
        .message { margin: 5px 0; padding: 5px; background: #f0f0f0; border-radius: 3px; }
        input, button { padding: 8px; margin: 5px; }
        input[type="text"] { width: 300px; }
    </style>
</head>
<body>
    <h1>Sockx Chat Example</h1>
    <div id="messages"></div>
    <div>
        <input type="text" id="messageInput" placeholder="Type a message...">
        <button onclick="sendMessage()">Send</button>
    </div>
    <div>
        <input type="text" id="roomInput" placeholder="Room name">
        <button onclick="joinRoom()">Join Room</button>
    </div>
    <div>
        <input type="text" id="roomMessageInput" placeholder="Room message">
        <input type="text" id="targetRoom" placeholder="Target room">
        <button onclick="sendRoomMessage()">Send to Room</button>
    </div>

    <script>
        const ws = new WebSocket('ws://localhost:8080/ws');
        const messagesDiv = document.getElementById('messages');

        ws.onopen = function() {
            addMessage('Connected to server');
        };

        ws.onmessage = function(event) {
            const msg = JSON.parse(event.data);
            addMessage('Event: ' + msg.event + ' - ' + JSON.stringify(msg.data));
        };

        ws.onerror = function(error) {
            addMessage('Error: ' + error);
        };

        ws.onclose = function() {
            addMessage('Disconnected from server');
        };

        function addMessage(text) {
            const div = document.createElement('div');
            div.className = 'message';
            div.textContent = text;
            messagesDiv.appendChild(div);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        function sendMessage() {
            const input = document.getElementById('messageInput');
            const message = input.value;
            if (message) {
                ws.send(JSON.stringify({
                    event: 'message',
                    data: message
                }));
                input.value = '';
            }
        }

        function joinRoom() {
            const input = document.getElementById('roomInput');
            const room = input.value;
            if (room) {
                ws.send(JSON.stringify({
                    event: 'join-room',
                    data: room
                }));
                input.value = '';
            }
        }

        function sendRoomMessage() {
            const messageInput = document.getElementById('roomMessageInput');
            const roomInput = document.getElementById('targetRoom');
            const message = messageInput.value;
            const room = roomInput.value;
            if (message && room) {
                ws.send(JSON.stringify({
                    event: 'room-message',
                    data: {
                        text: message,
                        room: room
                    }
                }));
                messageInput.value = '';
            }
        }

        document.getElementById('messageInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') sendMessage();
        });
    </script>
</body>
</html>
		`)
	})

	fmt.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
