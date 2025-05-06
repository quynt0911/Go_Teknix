package utils

import (
	"context"
	"time"

	"chat-app/redis"
)

// Hàm kiểm tra rate limit (giới hạn spam)
func RateLimit(userID string) bool {
	// Khóa Redis cho mỗi người dùng
	key := "user:" + userID + ":messages"

	// Tăng số lần gửi (Incr tăng biến đếm trong Redis)
	count, err := redis.Client.Incr(redis.Ctx, key).Result()
	if err != nil {
		return false // Nếu lỗi → chặn gửi
	}

	if count == 1 {
		// Nếu là lần đầu tiên, đặt thời gian hết hạn cho khóa
		redis.Client.Expire(redis.Ctx, key, time.Minute)
	}

	// Cho phép gửi nếu count <= 5 lần trong 1 phút
	return count <= 5
}
