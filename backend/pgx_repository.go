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
