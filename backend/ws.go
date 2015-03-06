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

	channelCreatedChan chan Channel
	channelRenamedChan chan Channel
	messagePostedChan  chan Message
	userCreatedChan    chan User
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

type RequestCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateChannel struct {
	Name string `json:"name"`
}

type RenameChannel struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// Standardized JSON-RPC errors
var JSONRPCParseError = Error{Code: -32700, Message: "Parse error"}
var JSONRPCInvalidRequest = Error{Code: -32600, Message: "Invalid Request"}
var JSONRPCMethodNotFound = Error{Code: -32601, Message: "Method not found"}
var JSONRPCInvalidParams = Error{Code: -32602, Message: "Invalid params"}

// Custom JSON-RPC errors 4000-4999
// Client errors - roughly correspond to HTTP 400-499 type errors
var JSONRPCAunthenticationError = Error{Code: 4001, Message: "Authentication error"}
var JSONRPCDuplicationError = Error{Code: 4002, Message: "Duplicate"}
var JSONRPCInvalidPasswordError = Error{Code: 4003, Message: "Invalid password"}
var JSONRPCUnauthenticatedError = Error{Code: 4004, Message: "Unauthenticated error"}

// Custom JSON-RPC errors 5000-5999
// Server errors -- roughly correspond to HTTP 500-599 type errors
var JSONRPCInternalError = Error{Code: 5000, Message: "Internal error"}
var JSONRPCSendEmailError = Error{Code: 5001, Message: "Unable to send email"}

func errorWithData(errTemplate Error, data interface{}) *Error {
	errTemplate.Data = data
	return &errTemplate
}

func (conn *ClientConn) Dispatch() {
	defer conn.removeRepositoryListeners()

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
			case "create_channel":
				response = conn.CreateChannel(req.Params)
			case "rename_channel":
				response = conn.RenameChannel(req.Params)
			case "logout":
				conn.user = User{}
			default:
				// unknown req method
				response.Error = errorWithData(JSONRPCMethodNotFound, req.Method)
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
		case channel := <-conn.channelCreatedChan:
			var msg struct {
				ID   int32  `json:"id"`
				Name string `json:"name"`
			}

			msg.ID = channel.ID
			msg.Name = channel.Name

			var notification struct {
				Method string      `json:"method"`
				Params interface{} `json:"params"`
			}

			notification.Method = "channel_created"
			notification.Params = msg
			err := websocket.JSON.Send(conn.ws, notification)
			if err != nil {
				fmt.Println(err)
				// Failed to send
				return
			}
		case channel := <-conn.channelRenamedChan:
			var msg struct {
				ID   int32  `json:"id"`
				Name string `json:"name"`
			}

			msg.ID = channel.ID
			msg.Name = channel.Name

			var notification struct {
				Method string      `json:"method"`
				Params interface{} `json:"params"`
			}

			notification.Method = "channel_renamed"
			notification.Params = msg
			err := websocket.JSON.Send(conn.ws, notification)
			if err != nil {
				fmt.Println(err)
				// Failed to send
				return
			}
		case message := <-conn.messagePostedChan:
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
		case user := <-conn.userCreatedChan:
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
				response.Error = errorWithData(JSONRPCParseError, err.Error())

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

func (conn *ClientConn) addRepositoryListeners() {
	conn.removeRepositoryListeners()

	conn.channelCreatedChan = make(chan Channel)
	conn.repo.ChannelCreatedSignal().Add(conn.channelCreatedChan)

	conn.channelRenamedChan = make(chan Channel)
	conn.repo.ChannelRenamedSignal().Add(conn.channelRenamedChan)

	conn.messagePostedChan = make(chan Message)
	conn.repo.MessagePostedSignal().Add(conn.messagePostedChan)

	conn.userCreatedChan = make(chan User)
	conn.repo.UserCreatedSignal().Add(conn.userCreatedChan)
}

func (conn *ClientConn) removeRepositoryListeners() {
	if conn.channelCreatedChan != nil {
		conn.repo.ChannelCreatedSignal().Remove(conn.channelCreatedChan)
	}

	if conn.channelRenamedChan != nil {
		conn.repo.ChannelRenamedSignal().Remove(conn.channelRenamedChan)
	}

	if conn.messagePostedChan != nil {
		conn.repo.MessagePostedSignal().Remove(conn.messagePostedChan)
	}

	if conn.userCreatedChan != nil {
		conn.repo.UserCreatedSignal().Remove(conn.userCreatedChan)
	}
}

func (conn *ClientConn) Register(params json.RawMessage) (response Response) {
	var registration struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(params, &registration); err != nil {
		response.Error = errorWithData(JSONRPCParseError, err.Error())
		return response
	}

	if registration.Name == "" {
		response.Error = errorWithData(JSONRPCInvalidParams, `Request must include the attribute "name"`)
		return response
	}

	if len(registration.Name) > 30 {
		response.Error = errorWithData(JSONRPCInvalidParams, `"name" must be less than 30 characters`)
		return response
	}

	err := ValidatePassword(registration.Password)
	if err != nil {
		response.Error = errorWithData(JSONRPCInvalidPasswordError, err.Error())
		return response
	}

	conn.user, err = conn.repo.CreateUser(registration.Name, registration.Email, registration.Password)
	if err != nil {
		if err, ok := err.(DuplicationError); ok {
			response.Error = errorWithData(JSONRPCDuplicationError, err.Field)
			return response
		} else {
			response.Error = errorWithData(JSONRPCInternalError, "Unable to create user")
			return response
		}
	}

	sessionID, err := conn.repo.CreateSession(conn.user.ID)
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Unable to create session")
		return response
	}

	conn.addRepositoryListeners()

	response.Result = LoginSuccess{UserID: conn.user.ID, SessionID: sessionID}

	return response
}

