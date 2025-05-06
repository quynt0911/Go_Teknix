package main

import (
	"fmt"
	"log"
	"net/http"
	"task-management-system/controllers"
	"task-management-system/middleware"
	"task-management-system/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"task-management-system/database"
)

var db *gorm.DB

// Hàm khởi tạo kết nối cơ sở dữ liệu PostgreSQL
func initDB() {
	dsn := "host=localhost user=postgres password=091123 dbname=taskdb port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Không thể kết nối đến PostgreSQL!")
	}
	fmt.Println("Kết nối thành công tới PostgreSQL!")

	// Tạo bảng nếu chưa có
	db.AutoMigrate(&models.User{}, &models.Task{})
}

func main() {
	initDB()

	database.ConnectDB()

	r := gin.Default()

	// Load giao diện từ thư mục templates/
	r.LoadHTMLGlob("templates/*")

	// Trang chủ
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Giao diện đăng nhập
	r.GET("/login-page", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login-page.html", nil)
	})

	// Giao diện quản lý tasks
	r.GET("/tasks-page", func(c *gin.Context) {
		c.HTML(http.StatusOK, "tasks-page.html", nil)
	})
	// Trang tạo task (chỉ admin)
	r.GET("/create-task", middleware.CheckAuth, middleware.CheckRole("admin"), func(c *gin.Context) {
		c.HTML(http.StatusOK, "create-task.html", nil)
	})

	// Trang cập nhật task (chỉ admin)
	r.GET("/update-task", middleware.CheckAuth, middleware.CheckRole("admin"), func(c *gin.Context) {
		c.HTML(http.StatusOK, "update-task.html", nil)
	})

	// Trang xác nhận xóa task (chỉ admin)
	r.GET("/delete-task", middleware.CheckAuth, middleware.CheckRole("admin"), func(c *gin.Context) {
		c.HTML(http.StatusOK, "delete-task.html", nil)
	})

	// API đăng nhập - trả về JWT token
	r.POST("/login", func(c *gin.Context) {
		var loginReq struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Kiểm tra tài khoản mặc định
		var role string
		if loginReq.Email == "admin@123" && loginReq.Password == "123456" {
			role = "admin"
		} else if loginReq.Email == "user@123" && loginReq.Password == "123456" {
			role = "user"
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc mật khẩu không đúng"})
			return
		}

		// Tạo JWT token, kèm theo role
		token, err := middleware.GenerateJWT(loginReq.Email, role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"role":  role,
		})

	})

	// Middleware kiểm tra role của người dùng
	r.POST("/tasks", middleware.CheckAuth, middleware.CheckRole("admin"), controllers.CreateTask)
	r.GET("/tasks", middleware.CheckAuth, controllers.GetTasks)
	r.PUT("/tasks/:id", middleware.CheckAuth, middleware.CheckRole("admin"), controllers.UpdateTask)
	r.DELETE("/tasks/:id", middleware.CheckAuth, middleware.CheckRole("admin"), controllers.DeleteTask)

	// Khởi chạy server tại cổng 8080
	r.Run(":8080")
}
