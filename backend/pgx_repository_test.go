package main

import (
	"github.com/jackc/pgx"
	"github.com/vaughan0/go-ini"
	"testing"
)

func getPgxRepository(t testing.TB) *PgxRepository {
	configPath := "../jchat.test.conf"
	conf, err := ini.LoadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	connPoolConfig, err := loadConnPoolConfig(conf)
	if err != nil {
		t.Fatal(err)
	}

	preparedStatements, err := loadPreparedStatements(conf)
	if err != nil {
		t.Fatal(err)
	}

	repo, err := NewPgxRepository(connPoolConfig, preparedStatements)
	if err != nil {
		t.Fatal(err)
	}

	mustExec := func(t testing.TB, sql string, arguments ...interface{}) (commandTag pgx.CommandTag) {
		commandTag, err := repo.pool.Exec(sql, arguments...)
		if err != nil {
			t.Fatalf("Exec unexpectedly failed with %v: %v", sql, err)
		}

		return commandTag
	}

	mustExec(t, "delete from messages")
	mustExec(t, "delete from channels")
	mustExec(t, "delete from users")

	return repo
}

func TestPgxRepositoryCreateAndLoginCycle(t *testing.T) {
	repo := getPgxRepository(t)
	testUserRepositoryCreateAndLoginCycle(t, repo)
}

func TestPgxRepositoryGetUser(t *testing.T) {
	repo := getPgxRepository(t)
	testUserRepositoryGetUser(t, repo)
}

func TestPgxRepositorySetPassword(t *testing.T) {
	repo := getPgxRepository(t)
	testUserRepositorySetPassword(t, repo)
}

func TestPgxRepositorySession(t *testing.T) {
	repo := getPgxRepository(t)
	user, err := repo.CreateUser("test", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Create unexpectedly failed: %v", err)
	}

	testSessionRepository(t, repo, user.ID)
}

func TestPgxRepositoryChat(t *testing.T) {
	repo := getPgxRepository(t)
	user, err := repo.CreateUser("test", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Create unexpectedly failed: %v", err)
	}

	testChatRepository(t, repo, user.ID)
}

func TestPgxRepositoryMessagePostedNotifier(t *testing.T) {
	repo := getPgxRepository(t)
	user, err := repo.CreateUser("test", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Create unexpectedly failed: %v", err)
	}

	testMessagePostedNotifier(t, repo, repo, user.ID)
}

func TestPgxRepositoryUserCreatedNotifier(t *testing.T) {
	repo := getPgxRepository(t)
	testUserCreatedNotifier(t, repo, repo)
}