func (conn *ClientConn) Login(body json.RawMessage) (response Response) {
	var credentials RequestCredentials

	err := json.Unmarshal(body, &credentials)
	if err != nil {
		response.Error = errorWithData(JSONRPCParseError, err.Error())
		return response
	}

	if credentials.Email == "" {
		response.Error = errorWithData(JSONRPCInvalidParams, `Request must include the attribute "email"`)
		return response
	}

	if credentials.Password == "" {
		response.Error = errorWithData(JSONRPCInvalidParams, `Request must include the attribute "password"`)
		return response
	}

	conn.user, err = conn.repo.Login(credentials.Email, credentials.Password)
	if err != nil {
		response.Error = errorWithData(JSONRPCAunthenticationError, "Bad email or password")
		return response
	}

	sessionID, err := conn.repo.CreateSession(conn.user.ID)
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Unable to create session")
		return response
	}

	conn.addRepositoryListeners()

	response.Result = LoginSuccess{UserID: conn.user.ID, SessionID: sessionID}

	return response
}

func (conn *ClientConn) ResumeSession(body json.RawMessage) (response Response) {
	var credentials struct {
		SessionID string `json:"session_id"`
	}

	err := json.Unmarshal(body, &credentials)
	if err != nil {
		response.Error = errorWithData(JSONRPCParseError, err.Error())
		return response
	}

	if credentials.SessionID == "" {
		response.Error = errorWithData(JSONRPCInvalidParams, `Request must include the attribute "session_id"`)
		return response
	}

	userID, err := conn.repo.GetUserIDBySessionID(credentials.SessionID)
	if err == ErrNotFound {
		response.Error = errorWithData(JSONRPCAunthenticationError, "Invalid sessionID")
		return response
	}
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Cannot resume session")
		return response
	}

	conn.user, err = conn.repo.GetUser(userID)

	conn.addRepositoryListeners()

	response.Result = LoginSuccess{UserID: conn.user.ID, SessionID: credentials.SessionID}

	return response
}

