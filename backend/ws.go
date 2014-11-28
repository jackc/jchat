package main

import (
	"encoding/json"
	"golang.org/x/net/websocket"
	log "gopkg.in/inconshreveable/log15.v2"
)

type ClientConn struct {
	ws       *websocket.Conn
	user     User
	userRepo UserRepository
	logger   log.Logger
}

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	ID     json.Number     `json:"id"`
}

type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
	ID     json.Number `json:"id"`
}

type Error struct {
	Code    int32       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type LoginSuccess struct {
	Name string `json:"name"`
}

const JSONRPCInvalidRequest = -32600
const JSONRPCParseError = -32700
const JSONRPCMethodNotFound = -32601
const JSONRPCInvalidParams = -32602

func (conn *ClientConn) Dispatch() {
	var req Request

	for {
		err := websocket.JSON.Receive(conn.ws, &req)
		if err != nil {
			return
		}

		var response Response

		switch req.Method {
		case "register":
			response = conn.Register(req.Params)
		case "login":
			response = conn.Login(req.Params)
		default:
			// unknown req method
			response.Error = &Error{Code: JSONRPCMethodNotFound, Message: "Method not found"}
		}

		response.ID = req.ID

		err = websocket.JSON.Send(conn.ws, response)
		if err != nil {
			// Failed to send
			return
		}

	}
}

func (conn *ClientConn) Register(params json.RawMessage) (response Response) {
	var registration struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(params, &registration); err != nil {
		response.Error = &Error{Code: JSONRPCParseError, Message: "Parse error"}
		return response
	}

	if registration.Name == "" {
		response.Error = &Error{Code: JSONRPCInvalidParams, Message: "Invalid params", Data: `Request must include the attribute "name"`}
		return response
	}

	if len(registration.Name) > 30 {
		response.Error = &Error{Code: JSONRPCInvalidParams, Message: "Invalid params", Data: `"name" must be less than 30 characters`}
		return response
	}

	err := ValidatePassword(registration.Password)
	if err != nil {
		response.Error = &Error{Code: 1, Message: "Invalid password", Data: err.Error()}
		return response
	}

	conn.user, err = conn.userRepo.Create(registration.Name, registration.Email, registration.Password)
	if err != nil {
		if err, ok := err.(DuplicationError); ok {
			response.Error = &Error{Code: 2, Message: "Already taken", Data: err.Field}
			return response
		} else {
			response.Error = &Error{Code: 3, Message: "Unable to create"}
			return response
		}
	}

	response.Result = LoginSuccess{Name: registration.Name}
	return response
}

func (conn *ClientConn) Login(body json.RawMessage) (response Response) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.Unmarshal(body, &credentials)
	if err != nil {
		response.Error = &Error{Code: JSONRPCParseError, Message: "Parse error"}
		return response
	}

	if credentials.Email == "" {
		response.Error = &Error{Code: JSONRPCInvalidParams, Message: "Invalid params", Data: `Request must include the attribute "email"`}
		return response
	}

	if credentials.Password == "" {
		response.Error = &Error{Code: JSONRPCInvalidParams, Message: "Invalid params", Data: `Request must include the attribute "password"`}
		return response
	}

	conn.user, err = conn.userRepo.Login(credentials.Email, credentials.Password)
	if err != nil {
		response.Error = &Error{Code: 5, Message: "Bad email or password"}
		return response
	}

	response.Result = LoginSuccess{Name: conn.user.Name}
	return response
}
