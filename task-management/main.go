package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var db *gorm.DB
var jwtKey = []byte("secretkey")
var redisClient *redis.Client

// User model
type User struct {
	ID       uint   `gorm:"primary_key"`
	Email    string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	Role     string `gorm:"default:'user'"`
}

// Task model
type Task struct {
	ID          uint      `gorm:"primary_key"`
	Title       string    `gorm:"not null"`
	Description string    `gorm:"not null"`
	Status      string    `gorm:"default:'pending'"`
	Category    string    `gorm:"not null"`
	DueDate     time.Time `gorm:"not null"`
	UserID      uint      `gorm:"not null"`
	User        User      `gorm:"foreignKey:UserID"`
}

// Claims struct for JWT
type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.StandardClaims
}

// InitDatabase initializes the database
func InitDatabase() (*gorm.DB, error) {
	dsn := "host=localhost user=postgres password=091123 dbname=taskdb port=5432 sslmode=disable"
	db, err := gorm.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&User{}, &Task{})
	return db, nil
}

// GenerateJWT generates a new JWT token
func GenerateJWT(email, role string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: email,
		Role:  role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// ParseJWT parses the JWT token
func ParseJWT(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("could not parse claims")
	}
	return claims, nil
}

// RoleMiddleware checks if the user has the correct role
func RoleMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		claims, err := ParseJWT(token)
		if err != nil || claims.Role != role {
			c.JSON(403, gin.H{"message": "forbidden"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitMiddleware limits the number of requests
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s", clientIP)
		limiter := redis.NewClient(&redis.Options{
			Addr: "localhost:6379", // Redis server
		})
		// Limit to 100 requests per hour
		count, err := limiter.Incr(context.Background(), key).Result()
		if err != nil {
			c.JSON(500, gin.H{"message": "internal server error"})
			c.Abort()
			return
		}
		if count == 1 {
			limiter.Expire(context.Background(), key, time.Hour)
		}
		if count > 100 {
			c.JSON(429, gin.H{"message": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// CreateTask handles creating a new task
func CreateTask(c *gin.Context) {
	var task Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid data"})
		return
	}
	db.Create(&task)
	c.JSON(http.StatusCreated, task)
}

// UpdateTask handles updating an existing task
func UpdateTask(c *gin.Context) {
	id := c.Param("id")
	var task Task
	if err := db.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
		return
	}
	c.ShouldBindJSON(&task)
	db.Save(&task)
	c.JSON(http.StatusOK, task)
}

// DeleteTask handles deleting a task
func DeleteTask(c *gin.Context) {
	id := c.Param("id")
	var task Task
	if err := db.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
		return
	}
	db.Delete(&task)
	c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
}

// Login handles user login and returns a JWT token
func Login(c *gin.Context) {
	var loginData User
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid data"})
		return
	}

	var user User
	if err := db.Where("email = ?", loginData.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid credentials"})
		return
	}

	// Assuming password verification is done
	token, err := GenerateJWT(user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func main() {
	fmt.Println("Connecting to PostgreSQL...")
	var err error
	db, err = InitDatabase()
	if err != nil {
		fmt.Println("failed to connect to database")
		return
	}

	r := gin.Default()

	// Rate limiting middleware
	r.Use(RateLimitMiddleware())

	// Public routes
	r.POST("/login", Login)

	// Protected routes
	r.POST("/tasks", RoleMiddleware("admin"), CreateTask)
	r.PUT("/tasks/:id", RoleMiddleware("admin"), UpdateTask)
	r.DELETE("/tasks/:id", RoleMiddleware("admin"), DeleteTask)

	r.Run(":8080")
}
