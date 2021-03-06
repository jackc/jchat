package main

import (
	"encoding/json"
	"golang.org/x/net/websocket"
	log "gopkg.in/inconshreveable/log15.v2"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func getTestWsServer(t testing.TB, repo Repository) *httptest.Server {
	logger := log.New()
	logger.SetHandler(log.DiscardHandler())

	return httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		conn := &ClientConn{
			ws:     ws,
			repo:   repo,
			logger: logger,
			mailer: nil,
		}

		conn.Dispatch()
	}))
}

func connectWebSocketClient(t testing.TB, server *httptest.Server) *websocket.Conn {
	origin := server.URL
	url := strings.Replace(server.URL, "http", "ws", 1)
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func login(t testing.TB, ws *websocket.Conn, email, password string) {
	request := struct {
		Method string             `json:"method"`
		Params RequestCredentials `json:"params"`
		ID     int32              `json:"id"`
	}{
		Method: "login",
		Params: RequestCredentials{email, password},
		ID:     1,
	}

	err := websocket.JSON.Send(ws, &request)
	if err != nil {
		t.Fatal(err)
	}

	var response struct {
		Result LoginSuccess `json:"result"`
		Error  *Error       `json:"error,omitempty"`
		ID     int32        `json:"id"`
	}
	err = websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.ID != request.ID {
		t.Fatalf("Expected response ID (%d) to equal request ID (%d), but it did not", response.ID, request.ID)
	}
	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}
}

func TestClientConnInvalidJSON(t *testing.T) {
	repo := getPgxRepository(t)
	server := getTestWsServer(t, repo)
	defer server.Close()
	ws := connectWebSocketClient(t, server)
	defer ws.Close()

	if _, err := ws.Write([]byte("This is not JSON")); err != nil {
		t.Fatal(err)
	}

	var response struct {
		Error *Error          `json:"error"`
		ID    json.RawMessage `json:"id"`
	}
	err := websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Fatal(err)
	}
	if string(response.ID) != "null" {
		t.Fatalf("Expected ID to be null, but it was %s", string(response.ID))
	}
	if response.Error == nil {
		t.Fatal("Expected Error to be present, but it was not")
	}
	if response.Error.Code != JSONRPCParseError.Code {
		t.Fatalf("Expected Error.Code to be %d, but it was %d", JSONRPCParseError.Code, response.Error.Code)
	}
}

func TestClientConnLoginFailure(t *testing.T) {
	repo := getPgxRepository(t)
	server := getTestWsServer(t, repo)
	defer server.Close()
	ws := connectWebSocketClient(t, server)
	defer ws.Close()

	request := struct {
		Method string             `json:"method"`
		Params RequestCredentials `json:"params"`
		ID     int32              `json:"id"`
	}{
		Method: "login",
		Params: RequestCredentials{"joe@example.com", "password"},
		ID:     1,
	}

	err := websocket.JSON.Send(ws, &request)
	if err != nil {
		t.Fatal(err)
	}

	var response struct {
		Result interface{} `json:"result,omitempty"`
		Error  *Error      `json:"error,omitempty"`
		ID     int32       `json:"id"`
	}
	err = websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.ID != request.ID {
		t.Fatalf("Expected response ID (%d) to equal request ID (%d), but it did not", response.ID, request.ID)
	}
	if response.Error == nil {
		t.Fatal("Expected Error to be present, but it was not")
	}
	if response.Error.Code != JSONRPCAunthenticationError.Code {
		t.Fatalf("Expected Error.Code to be %d, but it was %d", JSONRPCAunthenticationError.Code, response.Error.Code)
	}
}

