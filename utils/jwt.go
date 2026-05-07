package utils

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	StaffID     string   `json:"staff_id"`
	Username    string   `json:"username"`
	RoleID      uint     `json:"role_id"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func GenerateJWT(staffID, username string, roleID uint, isSystem bool, permissions []string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	expiredHours, _ := strconv.Atoi(os.Getenv("JWT_EXPIRED_HOURS"))
	if expiredHours == 0 {
		expiredHours = 24
	}

	claims := JWTClaims{
		StaffID:     staffID,
		Username:    username,
		RoleID:      roleID,
		IsSystem:    isSystem,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiredHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateJWT(tokenString string) (*JWTClaims, error) {
	secret := os.Getenv("JWT_SECRET")

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
