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
	createdUser, err := repo.Create("tester", "tester@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	foundUser, err := repo.Login("tester@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Login returned error: %v", err)
	}

	if createdUser != foundUser {
		t.Fatalf("Expected repo.Login to return %v, but it returned %v", createdUser, foundUser)
	}
}

func testUserRepositorySetPassword(t *testing.T, repo UserRepository) {
	user, err := repo.Create("tester", "tester@example.com", "oldpassword")
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	err = repo.SetPassword(user.ID, "newpassword")
	if err != nil {
		t.Fatalf("repo.SetPassword returned error: %v", err)
	}

	foundUser, err := repo.Login("tester@example.com", "oldpassword")
	if err != ErrNotFound {
		t.Fatalf("repo.Login should have returned error ErrNotFound, but it returned: %v", err)
	}
	if foundUser != user {
		t.Fatalf("repo.Login should not have returned user when password was wrong", err)
	}

	foundUser, err = repo.Login("tester@example.com", "newpassword")
	if err != nil {
		t.Fatalf("repo.Login returned error: %v", err)
	}

	if user != foundUser {
		t.Fatalf("Expected repo.Login to return %v, but it returned %v", user, foundUser)
	}
}

func testChatRepository(t *testing.T, repo ChatRepository, userID int32) {
	channels, err := repo.GetChannels()
	if err != nil {
		t.Fatalf("repo.GetChannels returned error: %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("Expected repo.GetChannels to return %d channels, but it was %d", 0, len(channels))
	}

	channelID, err := repo.CreateChannel("Test", userID)
	if err != nil {
		t.Fatalf("repo.CreateChannel returned error: %v", err)
	}

	channels, err = repo.GetChannels()
	if err != nil {
		t.Fatalf("repo.GetChannels returned error: %v", err)
	}
	if len(channels) != 1 {
		t.Errorf("Expected repo.GetChannels to return %d channels, but it was %d", 1, len(channels))
	}
	if channels[0].ID != channelID {
		t.Errorf("Expected channel to have ID %d, but it was %d", channelID, channels[0].ID)
	}
	if channels[0].Name != "Test" {
		t.Errorf("Expected channel to have name %s, but it was %s", "Test", channels[0].Name)
	}

	messages, err := repo.GetMessages(channelID, -1, 100)
	if err != nil {
		t.Fatalf("repo.GetMessages returned error: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected repo.GetMessages to return %d messages, but it was %d", 0, len(messages))
	}

	messageID, err := repo.PostMessage(channelID, userID, "Hello, world")
	if err != nil {
		t.Fatalf("repo.PostMessage returned error: %v", err)
	}

	messages, err = repo.GetMessages(channelID, -1, 100)
	if err != nil {
		t.Fatalf("repo.GetMessages returned error: %v", err)
	}
	if len(channels) != 1 {
		t.Errorf("Expected repo.GetMessages to return %d messages, but it was %d", 1, len(messages))
	}
	if messages[0].ID != messageID {
		t.Errorf("Expect message to have ID %d, but it was %d", messageID, messages[0].ID)
	}
	if messages[0].AuthorID != userID {
		t.Errorf("Expect message to have AuthorID %d, but it was %d", userID, messages[0].AuthorID)
	}
	if messages[0].Body != "Hello, world" {
		t.Errorf("Expect message to have Body %s, but it was %s", "Hello, world", messages[0].Body)
	}
}
