package redis

import (
	"context"
	"time"
	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()
var Client *redis.Client

// Hàm khởi tạo Redis client
func InitRedis() {
	Client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Địa chỉ Redis server
	})
}

// Lưu tin nhắn vào Redis (lịch sử chat)
func SaveMessage(userID string, message string) {
	entry := time.Now().Format(time.RFC3339) + " - " + userID + ": " + message
	Client.RPush(Ctx, "chat-history", entry)
}

// Thêm người dùng vào danh sách trực tuyến
func AddOnlineUser(userID string) {
	Client.SAdd(Ctx, "online-users", userID)
}

// Loại bỏ người dùng khỏi danh sách trực tuyến
func RemoveOnlineUser(userID string) {
	Client.SRem(Ctx, "online-users", userID)
}

// Kiểm tra nếu người dùng có thể gửi tin nhắn hay không (rate limiting)
func AllowSend(userID string) bool {
	key := "rate:" + userID
	count, _ := Client.Incr(Ctx, key).Result()
	if count == 1 {
		// Đặt hết hạn cho khóa sau 60 giây
		Client.Expire(Ctx, key, 60*time.Second)
	}
	// Giới hạn 5 tin nhắn trong 1 phút
	return count <= 5
}
