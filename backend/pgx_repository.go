package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx"
	"github.com/vaughan0/go-ini"
	"io"
	"strconv"
	"sync"
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

type PgxRepository struct {
	pool      *pgx.ConnPool
	listeners [](chan Message)
	mutex     sync.Mutex
}

func NewPgxRepository(pool *pgx.ConnPool) *PgxRepository {
	return &PgxRepository{pool: pool}
}

func (repo *PgxRepository) CreateUser(name, email, password string) (user User, err error) {
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

func (repo *PgxRepository) GetUser(userID int32) (user User, err error) {
	err = repo.pool.QueryRow("select id, name, email from users where id=$1",
		userID,
	).Scan(&user.ID, &user.Name, &user.Email)
	if err == pgx.ErrNoRows {
		return user, ErrNotFound
	}
	if err != nil {
		return user, err
	}

	return user, nil
}

func (repo *PgxRepository) Login(email, password string) (user User, err error) {
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

func (repo *PgxRepository) SetPassword(userID int32, password string) (err error) {
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

func (repo *PgxRepository) CreatePasswordResetToken(email string, requestIP string) (token string, err error) {
	var userID int32
	err = repo.pool.QueryRow("select id from users where email=$1",
		email,
	).Scan(&userID)
	if err == pgx.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	tokenBytes := make([]byte, 16)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}

	token = hex.EncodeToString(tokenBytes)

	_, err = repo.pool.Exec("insert into password_resets(token, user_id, request_ip, request_time) values($1, $2, $3, current_timestamp)",
		token, userID, requestIP)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (repo *PgxRepository) SetPasswordByToken(token, password string, completionIP string) error {
	digest, salt, err := DigestPassword(password)
	if err != nil {
		return err
	}

	commandTag, err := repo.pool.Exec(`
		with t as (
			update password_resets
			set completion_ip=$1,
			  completion_time=current_timestamp
			where token=$2
			  and completion_time is null
			returning user_id
		)
		update users
		set password_digest=$3,
		  password_salt=$4
		from t
		where users.id=t.user_id
		`,
		completionIP,
		token,
		digest,
		salt,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (repo *PgxRepository) CreateSession(userID int32) (sessionID string, err error) {
	sessionBytes := make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, sessionBytes)
	if err != nil {
		return "", err
	}

	_, err = repo.pool.Exec(`insert into sessions(id, user_id) values($1, $2)`, sessionBytes, userID)
	if err != nil {
		return "", err
	}

	sessionID = hex.EncodeToString(sessionBytes)

	return sessionID, err
}

func (repo *PgxRepository) DeleteSession(sessionID string) (err error) {
	sessionBytes, err := hex.DecodeString(sessionID)
	if err != nil {
		return err
	}

	commandTag, err := repo.pool.Exec(`delete from sessions where id=$1`, sessionBytes)
	if err != nil {
		return err
	}
	if commandTag != "DELETE 1" {
		return ErrNotFound
	}

	return nil
}

func (repo *PgxRepository) GetUserIDBySessionID(sessionID string) (userID int32, err error) {
	sessionBytes, err := hex.DecodeString(sessionID)
	if err != nil {
		return 0, err
	}

	err = repo.pool.QueryRow("select user_id from sessions where id=$1", sessionBytes).Scan(&userID)
	if err == pgx.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func (repo *PgxRepository) CreateChannel(name string, userID int32) (channelID int32, err error) {
	err = repo.pool.QueryRow(
		"insert into channels(name) values($1) returning id",
		name,
	).Scan(&channelID)
	if err != nil {
		return 0, err
	}

	return channelID, nil
}

func (repo *PgxRepository) GetChannels() (channels []Channel, err error) {
	channels = make([]Channel, 0, 8)
	rows, _ := repo.pool.Query("select id, name from channels order by name")

	for rows.Next() {
		var c Channel
		rows.Scan(&c.ID, &c.Name)
		channels = append(channels, c)
	}

	return channels, rows.Err()
}

func (repo *PgxRepository) PostMessage(channelID int32, authorID int32, body string) (messageID int64, err error) {
	message := Message{ChannelID: channelID, AuthorID: authorID, Body: body}
	err = repo.pool.QueryRow(
		"insert into messages(channel_id, user_id, body) values($1, $2, $3) returning id, creation_time",
		channelID,
		authorID,
		body,
	).Scan(&message.ID, &message.Time)
	if err != nil {
		return 0, err
	}

	go func() {
		repo.mutex.Lock()
		defer repo.mutex.Unlock()

		for _, l := range repo.listeners {
			l <- message
		}
	}()

	return message.ID, nil
}

func (repo *PgxRepository) GetMessages(channelID int32, beforeMessageID int32, maxCount int32) (messages []Message, err error) {
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

func (repo *PgxRepository) GetInit(userID int32) (json []byte, err error) {
	err = repo.pool.QueryRow(`
		select row_to_json(t)
		from (
		  select coalesce(json_agg(row_to_json(t)), '[]'::json) as channels
		  from (
		    select
		      id,
		      name,
		      (
		        select coalesce(json_agg(row_to_json(t)), '[]'::json)
		        from (
		          select
		            id,
		            user_id,
		            body,
		            extract(epoch from creation_time::timestamptz(0)) as creation_time
		          from messages
		          where messages.channel_id=channels.id
		          order by creation_time asc
		        ) t
		      ) messages
		    from channels
		  ) t
		) t
	`).Scan(&json)

	return json, err
}

func (repo *PgxRepository) Listen() chan Message {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	c := make(chan Message)
	repo.listeners = append(repo.listeners, c)
	return c
}

func (repo *PgxRepository) Unlisten(c chan Message) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	for i, l := range repo.listeners {
		if c == l {
			repo.listeners[i] = repo.listeners[len(repo.listeners)-1]
			repo.listeners = repo.listeners[:len(repo.listeners)-1]
			return
		}
	}
}
