package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var redisClient *redis.Client

func InitRedis() error {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	_, err := redisClient.Ping(ctx).Result()
	return err
}

func GetChatHistory() ([]string, error) {
	messages, err := redisClient.LRange(ctx, "chat-history", 0, -1).Result()
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func SaveMessage(userID, msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := timestamp + " - " + userID + ": " + msg
	redisClient.RPush(ctx, "chat-history", entry)
	redisClient.LTrim(ctx, "chat-history", -100, -1)
}

func AllowSend(userID string) bool {
	key := "rate:" + userID
	count, _ := redisClient.Incr(ctx, key).Result()
	if count == 1 {
		redisClient.Expire(ctx, key, time.Minute)
	}
	return count <= 5
}

func AddOnlineUser(userID string) {
	redisClient.SAdd(ctx, "online-users", userID)
}
func RemoveOnlineUser(userID string) {
	redisClient.SRem(ctx, "online-users", userID)
}
func GetOnlineUsers() ([]string, error) {
	users, err := redisClient.SMembers(ctx, "online-users").Result()
	if err != nil {
		return nil, err
	}
	return users, nil
}

func UpdateOnlineUsers() {
	onlineUsers, err := GetOnlineUsers()
	if err != nil {
		log.Printf("Error fetching online users: %v", err)
		return
	}

	onlineUsersMessage := "Online Users: " + strings.Join(onlineUsers, ", ")
	hub.Broadcast <- []byte(onlineUsersMessage)
}

// ================= WebSocket ================= //

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

var hub = Hub{
	Clients:    make(map[*websocket.Conn]string),
	Broadcast:  make(chan []byte),
	Register:   make(chan Client),
	Unregister: make(chan *websocket.Conn),
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.Conn] = client.UserID
			log.Printf("User %s has joined", client.UserID)
		case conn := <-h.Unregister:
			if user, ok := h.Clients[conn]; ok {
				delete(h.Clients, conn)
				conn.Close()
				log.Printf("User %s disconnected", user)
			}
		case msg := <-h.Broadcast:
			for conn := range h.Clients {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Printf("Send error: %v", err)
					conn.Close()
					delete(h.Clients, conn)
				}
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleWebSocket(c *gin.Context) {
	// Lấy tên người dùng từ query string
	userID := c.Query("user")
	if userID == "" {
		c.String(http.StatusBadRequest, "Missing user ID")
		return
	}

	// Tạo kết nối WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket error: %v", err)
		return
	}

	client := Client{Conn: conn, UserID: userID}
	hub.Register <- client
	AddOnlineUser(userID)

	// Gửi lại lịch sử tin nhắn từ Redis
	history, err := GetChatHistory()
	if err != nil {
		log.Printf("Error fetching chat history: %v", err)
	}
	for _, msg := range history {
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
	}

	// Gửi danh sách người dùng online khi kết nối
	UpdateOnlineUsers()

	defer func() {
		hub.Unregister <- conn
		RemoveOnlineUser(userID)
		conn.Close()
		UpdateOnlineUsers() // Gửi lại danh sách người dùng online khi ai đó offline
	}()

	// Nhận và gửi tin nhắn qua WebSocket
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if !AllowSend(userID) {
			conn.WriteMessage(websocket.TextMessage, []byte("⚠️ Bạn đang gửi quá nhanh!"))
			continue
		}
		SaveMessage(userID, string(msg)) // Lưu tin nhắn vào Redis
		formatted := []byte(userID + ": " + string(msg))
		hub.Broadcast <- formatted
	}
}

// ================== MAIN ================== //

func main() {
	if err := InitRedis(); err != nil {
		log.Fatalf("Redis failed: %v", err)
	}

	go hub.Run()

	r := gin.Default()
	r.StaticFile("/", "./public/index.html")
	r.GET("/ws", HandleWebSocket)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := r.Run(":8080"); err != nil {
			log.Fatal("Server error:", err)
		}
	}()

	<-stop
	log.Println("Server shutting down...")
}
