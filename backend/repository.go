package main

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"crypto/rand"
	"errors"
	"fmt"
	"time"
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

type Session struct {
	UserID int32
}

type SessionRepository interface {
	Create(session Session) (sessionID []byte, err error)
	Delete(sessionID []byte) (err error)
	GetSession(sessionID []byte) (session Session, err error)
}

type Channel struct {
	ID   int32
	Name string
}

type Message struct {
	ID       int32
	AuthorID int32
	Body     string
	Time     time.Time
}

type ChatRepository interface {
	CreateChannel(name string, userID int32) (channelID int32, err error)
	GetChannels() (channels []Channel, err error)
	PostMessage(channelID int32, authorID, body string) (messageID int32, err error)
	GetMessages(channelID int32, beforeMessageID int32, maxCount int32) (messages []Message, err error)
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

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	return nil
}
