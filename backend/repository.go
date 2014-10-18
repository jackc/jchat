package main

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"crypto/rand"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

type DuplicationError struct {
	Field string // Field or fields that caused the rejection
}

func (e DuplicationError) Error() string {
	return fmt.Sprintf("%s is already taken", e.Field)
}

type UserRepository interface {
	Create(name, email, password string) (userID int32, err error)
	Login(email, password string) (userID int32, err error)
	SetPassword(userID int32, password string) (err error)
}

type SessionRepository interface {
	Create(userID int32) (sessionID []byte, err error)
	Delete(sessionID []byte) (err error)
	GetUserIDBySessionID(sessionID []byte) (userID int32, err error)
}

func DigestPassword(password string) (digest, salt []byte, err error) {
	salt = make([]byte, 8)
	_, err = rand.Read(salt)
	if err != nil {
		return nil, nil, err
	}

	digest, err = scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		return nil, nil, err
	}

	return digest, salt, nil
}

func PasswordMatch(password string, validDigest, salt []byte) bool {
	digest, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		return false
	}

	return bytes.Equal(digest, validDigest)
}
