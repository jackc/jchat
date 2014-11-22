package main

import (
	"crypto/rand"
	"errors"
	"github.com/jackc/pgx"
	"github.com/vaughan0/go-ini"
	"io"
	"strconv"
)

func newConnPool(conf ini.File) (*pgx.ConnPool, error) {
	connConfig := pgx.ConnConfig{}

	connConfig.Host, _ = conf.Get("database", "host")
	if connConfig.Host == "" {
		return nil, errors.New("Config must contain database.host but it does not")
	}

	if p, ok := conf.Get("database", "port"); ok {
		n, err := strconv.ParseUint(p, 10, 16)
		connConfig.Port = uint16(n)
		if err != nil {
			return nil, err
		}
	}

	var ok bool

	if connConfig.Database, ok = conf.Get("database", "database"); !ok {
		return nil, errors.New("Config must contain database.database but it does not")
	}
	connConfig.User, _ = conf.Get("database", "user")
	connConfig.Password, _ = conf.Get("database", "password")

	connPoolConfig := pgx.ConnPoolConfig{
		ConnConfig:     connConfig,
		MaxConnections: 10,
	}

	pool, err := pgx.NewConnPool(connPoolConfig)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

type PgxUserRepository struct {
	pool *pgx.ConnPool
}

func NewPgxUserRepository(pool *pgx.ConnPool) *PgxUserRepository {
	return &PgxUserRepository{pool: pool}
}

func (repo *PgxUserRepository) Create(name, email, password string) (userID int32, err error) {
	digest, salt, err := DigestPassword(password)
	if err != nil {
		return 0, err
	}

	err = repo.pool.QueryRow(
		"insert into users(name, email, password_digest, password_salt) values($1, $2, $3, $4) returning id",
		name,
		email,
		digest,
		salt,
	).Scan(&userID)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func (repo *PgxUserRepository) Login(email, password string) (userID int32, err error) {
	var digest, salt []byte

	err = repo.pool.QueryRow("select id, password_digest, password_salt from users where email=$1",
		email,
	).Scan(&userID, &digest, &salt)
	if err == pgx.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}

	if !PasswordMatch(password, digest, salt) {
		return 0, ErrNotFound
	}

	return userID, nil
}

func (repo *PgxUserRepository) SetPassword(userID int32, password string) (err error) {
	digest, salt, err := DigestPassword(password)
	if err != nil {
		return err
	}

	commandTag, err := repo.pool.Exec("update users set password_digest=$1, password_salt=$2 where id=$3", digest, salt, userID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

type PgxSessionRepository struct {
	pool *pgx.ConnPool
}

func NewPgxSessionRepository(pool *pgx.ConnPool) *PgxSessionRepository {
	return &PgxSessionRepository{pool: pool}
}

func (repo *PgxSessionRepository) Create(session Session) (sessionID []byte, err error) {
	sessionID = make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, sessionID)
	if err != nil {
		return nil, err
	}

	_, err = repo.pool.Exec(`insert into sessions(id, user_id) values($1, $2)`, sessionID, session.UserID)
	return sessionID, err
}

func (repo *PgxSessionRepository) Delete(sessionID []byte) (err error) {
	commandTag, err := repo.pool.Exec(`delete from sessions where id=$1`, sessionID)
	if err != nil {
		return err
	}
	if commandTag != "DELETE 1" {
		return ErrNotFound
	}

	return nil
}

func (repo *PgxSessionRepository) GetSession(sessionID []byte) (session Session, err error) {
	err = repo.pool.QueryRow("select user_id from sessions where id=$1", sessionID).Scan(&session.UserID)
	if err == pgx.ErrNoRows {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}

	return session, nil
}

type PgxChatRepository struct {
	pool *pgx.ConnPool
}

func NewPgxChatRepository(pool *pgx.ConnPool) *PgxChatRepository {
	return &PgxChatRepository{pool: pool}
}

func (repo *PgxChatRepository) CreateChannel(name string, userID int32) (channelID int32, err error) {
	err = repo.pool.QueryRow(
		"insert into channels(name) values($1) returning id",
		name,
	).Scan(&channelID)
	if err != nil {
		return 0, err
	}

	return channelID, nil
}

func (repo *PgxChatRepository) GetChannels() (channels []Channel, err error) {
	channels = make([]Channel, 0, 8)
	rows, _ := repo.pool.Query("select id, name from channels order by name")

	for rows.Next() {
		var c Channel
		rows.Scan(&c.ID, &c.Name)
		channels = append(channels, c)
	}

	return channels, rows.Err()
}

func (repo *PgxChatRepository) PostMessage(channelID int32, authorID int32, body string) (messageID int64, err error) {
	err = repo.pool.QueryRow(
		"insert into messages(channel_id, user_id, body) values($1, $2, $3) returning id",
		channelID,
		authorID,
		body,
	).Scan(&messageID)
	if err != nil {
		return 0, err
	}

	return messageID, nil
}

func (repo *PgxChatRepository) GetMessages(channelID int32, beforeMessageID int32, maxCount int32) (messages []Message, err error) {
	messages = make([]Message, 0, 8)
	rows, _ := repo.pool.Query(`
		select id, user_id, body, creation_time
		from messages
		where channel_id=$1
		order by id desc
	`,
		channelID,
	)

	for rows.Next() {
		var m Message
		rows.Scan(&m.ID, &m.AuthorID, &m.Body, &m.Time)
		messages = append(messages, m)
	}

	return messages, rows.Err()
}
