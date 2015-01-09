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

func TestPgxRepositoryCreateAndLoginCycle(t *testing.T) {
	repo := NewPgxRepository(getPgxConnPool(t))
	testUserRepositoryCreateAndLoginCycle(t, repo)
}

func TestPgxRepositoryGetUser(t *testing.T) {
	repo := NewPgxRepository(getPgxConnPool(t))
	testUserRepositoryGetUser(t, repo)
}

func TestPgxRepositorySetPassword(t *testing.T) {
	repo := NewPgxRepository(getPgxConnPool(t))
	testUserRepositorySetPassword(t, repo)
}

func TestPgxRepositorySession(t *testing.T) {
	connPool := getPgxConnPool(t)
	repo := NewPgxRepository(connPool)
	user, err := repo.CreateUser("test", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Create unexpectedly failed: %v", err)
	}

	testSessionRepository(t, repo, user.ID)
}

func TestPgxRepositoryChat(t *testing.T) {
	connPool := getPgxConnPool(t)
	repo := NewPgxRepository(connPool)
	user, err := repo.CreateUser("test", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Create unexpectedly failed: %v", err)
	}

	testChatRepository(t, repo, user.ID)
}

func TestPgxRepositoryListen(t *testing.T) {
	connPool := getPgxConnPool(t)
	repo := NewPgxRepository(connPool)
	user, err := repo.CreateUser("test", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Create unexpectedly failed: %v", err)
	}

	testChatRepositoryListen(t, repo, user.ID)
}
