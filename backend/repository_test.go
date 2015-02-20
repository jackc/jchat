package main

import (
	"testing"
	"time"
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
	createdUser, err := repo.CreateUser("tester", "tester@example.com", "secret")
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

func testUserRepositoryGetUser(t *testing.T, repo UserRepository) {
	createdUser, err := repo.CreateUser("tester", "tester@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	foundUser, err := repo.GetUser(createdUser.ID)
	if err != nil {
		t.Fatalf("repo.GetUser returned error: %v", err)
	}

	if createdUser != foundUser {
		t.Fatalf("Expected repo.GetUser to return %v, but it returned %v", createdUser, foundUser)
	}
}

func testUserRepositorySetPassword(t *testing.T, repo UserRepository) {
	user, err := repo.CreateUser("tester", "tester@example.com", "oldpassword")
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

func testUserRepositoryResetPasswordsLifeCycle(t *testing.T, repo UserRepository) {
	_, err := repo.CreatePasswordResetToken("missing@example.com", "127.0.0.1")
	if err != ErrNotFound {
		t.Fatalf("repo.CreatePasswordResetToken with invalid email should have returned ErrNotFound, but was: %v", err)
	}

	user, err := repo.CreateUser("tester", "tester@example.com", "oldpassword")
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	err = repo.SetPasswordByToken("invalidtoken", "newpassword", "127.0.0.1")
	if err != ErrNotFound {
		t.Fatalf("repo.SetPasswordByToken should have returned ErrNotFound but it returned: %v", err)
	}

	foundUser, err := repo.Login("tester@example.com", "oldpassword")
	if err != nil {
		t.Fatalf("repo.Login returned error: %v", err)
	}
	if foundUser != user {
		t.Fatalf("Wrong user returned: %v", foundUser)
	}

	token, err := repo.CreatePasswordResetToken("tester@example.com", "127.0.01")
	if err != nil {
		t.Fatalf("repo.CreatePasswordReset returned error: %v", err)
	}

	err = repo.SetPasswordByToken(token, "newpassword", "127.0.01")
	if err != nil {
		t.Fatalf("repo.SetPasswordByToken returned error: %v", err)
	}

	foundUser, err = repo.Login("tester@example.com", "newpassword")
	if err != nil {
		t.Fatalf("repo.Login returned error: %v", err)
	}
	if foundUser != user {
		t.Fatalf("Wrong user returned: %v", foundUser)
	}
}

func testSessionRepository(t *testing.T, repo SessionRepository, userID int32) {
	sessionID, err := repo.CreateSession(userID)
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	foundUserID, err := repo.GetUserIDBySessionID(sessionID)
	if err != nil {
		t.Fatalf("repo.GetUserIDBySessionID returned error: %v", err)
	}
	if userID != foundUserID {
		t.Fatalf("Expected repo.GetUserIDBySessionID to return %d, but it was %d", userID, foundUserID)
	}

	err = repo.DeleteSession(sessionID)
	if err != nil {
		t.Fatalf("repo.DeleteSession returned error: %v", err)
	}

	_, err = repo.GetUserIDBySessionID(sessionID)
	if err != ErrNotFound {
		t.Fatalf("Expected repo.GetUserIDBySessionID to return ErrNotFound, but returned error: %v", err)
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

func testMessagePostedNotifier(t *testing.T, signaler MessagePostedSignaler, repo ChatRepository, userID int32) {
	channelID, err := repo.CreateChannel("Test", userID)
	if err != nil {
		t.Fatalf("repo.CreateChannel returned error: %v", err)
	}

	var message Message
	finished := make(chan bool)

	c := make(chan Message)
	signaler.MessagePostedSignal().Add(c)
	go func() {
		message = <-c
		finished <- true
	}()

	messageID, err := repo.PostMessage(channelID, userID, "Hello, world")
	if err != nil {
		t.Fatalf("repo.PostMessage returned error: %v", err)
	}

	select {
	case <-finished:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("Never received message on channel c")
	}

	if message.ID != messageID {
		t.Errorf("Expected message.ID to be %v, but it was %v", messageID, message.ID)
	}
	if message.ChannelID != channelID {
		t.Errorf("Expected message.ChannelID to be %v, but it was %v", channelID, message.ChannelID)
	}
	if message.AuthorID != userID {
		t.Errorf("Expected message.AuthorID to be %v, but it was %v", userID, message.AuthorID)
	}
	if message.Body != "Hello, world" {
		t.Errorf("Expected message.Body to be %v, but it was %v", "Hello, world", message.Body)
	}

	signaler.MessagePostedSignal().Remove(c)

	// If the Unlisten didn't work this will hang
	_, err = repo.PostMessage(channelID, userID, "Goodbye, world")
	if err != nil {
		t.Fatalf("repo.PostMessage returned error: %v", err)
	}
}

func testUserCreatedNotifier(t *testing.T, signaler UserCreatedSignaler, repo UserRepository) {
	var notification User
	finished := make(chan bool)

	c := make(chan User)
	signaler.UserCreatedSignal().Add(c)
	go func() {
		notification = <-c
		finished <- true
	}()

	user, err := repo.CreateUser("john", "john@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.CreateUser returned error: %v", err)
	}

	select {
	case <-finished:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("Never received message on channel c")
	}

	if notification != user {
		t.Errorf("Expected notification to be %v, but it was %v", user, notification)
	}

	signaler.UserCreatedSignal().Remove(c)

	// If the Unlisten didn't work this will hang
	_, err = repo.CreateUser("mark", "mark@example.com", "secret")
	if err != nil {
		t.Fatalf("repo.CreateUser returned error: %v", err)
	}
}

func testChannelCreatedSignaler(t *testing.T, signaler ChannelCreatedSignaler, chatRepo ChatRepository, userRepo UserRepository) {
	user, err := userRepo.CreateUser("john", "john@example.com", "secret")
	if err != nil {
		t.Fatalf("userRepo.CreateUser returned error: %v", err)
	}

	var notification Channel
	finished := make(chan bool)

	c := make(chan Channel)
	signaler.ChannelCreatedSignal().Add(c)
	go func() {
		notification = <-c
		finished <- true
	}()

	channelID, err := chatRepo.CreateChannel("General", user.ID)
	if err != nil {
		t.Fatalf("chatRepo.CreateUser returned error: %v", err)
	}

	select {
	case <-finished:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("Never received message on channel c")
	}

	if notification.ID != channelID {
		t.Errorf("Expected notification.ID to be %v, but it was %v", channelID, notification.ID)
	}

	if notification.Name != "General" {
		t.Errorf("Expected notification.Name to be %v, but it was %v", "General", notification.Name)
	}

	signaler.ChannelCreatedSignal().Remove(c)

	// If the Remove didn't work this will hang
	_, err = chatRepo.CreateChannel("Random", user.ID)
	if err != nil {
		t.Fatalf("chatRepo.CreateChannel returned error: %v", err)
	}
}
