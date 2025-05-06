package websocket

import (
	"chat-app/redis"
	"net/http"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Cho phép kết nối từ mọi nguồn
		return true
	},
}

func HandleWebSocket(c *gin.Context) {
	// Lấy userID từ query parameter
	userID := c.Query("user")
	if userID == "" {
		// Trả về lỗi nếu thiếu userID
		c.String(http.StatusBadRequest, "Missing user ID")
		return
	}

	// Upgrade kết nối HTTP lên WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Đăng ký người dùng vào Hub
	client := Client{Conn: conn, UserID: userID}
	HubInstance.Register <- client

	// Thêm người dùng vào danh sách trực tuyến trong Redis
	redis.AddOnlineUser(userID)

	// Đảm bảo người dùng bị xóa khỏi Hub và Redis khi kết nối đóng
	defer func() {
		HubInstance.Unregister <- conn
		redis.RemoveOnlineUser(userID)
		conn.Close()
	}()

	// Đọc và xử lý các tin nhắn từ WebSocket
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// Nếu có lỗi khi đọc tin nhắn (chắc là người dùng đóng kết nối)
			log.Printf("Error reading message: %v", err)
			break
		}

		// Kiểm tra rate limit, nếu vượt quá sẽ không gửi tin nhắn
		if !redis.AllowSend(userID) {
			conn.WriteMessage(websocket.TextMessage, []byte("⚠️ Bạn đang gửi quá nhanh! Vui lòng đợi!"))
			continue
		}

		// Lưu tin nhắn vào Redis
		redis.SaveMessage(userID, string(msg))

		// Định dạng lại tin nhắn và gửi đến tất cả người dùng trong Hub
		formatted := []byte(userID + ": " + string(msg))
		HubInstance.Broadcast <- formatted
	}
}