func TestClientConnLoginSuccess(t *testing.T) {
	repo := getPgxRepository(t)

	user, err := repo.CreateUser("joe", "joe@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}

	server := getTestWsServer(t, repo)
	defer server.Close()
	ws := connectWebSocketClient(t, server)
	defer ws.Close()

	request := struct {
		Method string             `json:"method"`
		Params RequestCredentials `json:"params"`
		ID     int32              `json:"id"`
	}{
		Method: "login",
		Params: RequestCredentials{"joe@example.com", "password"},
		ID:     1,
	}

	err = websocket.JSON.Send(ws, &request)
	if err != nil {
		t.Fatal(err)
	}

	var response struct {
		Result LoginSuccess `json:"result"`
		Error  *Error       `json:"error,omitempty"`
		ID     int32        `json:"id"`
	}
	err = websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.ID != request.ID {
		t.Fatalf("Expected response ID (%d) to equal request ID (%d), but it did not", response.ID, request.ID)
	}
	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}
	if response.Result.UserID != user.ID {
		t.Fatalf("Expected Result.UserID to be %d, but it was %d", user.ID, response.Result.UserID)
	}
}

func TestClientConnUnauthenticatedUserDoesNotReceiveNotifications(t *testing.T) {
	repo := getPgxRepository(t)

	server := getTestWsServer(t, repo)
	defer server.Close()
	ws := connectWebSocketClient(t, server)
	defer ws.Close()

	// CreateUser will cause user_created message to be sent to authenticated users
	user, err := repo.CreateUser("joe", "joe@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}

	channelID, err := repo.CreateChannel("General", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.PostMessage(channelID, user.ID, "Hello")
	if err != nil {
		t.Fatal(err)
	}

	err = ws.SetReadDeadline(time.Now().Add(time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	bytesRead, _ := ws.Read(buf)
	if bytesRead != 0 {
		t.Fatalf("Unauthenticated client web socket received unexpected message: %s", string(buf))
	}
}

func TestClientConnUnauthenticatedUserCannotInitChat(t *testing.T) {
	repo := getPgxRepository(t)

	server := getTestWsServer(t, repo)
	defer server.Close()
	ws := connectWebSocketClient(t, server)
	defer ws.Close()

	request := struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
		ID     int32           `json:"id"`
	}{
		Method: "init_chat",
		Params: []byte("{}"),
		ID:     1,
	}

	err := websocket.JSON.Send(ws, &request)
	if err != nil {
		t.Fatal(err)
	}

	var response struct {
		Result interface{} `json:"result,omitempty"`
		Error  *Error      `json:"error,omitempty"`
		ID     int32       `json:"id"`
	}
	err = websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.ID != request.ID {
		t.Fatalf("Expected response ID (%d) to equal request ID (%d), but it did not", response.ID, request.ID)
	}
	if response.Error == nil {
		t.Fatalf("Expected an error, but didn't get one. %#v", response)
	}
	if response.Error.Code != JSONRPCUnauthenticatedError.Code {
		t.Fatalf("Expected Error.Code to be %d, but it was %d", JSONRPCUnauthenticatedError.Code, response.Error.Code)
	}
}

func TestClientConnCreateChannel(t *testing.T) {
	repo := getPgxRepository(t)

	_, err := repo.CreateUser("joe", "joe@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}

	server := getTestWsServer(t, repo)
	defer server.Close()
	ws := connectWebSocketClient(t, server)
	defer ws.Close()

	login(t, ws, "joe@example.com", "password")

	request := struct {
		Method string        `json:"method"`
		Params CreateChannel `json:"params"`
		ID     int32         `json:"id"`
	}{
		Method: "create_channel",
		Params: CreateChannel{Name: "General"},
		ID:     1,
	}

	err = websocket.JSON.Send(ws, &request)
	if err != nil {
		t.Fatal(err)
	}

	var response struct {
		Result interface{} `json:"result,omitempty"`
		Error  *Error      `json:"error,omitempty"`
		ID     int32       `json:"id"`
	}
	err = websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.ID != request.ID {
		t.Fatalf("Expected response ID (%d) to equal request ID (%d), but it did not", response.ID, request.ID)
	}
	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}
	if response.Result != true {
		t.Fatalf("Expected Result to be %v, but it was %v", true, response.Result)
	}
}

func TestClientConnIsNotifiedChannelCreated(t *testing.T) {
	repo := getPgxRepository(t)

	user, err := repo.CreateUser("joe", "joe@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}

	server := getTestWsServer(t, repo)
	defer server.Close()
	ws := connectWebSocketClient(t, server)
	defer ws.Close()

	login(t, ws, "joe@example.com", "password")

	channelID, err := repo.CreateChannel("General", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	type channelCreated struct {
		ID   int32  `json:"id"`
		Name string `json:"name"`
	}

	err = ws.SetReadDeadline(time.Now().Add(time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	var notice struct {
		Method string         `json:"method"`
		Params channelCreated `json:"params"`
	}
	err = websocket.JSON.Receive(ws, &notice)
	if err != nil {
		t.Fatal(err)
	}

	if notice.Method != "channel_created" {
		t.Fatalf("Expected notice.Method to be %s, but it was %s", "channel_created", notice.Method)
	}
	if notice.Params.ID != channelID {
		t.Fatalf("Expected notice.Params.ID to be %d, but it was %d", channelID, notice.Params.ID)
	}
	if notice.Params.Name != "General" {
		t.Fatalf("Expected notice.Params.Name to be %s, but it was %s", "General", notice.Params.Name)
	}
}

func TestClientConnRenameChannel(t *testing.T) {
	repo := getPgxRepository(t)

	user, err := repo.CreateUser("joe", "joe@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}

	channelID, err := repo.CreateChannel("Foo", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	server := getTestWsServer(t, repo)
	defer server.Close()
	ws := connectWebSocketClient(t, server)
	defer ws.Close()

	login(t, ws, "joe@example.com", "password")

	request := struct {
		Method string        `json:"method"`
		Params RenameChannel `json:"params"`
		ID     int32         `json:"id"`
	}{
		Method: "rename_channel",
		Params: RenameChannel{ID: channelID, Name: "Bar"},
		ID:     1,
	}

	err = websocket.JSON.Send(ws, &request)
	if err != nil {
		t.Fatal(err)
	}

	var response struct {
		Result interface{} `json:"result,omitempty"`
		Error  *Error      `json:"error,omitempty"`
		ID     int32       `json:"id"`
	}
	err = websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.ID != request.ID {
		t.Fatalf("Expected response ID (%d) to equal request ID (%d), but it did not", response.ID, request.ID)
	}
	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}
	if response.Result != true {
		t.Fatalf("Expected Result to be %v, but it was %v", true, response.Result)
	}

	channels, err := repo.GetChannels()
	if err != nil {
		t.Fatalf("repo.GetChannels returned error: %v", err)
	}
	if len(channels) != 1 {
		t.Errorf("Expected repo.GetChannels to return %d channels, but it was %d", 1, len(channels))
	}
	if channels[0].ID != channelID {
		t.Errorf("Expected channel to have ID %d, but it was %d", channelID, channels[0].ID)
	}
	if channels[0].Name != "Bar" {
		t.Errorf("Expected channel to have name %s, but it was %s", "Test", channels[0].Name)
	}
}

func TestClientConnIsNotifiedChannelRenamed(t *testing.T) {
	repo := getPgxRepository(t)

	user, err := repo.CreateUser("joe", "joe@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}

	channelID, err := repo.CreateChannel("Foo", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	server := getTestWsServer(t, repo)
	defer server.Close()
	ws := connectWebSocketClient(t, server)
	defer ws.Close()

	login(t, ws, "joe@example.com", "password")

	err = repo.RenameChannel(channelID, "Bar")
	if err != nil {
		t.Fatal(err)
	}

	type channelRenamed struct {
		ID   int32  `json:"id"`
		Name string `json:"name"`
	}

	err = ws.SetReadDeadline(time.Now().Add(time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	var notice struct {
		Method string         `json:"method"`
		Params channelRenamed `json:"params"`
	}
	err = websocket.JSON.Receive(ws, &notice)
	if err != nil {
		t.Fatal(err)
	}

	if notice.Method != "channel_renamed" {
		t.Fatalf("Expected notice.Method to be %s, but it was %s", "channel_renamed", notice.Method)
	}
	if notice.Params.ID != channelID {
		t.Fatalf("Expected notice.Params.ID to be %d, but it was %d", channelID, notice.Params.ID)
	}
	if notice.Params.Name != "Bar" {
		t.Fatalf("Expected notice.Params.Name to be %s, but it was %s", "Bar", notice.Params.Name)
	}
}
