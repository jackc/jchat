package main

import (
	"errors"
	"github.com/jackc/pgx"
	"github.com/vaughan0/go-ini"
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

func (repo *PgxUserRepository) Create(name, email, password string) (user User, err error) {
	digest, salt, err := DigestPassword(password)
	if err != nil {
		return user, err
	}

	err = repo.pool.QueryRow(
		"insert into users(name, email, password_digest, password_salt) values($1, $2, $3, $4) returning id, name, email",
		name,
		email,
		digest,
		salt,
	).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		return user, err
	}

	return user, nil
}

func (repo *PgxUserRepository) Login(email, password string) (user User, err error) {
	var digest, salt []byte

	err = repo.pool.QueryRow("select id, name, email, password_digest, password_salt from users where email=$1",
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &digest, &salt)
	if err == pgx.ErrNoRows {
		return user, ErrNotFound
	}
	if err != nil {
		return user, err
	}

	if !PasswordMatch(password, digest, salt) {
		return user, ErrNotFound
	}

	return user, nil
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
