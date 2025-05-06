package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"chat-app/utils"
	"chat-app/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	// Khởi tạo Redis và kiểm tra kết nối
	err := utils.InitRedis()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Khởi chạy Hub ở goroutine
	go websocket.HubInstance.Run()

	// Tạo router Gin
	router := gin.Default()

	// Cung cấp file HTML cho giao diện người dùng
	router.StaticFile("/", "./public/index.html")

	// WebSocket endpoint
	router.GET("/ws", websocket.HandleWebSocket)

	// Cấu hình để dừng server khi có tín hiệu từ hệ thống
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Chạy server
	go func() {
		if err := router.Run(":8080"); err != nil {
			log.Fatal("Server failed:", err)
		}
	}()

	// Chờ tín hiệu dừng server (Ctrl+C hoặc Kill)
	<-stop
	log.Println("Shutting down server gracefully...")
}