func (conn *ClientConn) RequestPasswordReset(body json.RawMessage) (response Response) {
	var reset struct {
		Email string `json:"email"`
	}

	err := json.Unmarshal(body, &reset)
	if err != nil {
		response.Error = errorWithData(JSONRPCParseError, err.Error())
		return response
	}

	if reset.Email == "" {
		response.Error = errorWithData(JSONRPCInvalidParams, `Request must include the attribute "email"`)
		return response
	}

	var remoteIP string
	remoteIP, _, err = net.SplitHostPort(conn.ws.Request().RemoteAddr)
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Unable to get remoteIP")
		return response
	}

	token, err := conn.repo.CreatePasswordResetToken(reset.Email, remoteIP)
	if err == ErrNotFound {
		response.Result = true // don't reveal whether email address is taken or not
		return response
	}
	if err != nil {
		fmt.Println(err)
		response.Error = errorWithData(JSONRPCInternalError, "Unable to create password reset token")
		return response
	}

	if conn.mailer == nil {
		response.Error = errorWithData(JSONRPCSendEmailError, "Mail is not configured")
		return response
	}

	err = conn.mailer.SendPasswordResetMail(reset.Email, token)
	if err != nil {
		response.Error = errorWithData(JSONRPCSendEmailError, "Send email failed")
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
		response.Error = errorWithData(JSONRPCParseError, err.Error())
		return response
	}

	var remoteIP string
	remoteIP, _, err = net.SplitHostPort(conn.ws.Request().RemoteAddr)
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Unable to get remoteIP")
		return response
	}

	err = conn.repo.SetPasswordByToken(resetPassword.Token, resetPassword.Password, remoteIP)
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Failed to update password")
		return response
	}

	response.Result = true
	return response
}

func (conn *ClientConn) InitChat(body json.RawMessage) (response Response) {
	if conn.user.ID == 0 {
		response.Error = &JSONRPCUnauthenticatedError
		return response
	}

	initJSON, err := conn.repo.GetInit(conn.user.ID)
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Unable to initialize chat")
		return response
	}

	rawInit := json.RawMessage(initJSON)
	response.Result = &rawInit
	return response
}

func (conn *ClientConn) PostMessage(body json.RawMessage) (response Response) {
	if conn.user.ID == 0 {
		response.Error = &JSONRPCUnauthenticatedError
		return response
	}

	var message struct {
		ChannelID int32  `json:"channel_id"`
		Text      string `json:"text"`
	}

	err := json.Unmarshal(body, &message)
	if err != nil {
		response.Error = errorWithData(JSONRPCParseError, err.Error())
		return response
	}

	_, err = conn.repo.PostMessage(message.ChannelID, conn.user.ID, message.Text)
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Unable to post message")
		return response
	}

	response.Result = true
	return response
}

func (conn *ClientConn) CreateChannel(body json.RawMessage) (response Response) {
	if conn.user.ID == 0 {
		response.Error = &JSONRPCUnauthenticatedError
		return response
	}

	var message CreateChannel

	err := json.Unmarshal(body, &message)
	if err != nil {
		response.Error = errorWithData(JSONRPCParseError, err.Error())
		return response
	}

	_, err = conn.repo.CreateChannel(message.Name, conn.user.ID)
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Unable to create channel")
		return response
	}

	response.Result = true
	return response
}

func (conn *ClientConn) RenameChannel(body json.RawMessage) (response Response) {
	if conn.user.ID == 0 {
		response.Error = &JSONRPCUnauthenticatedError
		return response
	}

	var message RenameChannel

	err := json.Unmarshal(body, &message)
	if err != nil {
		response.Error = errorWithData(JSONRPCParseError, err.Error())
		return response
	}

	err = conn.repo.RenameChannel(message.ID, message.Name)
	if err != nil {
		response.Error = errorWithData(JSONRPCInternalError, "Unable to rename channel")
		return response
	}

	response.Result = true
	return response
}
