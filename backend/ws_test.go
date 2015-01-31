package main

import (
	"encoding/json"
	"golang.org/x/net/websocket"
	log "gopkg.in/inconshreveable/log15.v2"
	"net/http/httptest"
	"strings"
	"testing"
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
	if response.Error.Code != JSONRPCParseError {
		t.Fatalf("Expected Error.Code to be %d, but it was %d", JSONRPCParseError, response.Error.Code)
	}
}
