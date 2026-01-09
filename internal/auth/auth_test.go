package auth

import (
	"github.com/google/uuid"
	"time"
	"testing"
)
func TestHashAndCheckPassword(t *testing.T) {
	password := "TestPassword123!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Error hashing password: %s", err)
	}
	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("Error checking password hash: %s", err)
	}
	if !match {
		t.Fatalf("Password and hash do not match")
	}
}

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secretKey := "TestSecretKey"
	expiresIn := time.Minute * 5

	tokenString, err := MakeJWT(userID, secretKey, expiresIn)
	if err != nil {
		t.Fatalf("Error making JWT: %s", err)
	}

	returnedUserID, err := ValidateJWT(tokenString, secretKey)
	if err != nil {
		t.Fatalf("Error validating JWT: %s", err)
	}

	if returnedUserID != userID {
		t.Fatalf("Returned user ID does not match original. Got %s, want %s", returnedUserID, userID)
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	secretKey := "TestSecretKey"
	expiresIn := time.Second * 1

	tokenString, err := MakeJWT(userID, secretKey, expiresIn)
	if err != nil {
		t.Fatalf("Error making JWT: %s", err)
	}

	time.Sleep(time.Second * 2)

	_, err = ValidateJWT(tokenString, secretKey)
	if err == nil {
		t.Fatalf("Expected error validating expired JWT, got none")
	}
}