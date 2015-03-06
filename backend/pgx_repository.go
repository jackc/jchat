package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jackc/pgx"
	"github.com/vaughan0/go-ini"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func loadConnPoolConfig(conf ini.File) (pgx.ConnPoolConfig, error) {
	connConfig := pgx.ConnConfig{}

	connConfig.Host, _ = conf.Get("database", "host")
	if connConfig.Host == "" {
		return pgx.ConnPoolConfig{}, errors.New("Config must contain database.host but it does not")
	}

	if p, ok := conf.Get("database", "port"); ok {
		n, err := strconv.ParseUint(p, 10, 16)
		connConfig.Port = uint16(n)
		if err != nil {
			return pgx.ConnPoolConfig{}, err
		}
	}

	var ok bool

	if connConfig.Database, ok = conf.Get("database", "database"); !ok {
		return pgx.ConnPoolConfig{}, errors.New("Config must contain database.database but it does not")
	}
	connConfig.User, _ = conf.Get("database", "user")
	connConfig.Password, _ = conf.Get("database", "password")

	return pgx.ConnPoolConfig{
		ConnConfig:     connConfig,
		MaxConnections: 10,
	}, nil
}

func loadPreparedStatements(conf ini.File) (map[string]string, error) {
	sqlPath, _ := conf.Get("database", "sql_path")
	if sqlPath == "" {
		return nil, errors.New("Config must contain database.sql_path but it does not")
	}

	sqlPath = strings.TrimRight(sqlPath, string(filepath.Separator))

	filePaths, err := filepath.Glob(filepath.Join(sqlPath, "*.sql"))
	if err != nil {
		return nil, err
	}

	if len(filePaths) == 0 {
		absSqlPath, err := filepath.Abs(sqlPath)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Did not find any prepared statements at: %v (%v)", sqlPath, absSqlPath)
	}

	preparedStatements := make(map[string]string, len(filePaths))

	for _, p := range filePaths {
		body, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, err
		}

		preparedStatements[strings.Replace(path.Base(p), ".sql", "", 1)] = string(body)
	}

	return preparedStatements, nil
}

type PgxRepository struct {
	pool                 *pgx.ConnPool
	userCreatedSignal    UserSignal
	channelCreatedSignal ChannelSignal
	messagePostedSignal  MessageSignal
}

func NewPgxRepository(config pgx.ConnPoolConfig, preparedStatements map[string]string) (*PgxRepository, error) {
	config.AfterConnect = func(conn *pgx.Conn) error {
		for name, sql := range preparedStatements {
			_, err := conn.Prepare(name, sql)
			if err != nil {
				return err
			}
		}

		return nil
	}

	pool, err := pgx.NewConnPool(config)
	if err != nil {
		return nil, err
	}

	return &PgxRepository{pool: pool}, nil
}

func (repo *PgxRepository) MessagePostedSignal() *MessageSignal {
	return &repo.messagePostedSignal
}

func (repo *PgxRepository) UserCreatedSignal() *UserSignal {
	return &repo.userCreatedSignal
}

func (repo *PgxRepository) ChannelCreatedSignal() *ChannelSignal {
	return &repo.channelCreatedSignal
}

func (repo *PgxRepository) CreateUser(name, email, password string) (user User, err error) {
	digest, salt, err := DigestPassword(password)
	if err != nil {
		return user, err
	}

	err = repo.pool.QueryRow(
		"create_user",
		name,
		email,
		digest,
		salt,
	).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		return user, err
	}

	go repo.userCreatedSignal.Dispatch(user)

	return user, nil
}

func (repo *PgxRepository) GetUser(userID int32) (user User, err error) {
	err = repo.pool.QueryRow("get_user",
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

	err = repo.pool.QueryRow("login",
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

	commandTag, err := repo.pool.Exec("set_password", digest, salt, userID)
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
	err = repo.pool.QueryRow("get_user_by_email", email).Scan(&userID)
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

	_, err = repo.pool.Exec("create_password_reset", token, userID, requestIP)
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

	commandTag, err := repo.pool.Exec("set_password_from_password_reset",
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

	_, err = repo.pool.Exec("create_session", sessionBytes, userID)
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

	commandTag, err := repo.pool.Exec("delete_session", sessionBytes)
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

	err = repo.pool.QueryRow("get_user_id_from_session", sessionBytes).Scan(&userID)
	if err == pgx.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func (repo *PgxRepository) CreateChannel(name string, userID int32) (channelID int32, err error) {
	err = repo.pool.QueryRow("create_channel", name).Scan(&channelID)
	if err != nil {
		return 0, err
	}

	go repo.channelCreatedSignal.Dispatch(Channel{ID: channelID, Name: name})

	return channelID, nil
}

func (repo *PgxRepository) RenameChannel(channelID int32, name string) (err error) {
	commandTag, err := repo.pool.Exec("rename_channel", channelID, name)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (repo *PgxRepository) GetChannels() (channels []Channel, err error) {
	channels = make([]Channel, 0, 8)
	rows, _ := repo.pool.Query("get_channels")

	for rows.Next() {
		var c Channel
		rows.Scan(&c.ID, &c.Name)
		channels = append(channels, c)
	}

	return channels, rows.Err()
}

func (repo *PgxRepository) PostMessage(channelID int32, authorID int32, body string) (messageID int64, err error) {
	message := Message{ChannelID: channelID, AuthorID: authorID, Body: body}
	err = repo.pool.QueryRow("post_message", channelID, authorID, body).Scan(&message.ID, &message.Time)
	if err != nil {
		return 0, err
	}

	go repo.messagePostedSignal.Dispatch(message)

	return message.ID, nil
}

func (repo *PgxRepository) GetMessages(channelID int32, beforeMessageID int32, maxCount int32) (messages []Message, err error) {
	messages = make([]Message, 0, 8)
	rows, _ := repo.pool.Query("get_messages", channelID)

	for rows.Next() {
		var m Message
		rows.Scan(&m.ID, &m.AuthorID, &m.Body, &m.Time)
		messages = append(messages, m)
	}

	return messages, rows.Err()
}

func (repo *PgxRepository) GetInit(userID int32) (json []byte, err error) {
	err = repo.pool.QueryRow("get_init").Scan(&json)
	return json, err
}
