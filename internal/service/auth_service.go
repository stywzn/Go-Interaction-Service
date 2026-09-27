package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stywzn/Go-Interaction-Service/config"
	"github.com/stywzn/Go-Interaction-Service/internal/model"
)

var jwtSecret = []byte("your-super-secret-key-change-in-prod")

// Claims JWT 声明结构
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Register 用户注册
func Register(username, password string) (*model.User, error) {
	var existingUser model.User
	if err := config.DB.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("用户已存在")
	}

	user := &model.User{
		Username:     username,
		Password:     password,
		StorageQuota: 5,
	}

	if err := config.DB.Create(user).Error; err != nil {
		return nil, fmt.Errorf("用户注册失败: %v", err)
	}
	return user, nil
}

// Login 用户登录
func Login(username, password string) (string, *model.User, error) {
	var user model.User
	if err := config.DB.Where("username = ? AND password = ?", username, password).First(&user).Error; err != nil {
		return "", nil, fmt.Errorf("用户名或密码错误")
	}

	// 生成 JWT Token
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", nil, fmt.Errorf("Token 生成失败: %v", err)
	}

	return tokenString, &user, nil
}

// VerifyToken 验证 JWT Token
func VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("Token 无效")
	}
	return claims, nil
}
