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
	"golang.org/x/crypto/bcrypt"
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

// Register: API tạo tài khoản mới
func Register(c *gin.Context) {
	var registerReq struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&registerReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Kiểm tra xem email đã tồn tại trong cơ sở dữ liệu chưa
	var existingUser User
	if err := DB.Where("email = ?", registerReq.Email).First(&existingUser).Error; err == nil {
		// Nếu email đã tồn tại, trả về lỗi
		c.JSON(http.StatusConflict, gin.H{"error": "Email đã được sử dụng"})
		return
	}

	// Mã hóa mật khẩu
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registerReq.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mã hóa mật khẩu"})
		return
	}

	user := User{
		Name:     registerReq.Name,
		Email:    registerReq.Email,
		Password: string(hashedPassword),
		Role:     registerReq.Role,
	}

	// Tạo người dùng mới
	if err := DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo tài khoản"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tạo tài khoản thành công"})
}

// Login: API đăng nhập và tạo token
func Login(c *gin.Context) {
	var loginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	if err := DB.Where("email = ?", loginReq.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc mật khẩu không đúng"})
		return
	}

	// So sánh mật khẩu đã mã hóa
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc mật khẩu không đúng"})
		return
	}

	// Sinh JWT Token
	token, err := GenerateJWT(user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "role": user.Role})
}

// CreateTask: API tạo task mới
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

// GetTasks: API lấy danh sách task
func GetTasks(c *gin.Context) {
	var tasks []Task
	if err := DB.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

// UpdateTask: API cập nhật task
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

// DeleteTask: API xóa task
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

	// Đăng ký tài khoản mới
	r.POST("/register", Register)

	// Đăng nhập và sinh token JWT
	r.POST("/login", Login)

	// API CRUD
	r.POST("/tasks", CheckAuth, CheckRole("admin"), CreateTask)
	r.GET("/tasks", CheckAuth, GetTasks)
	r.PUT("/tasks/:id", CheckAuth, CheckRole("admin"), UpdateTask)
	r.DELETE("/tasks/:id", CheckAuth, CheckRole("admin"), DeleteTask)

	r.Run(":8080")
}
