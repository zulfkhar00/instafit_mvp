package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtSecret []byte) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": "Authorization token required"})
			return
		}

		// Validate the token and extract the user ID.
		userId, err := validateTokenAndExtractUserID(authHeader, jwtSecret)
		if err != nil {
			log.Printf("Auth error: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		// Inject userId into context
		c.Set("userId", userId)
		c.Next(ctx)
	}
}

func validateTokenAndExtractUserID(authHeader string, jwtSecret []byte) (string, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("authorization header format must be Bearer {token}")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token claims")
	}

	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		return "", errors.New("expiration time (exp) claim missing")
	}
	if time.Now().After(exp.Time) {
		return "", errors.New("token has expired")
	}

	userId, ok := claims["userId"].(string)
	if !ok || userId == "" {
		return "", errors.New("userId claim missing or invalid")
	}

	return userId, nil
}
