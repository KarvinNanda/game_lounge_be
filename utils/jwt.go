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

// Audience membedakan staff token dan customer token. Keduanya ditandatangani
// dengan secret yang sama, jadi tanpa aud sebuah customer token lolos
// ValidateJWT (StaffID kosong) dan bisa dipakai sebagai staff_token.
const (
	audienceStaff    = "staff"
	audienceCustomer = "customer"
)

// parseHS256 memverifikasi signature HS256, exp (wajib), dan audience.
func parseHS256(tokenString string, claims jwt.Claims, audience string) (*jwt.Token, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}
	return jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (interface{}, error) {
		return secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(audience),
	)
}

type JWTClaims struct {
	StaffID     string   `json:"staff_id"`
	Username    string   `json:"username"`
	RoleID      uint     `json:"role_id"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions"`
	// TokenVersion harus sama dengan staffs.token_version; naik saat logout /
	// ganti password, sehingga token lama langsung tidak berlaku.
	TokenVersion uint `json:"tv"`
	jwt.RegisteredClaims
}

func GenerateJWT(staffID, username string, roleID uint, isSystem bool, permissions []string, tokenVersion uint) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	expiredHours, _ := strconv.Atoi(os.Getenv("JWT_EXPIRED_HOURS"))
	if expiredHours == 0 {
		expiredHours = 24
	}

	claims := JWTClaims{
		StaffID:      staffID,
		Username:     username,
		RoleID:       roleID,
		IsSystem:     isSystem,
		Permissions:  permissions,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{audienceStaff},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiredHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := parseHS256(tokenString, &JWTClaims{}, audienceStaff)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid || claims.StaffID == "" {
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
	TokenVersion uint   `json:"tv"`            // = customers.token_version
	jwt.RegisteredClaims
}

// GenerateCustomerJWT membuat token JWT untuk customer (expire 7 hari).
func GenerateCustomerJWT(customerID, customerType string, tokenVersion uint) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	claims := CustomerJWTClaims{
		CustomerID:   customerID,
		CustomerType: customerType,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{audienceCustomer},
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
	token, err := parseHS256(tokenStr, &CustomerJWTClaims{}, audienceCustomer)
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
