package auth

import (
	"github.com/alexedwards/argon2id"
	"log"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
	"net/http"
	"errors"
	"fmt"
	"crypto/rand"
	"encoding/hex"
)

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		return "", err
	}
	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		log.Printf("Error comparing password and hash: %s", err)
		return false, err
	}
	return match, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "chirpy",
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject: userID.String(),
	})
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		log.Printf("Error signing JWT: %s", err)
		return "", err
	}
	return tokenString, nil
}

func ValidateJWT(tokenString, secretKey string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		log.Printf("Error parsing JWT: %s", err)
		return uuid.Nil, err
	}
	if token.Valid {
		userId, err := uuid.Parse(claims.Subject)
		if err != nil {
			log.Printf("Error parsing user ID from JWT claims: %s", err)
			return uuid.Nil, err
		}
		return userId, nil
	} else {
		log.Printf("Invalid JWT token")
		return uuid.Nil, err
	}
}

func GetBearerToken(header http.Header) (string, error) {
	authHeader := header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization header missing")
	}
	var token string
	_, err := fmt.Sscanf(authHeader, "Bearer %s", &token)
	if err != nil {
		log.Printf("Error extracting bearer token: %s", err)
		return "", err
	}

	return token, nil
}

func MakeRefreshToken() (string) {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	hex := hex.EncodeToString(bytes)
	return hex
}