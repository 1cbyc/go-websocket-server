package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Auth struct {
	Secret     []byte
	Expiry     time.Duration
}

func New(secret string, expiry time.Duration) *Auth {
	return &Auth{
		Secret: []byte(secret),
		Expiry: expiry,
	}
}

func (a *Auth) GenerateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(a.Expiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.Secret)
}

func (a *Auth) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return a.Secret, nil
	})
	if err != nil || !token.Valid {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", err
	}
	userID, ok := claims["sub"].(string)
	if !ok {
		return "", err
	}
	return userID, nil
} 