package main

import (
	"testing"
)

func TestPasswordDigestAndPasswordMatch(t *testing.T) {
	t.Parallel()

	digest, salt, err := DigestPassword("secret")
	if err != nil {
		t.Fatalf("DigestPassword returned error: %v", err)
	}

	if !PasswordMatch("secret", digest, salt) {
		t.Fatal("PasswordMatch should have been true with correct password, but it wasn't")
	}

	if PasswordMatch("wrong", digest, salt) {
		t.Fatal("PasswordMatch should have been false with incorrect password, but it wasn't")
	}
}

func testUserRepositoryCreateAndLoginCycle(t *testing.T, repo UserRepository) {
	createdUserID, err := repo.Create("tester", "tester@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	foundUserID, err := repo.Login("tester@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Login returned error: %v", err)
	}

	if createdUserID != foundUserID {
		t.Fatalf("Expected repo.Login to return %v, but it returned %v", createdUserID, foundUserID)
	}
}

func testUserRepositorySetPassword(t *testing.T, repo UserRepository) {
	userID, err := repo.Create("tester", "tester@example.com", "oldpassword")
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	err = repo.SetPassword(userID, "newpassword")
	if err != nil {
		t.Fatalf("repo.SetPassword returned error: %v", err)
	}

	foundUserID, err := repo.Login("tester@example.com", "oldpassword")
	if err != ErrNotFound {
		t.Fatalf("repo.Login should have returned error ErrNotFound, but it returned: %v", err)
	}
	if foundUserID == userID {
		t.Fatalf("repo.Login should not have returned userID when password was wrong", err)
	}

	foundUserID, err = repo.Login("tester@example.com", "newpassword")
	if err != nil {
		t.Fatalf("repo.Login returned error: %v", err)
	}

	if userID != foundUserID {
		t.Fatalf("Expected repo.Login to return %v, but it returned %v", userID, foundUserID)
	}
}

func testSessionRepository(t *testing.T, repo SessionRepository, userID int32) {
	sessionID, err := repo.Create(Session{UserID: userID})
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	session, err := repo.GetSession(sessionID)
	if err != nil {
		t.Fatalf("repo.GetUserIDBySessionID returned error: %v", err)
	}
	if userID != session.UserID {
		t.Fatalf("Expected repo.GetSession to return session.UserID %v, but it was %d", userID, session.UserID)
	}

	err = repo.Delete(sessionID)
	if err != nil {
		t.Fatalf("repo.Delete returned error: %v", err)
	}

	_, err = repo.GetSession(sessionID)
	if err != ErrNotFound {
		t.Fatalf("Expected repo.GetUserIDBySessionID to return ErrNotFound, but returned error: %v", err)
	}
}
