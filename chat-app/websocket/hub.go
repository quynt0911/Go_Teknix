package websocket

import (
	"github.com/gorilla/websocket"
	"log"
)

type Client struct {
	Conn   *websocket.Conn
	UserID string
}

type Hub struct {
	Clients    map[*websocket.Conn]string
	Broadcast  chan []byte
	Register   chan Client
	Unregister chan *websocket.Conn
}

var HubInstance = Hub{
	Clients:    make(map[*websocket.Conn]string),
	Broadcast:  make(chan []byte),
	Register:   make(chan Client),
	Unregister: make(chan *websocket.Conn),
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			// Đăng ký client mới
			h.Clients[client.Conn] = client.UserID
			log.Printf("User %s has joined the chat", client.UserID)
		case conn := <-h.Unregister:
			// Xóa client khỏi Hub và đóng kết nối
			if _, ok := h.Clients[conn]; ok {
				delete(h.Clients, conn)
				conn.Close()
				log.Printf("A client disconnected")
			}
		case msg := <-h.Broadcast:
			// Phát broadcast đến tất cả clients
			for conn := range h.Clients {
				err := conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					log.Printf("Error sending message to %s: %v", h.Clients[conn], err)
					conn.Close() // Nếu có lỗi, đóng kết nối
					delete(h.Clients, conn) // Xóa client khỏi Hub
				}
			}
		}
	}
}
