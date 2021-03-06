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
	GetUser(userID int32) (user User, err error)
	CreateUser(name, email, password string) (user User, err error)
	Login(email, password string) (user User, err error)
	SetPassword(userID int32, password string) (err error)

	CreatePasswordResetToken(email string, requestIP string) (token string, err error)
	SetPasswordByToken(token, password string, completionIP string) error
}

type UserCreatedSignaler interface {
	UserCreatedSignal() *UserSignal
}

// +gen signal
type User struct {
	ID    int32
	Name  string
	Email string
}

// +gen signal
type Channel struct {
	ID   int32
	Name string
}

// +gen signal
type Message struct {
	ID        int64
	ChannelID int32
	AuthorID  int32
	Body      string
	Time      time.Time
}

type SessionRepository interface {
	CreateSession(userID int32) (sessionID string, err error)
	DeleteSession(sessionID string) (err error)
	GetUserIDBySessionID(sessionID string) (userID int32, err error)
}

type ChatRepository interface {
	CreateChannel(name string, userID int32) (channelID int32, err error)
	RenameChannel(channelID int32, name string) (err error)
	GetChannels() (channels []Channel, err error)
	PostMessage(channelID int32, authorID int32, body string) (messageID int64, err error)
	GetMessages(channelID int32, beforeMessageID int32, maxCount int32) (messages []Message, err error)
	GetInit(userID int32) (json []byte, err error)
}

type ChannelCreatedSignaler interface {
	ChannelCreatedSignal() *ChannelSignal
}

type ChannelRenamedSignaler interface {
	ChannelRenamedSignal() *ChannelSignal
}

type MessagePostedSignaler interface {
	MessagePostedSignal() *MessageSignal
}

type Repository interface {
	UserRepository
	UserCreatedSignaler
	SessionRepository
	ChatRepository
	ChannelCreatedSignaler
	ChannelRenamedSignaler
	MessagePostedSignaler
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
