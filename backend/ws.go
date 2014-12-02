package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	log "gopkg.in/inconshreveable/log15.v2"
	"net"
)

type ClientConn struct {
	ws       *websocket.Conn
	user     User
	userRepo UserRepository
	chatRepo ChatRepository
	logger   log.Logger
	mailer   Mailer
}

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	ID     *json.Number    `json:"id,omitempty"`
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
	UserID int32            `json:"userID"`
	Init   *json.RawMessage `json:"init"`
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
		case "request_password_reset":
			response = conn.RequestPasswordReset(req.Params)
		case "reset_password":
			response = conn.ResetPassword(req.Params)
		default:
			// unknown req method
			response.Error = &Error{Code: JSONRPCMethodNotFound, Message: "Method not found"}
		}

		response.ID = *req.ID

		err = websocket.JSON.Send(conn.ws, response)
		if err != nil {
			fmt.Println(err)
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

	initJSON, err := conn.chatRepo.GetInit(conn.user.ID)
	if err != nil {
		response.Error = &Error{Code: 12, Message: "Unable to initialize chat"}
		return response
	}

	rawInit := json.RawMessage(initJSON)
	response.Result = LoginSuccess{UserID: conn.user.ID, Init: &rawInit}

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

	initJSON, err := conn.chatRepo.GetInit(conn.user.ID)
	if err != nil {
		response.Error = &Error{Code: 12, Message: "Unable to initialize chat"}
		return response
	}

	rawInit := json.RawMessage(initJSON)
	response.Result = LoginSuccess{UserID: conn.user.ID, Init: &rawInit}

	return response
}

func (conn *ClientConn) RequestPasswordReset(body json.RawMessage) (response Response) {
	var reset struct {
		Email string `json:"email"`
	}

	err := json.Unmarshal(body, &reset)
	if err != nil {
		response.Error = &Error{Code: JSONRPCParseError, Message: "Parse error"}
		return response
	}

	if reset.Email == "" {
		response.Error = &Error{Code: JSONRPCInvalidParams, Message: "Invalid params", Data: `Request must include the attribute "email"`}
		return response
	}

	var remoteIP string
	remoteIP, _, err = net.SplitHostPort(conn.ws.Request().RemoteAddr)
	if err != nil {
		response.Error = &Error{Code: 10, Message: "Unable to get remoteIP"}
		return response
	}

	token, err := conn.userRepo.CreatePasswordResetToken(reset.Email, remoteIP)
	if err == ErrNotFound {
		response.Result = true // don't reveal whether email address is taken or not
		return response
	}
	if err != nil {
		fmt.Println(err)
		response.Error = &Error{Code: 7, Message: "Unable to create password reset token"}
		return response
	}

	if conn.mailer == nil {
		response.Error = &Error{Code: 8, Message: "Mail is not configured -- cannot send password reset email"}
		return response
	}

	err = conn.mailer.SendPasswordResetMail(reset.Email, token)
	if err != nil {
		response.Error = &Error{Code: 9, Message: "Send email failed"}
		return response
	}

	response.Result = true // don't reveal whether email address is taken or not
	return response
}

func (conn *ClientConn) ResetPassword(body json.RawMessage) (response Response) {
	var resetPassword struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}

	err := json.Unmarshal(body, &resetPassword)
	if err != nil {
		response.Error = &Error{Code: JSONRPCParseError, Message: "Parse error"}
		return response
	}

	var remoteIP string
	remoteIP, _, err = net.SplitHostPort(conn.ws.Request().RemoteAddr)
	if err != nil {
		response.Error = &Error{Code: 10, Message: "Unable to get remoteIP"}
		return response
	}

	err = conn.userRepo.SetPasswordByToken(resetPassword.Token, resetPassword.Password, remoteIP)
	if err != nil {
		response.Error = &Error{Code: 11, Message: "Failed to update password"}
		return response
	}

	response.Result = true
	return response
}
