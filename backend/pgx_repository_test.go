package main

import (
	"github.com/jackc/pgx"
	"github.com/vaughan0/go-ini"
	"testing"
)

var sharedPgxConnPool *pgx.ConnPool

func getPgxConnPool(t testing.TB) *pgx.ConnPool {
	if sharedPgxConnPool == nil {
		configPath := "../jchat.test.conf"
		conf, err := ini.LoadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		pool, err := newConnPool(conf)
		if err != nil {
			t.Fatal(err)
		}

		sharedPgxConnPool = pool
	}

	mustExec := func(t testing.TB, sql string, arguments ...interface{}) (commandTag pgx.CommandTag) {
		commandTag, err := sharedPgxConnPool.Exec(sql, arguments...)
		if err != nil {
			t.Fatalf("Exec unexpectedly failed with %v: %v", sql, err)
		}

		return commandTag
	}

	mustExec(t, "delete from messages")
	mustExec(t, "delete from channels")
	mustExec(t, "delete from users")

	return sharedPgxConnPool
}

func mustExec(t testing.TB, sql string, arguments ...interface{}) (commandTag pgx.CommandTag) {
	pool := getPgxConnPool(t)

	commandTag, err := pool.Exec(sql, arguments...)
	if err != nil {
		t.Fatalf("Exec unexpectedly failed with %v: %v", sql, err)
	}

	return commandTag
}

func TestPgxUserRepositoryCreateAndLoginCycle(t *testing.T) {
	repo := NewPgxUserRepository(getPgxConnPool(t))
	testUserRepositoryCreateAndLoginCycle(t, repo)
}

func TestPgxUserRepositorySetPassword(t *testing.T) {
	repo := NewPgxUserRepository(getPgxConnPool(t))
	testUserRepositorySetPassword(t, repo)
}

func TestPgxSessionRepository(t *testing.T) {
	connPool := getPgxConnPool(t)
	userRepo := NewPgxUserRepository(connPool)
	userID, err := userRepo.Create("test", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("userRepo.Create unexpectedly failed: %v", err)
	}

	sessionRepo := NewPgxSessionRepository(connPool)
	testSessionRepository(t, sessionRepo, userID)
}

func TestPgxChatRepository(t *testing.T) {
	connPool := getPgxConnPool(t)
	userRepo := NewPgxUserRepository(connPool)
	userID, err := userRepo.Create("test", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("userRepo.Create unexpectedly failed: %v", err)
	}

	chatRepo := NewPgxChatRepository(connPool)
	testChatRepository(t, chatRepo, userID)
}
