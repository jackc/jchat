package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	log "gopkg.in/inconshreveable/log15.v2"
	"net"
)

type ClientConn struct {
	ws     *websocket.Conn
	user   User
	repo   Repository
	logger log.Logger
	mailer Mailer
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
	UserID    int32  `json:"userID"`
	SessionID string `json:"sessionID"`
}

const JSONRPCInvalidRequest = -32600
const JSONRPCParseError = -32700
const JSONRPCMethodNotFound = -32601
const JSONRPCInvalidParams = -32602

// TODO - separate handling of logged in / not logged in
func (conn *ClientConn) Dispatch() {
	chatChan := make(chan Message)
	conn.repo.MessagePostedSignal().Add(chatChan)
	defer conn.repo.MessagePostedSignal().Remove(chatChan)

	userCreatedChan := make(chan User)
	conn.repo.UserCreatedSignal().Add(userCreatedChan)
	defer conn.repo.UserCreatedSignal().Remove(userCreatedChan)

	reqChan := make(chan Request)
	errChan := make(chan error)

	go func() {
		var req Request

		for {
			err := websocket.JSON.Receive(conn.ws, &req)
			if err != nil {
				errChan <- err
				return
			}

			reqChan <- req
		}
	}()

	for {
		select {
		case req := <-reqChan:
			var response Response

			switch req.Method {
			case "register":
				response = conn.Register(req.Params)
			case "login":
				response = conn.Login(req.Params)
			case "logout":
				conn.user = User{}
			case "resume_session":
				response = conn.ResumeSession(req.Params)
			case "request_password_reset":
				response = conn.RequestPasswordReset(req.Params)
			case "reset_password":
				response = conn.ResetPassword(req.Params)
			case "init_chat":
				response = conn.InitChat(req.Params)
			case "post_message":
				response = conn.PostMessage(req.Params)
			default:
				// unknown req method
				response.Error = &Error{Code: JSONRPCMethodNotFound, Message: "Method not found"}
			}

			if req.ID != nil {
				response.ID = *req.ID

				err := websocket.JSON.Send(conn.ws, response)
				if err != nil {
					fmt.Println(err)
					// Failed to send
					return
				}
			}
		case message := <-chatChan:
			var msg struct {
				ID           int64  `json:"id"`
				ChannelID    int32  `json:"channel_id"`
				AuthorID     int32  `json:"author_id"`
				Body         string `json:"body"`
				CreationTime int64  `json:"creation_time"`
			}

			msg.ID = message.ID
			msg.ChannelID = message.ChannelID
			msg.AuthorID = message.AuthorID
			msg.Body = message.Body
			msg.CreationTime = message.Time.Unix()

			var notification struct {
				Method string      `json:"method"`
				Params interface{} `json:"params"`
			}

			notification.Method = "message_posted"
			notification.Params = msg
			err := websocket.JSON.Send(conn.ws, notification)
			if err != nil {
				fmt.Println(err)
				// Failed to send
				return
			}
		case user := <-userCreatedChan:
			var msg struct {
				ID   int32  `json:"id"`
				Name string `json:"name"`
			}

			msg.ID = user.ID
			msg.Name = user.Name

			var notification struct {
				Method string      `json:"method"`
				Params interface{} `json:"params"`
			}

			notification.Method = "user_created"
			notification.Params = msg
			err := websocket.JSON.Send(conn.ws, notification)
			if err != nil {
				fmt.Println(err)
				// Failed to send
				return
			}
		case err := <-errChan:
			if _, ok := err.(*json.SyntaxError); ok {
				var response Response
				response.ID = json.Number("null")
				response.Error = &Error{
					Code:    JSONRPCParseError,
					Message: err.Error(),
				}

				err = websocket.JSON.Send(conn.ws, response)
				if err != nil {
					fmt.Println(err)
					// Failed to send
					return
				}
			}

			fmt.Println("errChan: ", err)
			fmt.Printf("%T", err)
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

	conn.user, err = conn.repo.CreateUser(registration.Name, registration.Email, registration.Password)
	if err != nil {
		if err, ok := err.(DuplicationError); ok {
			response.Error = &Error{Code: 2, Message: "Already taken", Data: err.Field}
			return response
		} else {
			response.Error = &Error{Code: 3, Message: "Unable to create"}
			return response
		}
	}

	sessionID, err := conn.repo.CreateSession(conn.user.ID)
	if err != nil {
		response.Error = &Error{Code: 5, Message: "Unable to create session"}
		return response
	}

	response.Result = LoginSuccess{UserID: conn.user.ID, SessionID: sessionID}

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

	conn.user, err = conn.repo.Login(credentials.Email, credentials.Password)
	if err != nil {
		response.Error = &Error{Code: 5, Message: "Bad email or password"}
		return response
	}

	sessionID, err := conn.repo.CreateSession(conn.user.ID)
	if err != nil {
		response.Error = &Error{Code: 5, Message: "Unable to create session"}
		return response
	}

	response.Result = LoginSuccess{UserID: conn.user.ID, SessionID: sessionID}

	return response
}

func (conn *ClientConn) ResumeSession(body json.RawMessage) (response Response) {
	var credentials struct {
		SessionID string `json:"session_id"`
	}

	err := json.Unmarshal(body, &credentials)
	if err != nil {
		response.Error = &Error{Code: JSONRPCParseError, Message: "Parse error"}
		return response
	}

	if credentials.SessionID == "" {
		response.Error = &Error{Code: JSONRPCInvalidParams, Message: "Invalid params", Data: `Request must include the attribute "session_id"`}
		return response
	}

	userID, err := conn.repo.GetUserIDBySessionID(credentials.SessionID)
	if err != nil {
		response.Error = &Error{Code: 14, Message: "Cannot resume session"}
		return response
	}

	conn.user, err = conn.repo.GetUser(userID)

	response.Result = LoginSuccess{UserID: conn.user.ID, SessionID: credentials.SessionID}

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

	token, err := conn.repo.CreatePasswordResetToken(reset.Email, remoteIP)
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

	err = conn.repo.SetPasswordByToken(resetPassword.Token, resetPassword.Password, remoteIP)
	if err != nil {
		response.Error = &Error{Code: 11, Message: "Failed to update password"}
		return response
	}

	response.Result = true
	return response
}

func (conn *ClientConn) InitChat(body json.RawMessage) (response Response) {
	initJSON, err := conn.repo.GetInit(conn.user.ID)
	if err != nil {
		response.Error = &Error{Code: 12, Message: "Unable to initialize chat"}
		return response
	}

	rawInit := json.RawMessage(initJSON)
	response.Result = &rawInit
	return response
}

func (conn *ClientConn) PostMessage(body json.RawMessage) (response Response) {
	var message struct {
		ChannelID int32  `json:"channel_id"`
		Text      string `json:"text"`
	}

	err := json.Unmarshal(body, &message)
	if err != nil {
		response.Error = &Error{Code: JSONRPCParseError, Message: "Parse error"}
		return response
	}

	_, err = conn.repo.PostMessage(message.ChannelID, conn.user.ID, message.Text)
	if err != nil {
		response.Error = &Error{Code: 13, Message: "Unable to post message"}
		return response
	}

	response.Result = true
	return response
}
