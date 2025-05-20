package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(userId string) (string, error) {
	jwtSecretStr := os.Getenv("JWT_SECRET")
	if jwtSecretStr == "" {
		return "", errors.New("JWT_SECRET not set in environment")
	}
	jwtSecret := []byte(jwtSecretStr)

	// Set standard JWT claims
	claims := jwt.MapClaims{
		"userId": userId,
		"iat":    time.Now().Unix(),                     // issued at
		"exp":    time.Now().Add(time.Hour * 72).Unix(), // expires in 72 hours
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
