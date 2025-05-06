package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

var jwtKey = []byte("your_secret_key")

// Tạo JWT token cho người dùng
func GenerateJWT(email string, role string) (string, error) {
	claims := &jwt.MapClaims{
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iss":   "task-management-system",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Println("Lỗi khi tạo token:", err)
		return "", err
	}
	return tokenString, nil
}

// Middleware xác thực JWT
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

	// Lưu email và role vào context để dùng cho các middleware hoặc controller sau này
	c.Set("email", (*claims)["email"])
	c.Set("role", (*claims)["role"])
	c.Next()
}

// Middleware kiểm tra quyền truy cập của người dùng dựa trên role
func CheckRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy thông tin role từ context đã xác thực JWT
		role, exists := c.Get("role")
		if !exists || role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không có quyền truy cập"})
			c.Abort()
			return
		}
		c.Next()
	}
}
