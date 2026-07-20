package utils

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtSecret mengembalikan secret key JWT dari environment.
// Secret kosong DITOLAK — token yang ditandatangani dengan secret kosong
// dapat dipalsukan dengan mudah oleh penyerang.
func jwtSecret() ([]byte, error) {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		return nil, errors.New("JWT_SECRET belum dikonfigurasi")
	}
	return []byte(s), nil
}

type JWTClaims struct {
	StaffID     string   `json:"staff_id"`
	Username    string   `json:"username"`
	RoleID      uint     `json:"role_id"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func GenerateJWT(staffID, username string, roleID uint, isSystem bool, permissions []string) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
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
	return token.SignedString(secret)
}

func ValidateJWT(tokenString string) (*JWTClaims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
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

// ── Customer JWT ──────────────────────────────────────────────────────────────

// CustomerJWTClaims adalah claims untuk customer token.
// Berbeda dari staff: punya field customer_id bukan staff_id.
type CustomerJWTClaims struct {
	CustomerID   string `json:"customer_id"`
	CustomerType string `json:"customer_type"` // "member" | "regular"
	jwt.RegisteredClaims
}

// GenerateCustomerJWT membuat token JWT untuk customer (expire 7 hari).
func GenerateCustomerJWT(customerID, customerType string) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	claims := CustomerJWTClaims{
		CustomerID:   customerID,
		CustomerType: customerType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateCustomerJWT memvalidasi token customer.
// Return error jika bukan customer token (cek field customer_id).
func ValidateCustomerJWT(tokenStr string) (*CustomerJWTClaims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(tokenStr, &CustomerJWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*CustomerJWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.CustomerID == "" {
		return nil, errors.New("bukan customer token")
	}
	return claims, nil
}
