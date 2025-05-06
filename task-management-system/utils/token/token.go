package token

import (
	"errors"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

var jwtKey = []byte("your_secret_key")

type Claims struct {
	UserID uint
	Role   string
	jwt.StandardClaims
}

// GenerateToken: Tạo token mới
func GenerateToken(userID uint, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(72 * time.Hour).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// ExtractTokenMetadata: Lấy user info từ token
func ExtractTokenMetadata(c *gin.Context) (*Claims, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, errors.New("Authorization header missing")
	}

	tokenStr := strings.Replace(authHeader, "Bearer ", "", 1)
	tokenObj, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !tokenObj.Valid {
		return nil, errors.New("Invalid token")
	}

	claims, ok := tokenObj.Claims.(*Claims)
	if !ok {
		return nil, errors.New("Couldn't parse claims")
	}

	return claims, nil
}
