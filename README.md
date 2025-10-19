# 🧩 Sockx

> A lightweight, Socket.IO-style real-time server written in Go — but fully compatible with any WebSocket client.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Status](https://img.shields.io/badge/status-active-success.svg)
![Contributions](https://img.shields.io/badge/contributions-welcome-brightgreen.svg)

---

## 🚀 Overview

**Sockx** is a Go library for building **real-time communication systems** using **Gorilla WebSocket** and Go’s native `net/http`.  
It’s inspired by Socket.IO, but without vendor lock-in — meaning you can connect **any standard WebSocket client**.

Sockx provides an **event-driven API** with:
- 🎯 **Custom Events**
- 🏠 **Rooms**
- 🧭 **Namespaces**
- 📡 **Broadcasts (one-to-one, one-to-many, global)**
- ⚙️ **HTTP integration** using Go’s standard library

Perfect for chat apps, dashboards, multiplayer games, IoT hubs, and live collaboration tools.

---

## 🧱 Features

| Feature | Description |
|----------|-------------|
| **Rooms** | Group clients and send messages to specific subsets. |
| **Namespaces** | Create isolated communication channels. |
| **Broadcast** | Emit to all connected clients or specific rooms. |
| **Event System** | Define custom events and handlers with ease. |
| **Pure WebSocket Compatibility** | Works with any WebSocket client (browser, Python, Go, Node, etc.). |
| **Extensible Design** | Built with modularity in mind for middleware and hooks. |

---
