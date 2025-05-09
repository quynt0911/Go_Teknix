package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ==================== Models ====================

type Task struct {
	gorm.Model
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	DueDate     string `json:"due_date"`
	Category    string `json:"category"`
	UserID      uint   `json:"user_id"`
}

type User struct {
	gorm.Model
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// ==================== Database ====================

var DB *gorm.DB

func ConnectDB() {
	var err error
	dsn := "host=localhost user=postgres password=091123 dbname=taskdb port=5432 sslmode=disable"
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	if err := DB.AutoMigrate(&User{}, &Task{}); err != nil {
		log.Fatal("❌ AutoMigrate error:", err)
	}
	fmt.Println("✅ Connected to Database")
}

// ==================== JWT Middleware ====================

var jwtKey = []byte("your_secret_key")

// GenerateJWT: Tạo JWT token từ email và role
func GenerateJWT(email string, role string) (string, error) {
	claims := &jwt.MapClaims{
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iss":   "task-management-system",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// CheckAuth: Middleware xác thực JWT
func CheckAuth(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu header Authorization"})
		c.Abort()
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Header Authorization không hợp lệ"})
		c.Abort()
		return
	}
	tokenString := parts[1]
	claims := &jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
		c.Abort()
		return
	}
	c.Set("email", (*claims)["email"])
	c.Set("role", (*claims)["role"])
	c.Next()
}

// CheckRole: Middleware kiểm tra role người dùng
func CheckRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không có quyền truy cập"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ==================== Controllers ====================

func CreateTask(c *gin.Context) {
	var task Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := DB.Create(&task)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func GetTasks(c *gin.Context) {
	var tasks []Task
	if err := DB.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func UpdateTask(c *gin.Context) {
	id := c.Param("id")
	var existingTask Task
	if err := DB.Where("id = ?", id).First(&existingTask).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	var updateData Task
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existingTask.Title = updateData.Title
	existingTask.Description = updateData.Description
	existingTask.Status = updateData.Status
	existingTask.DueDate = updateData.DueDate
	existingTask.Category = updateData.Category
	existingTask.UserID = updateData.UserID

	if err := DB.Save(&existingTask).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existingTask)
}

func DeleteTask(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}
	if err := DB.Delete(&Task{}, taskID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

// ==================== Main Function ====================

func main() {
	ConnectDB()
	r := gin.Default()

	// Đăng nhập và sinh token JWT
	r.POST("/login", func(c *gin.Context) {
		var loginReq struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var role string
		if loginReq.Email == "admin@123" && loginReq.Password == "123456" {
			role = "admin"
		} else if loginReq.Email == "user@123" && loginReq.Password == "123456" {
			role = "user"
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc mật khẩu không đúng"})
			return
		}
		token, err := GenerateJWT(loginReq.Email, role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token, "role": role})
	})

	// API CRUD
	r.POST("/tasks", CheckAuth, CheckRole("admin"), CreateTask)
	r.GET("/tasks", CheckAuth, GetTasks)
	r.PUT("/tasks/:id", CheckAuth, CheckRole("admin"), UpdateTask)
	r.DELETE("/tasks/:id", CheckAuth, CheckRole("admin"), DeleteTask)

	r.Run(":8080")
}
