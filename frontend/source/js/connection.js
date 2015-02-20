(function() {
  "use strict"

  window.Connection = function() {
    this.opened = new signals.Signal();
    this.lost = new signals.Signal();

    this.firstRequestStarted = new signals.Signal()
    this.lastRequestFinished = new signals.Signal()
    this.channelCreated = new signals.Signal()
    this.messagePosted = new signals.Signal()
    this.userCreated = new signals.Signal()

    this.wsOnMessage = this.wsOnMessage.bind(this)
    this.wsOnClose = this.wsOnClose.bind(this)
    this.wsOnError = this.wsOnError.bind(this)
    this.wsOnOpen = this.wsOnOpen.bind(this)

    this.connect()
  }

  Connection.prototype = {
    connectAttemptCount: 0,
    nextRequestID: 0,

    hostRelativeWsURI: function(path) {
      var l = window.location
      var wsURI

      if(l.protocol == "https:") {
        wsURI = "wss:"
      } else {
        wsURI = "ws:"
      }

      wsURI += "//" + l.host
      wsURI += path

      return wsURI
    },

    connect: function() {
      console.log("Attempting connection")
      this.ws = new WebSocket(this.hostRelativeWsURI("/ws"))
      this.ws.onmessage = this.wsOnMessage
      this.ws.onclose = this.wsOnClose
      this.ws.onerror = this.wsOnError
      this.ws.onopen = this.wsOnOpen

      this.pendingRequests = {}
    },

    wsOnMessage: function(evt) {
      var data = JSON.parse(evt.data)

      if(typeof data.id !== 'undefined') {
        this.onResponse(data)
      } else {
        this.onNotification(data)
      }
    },

    wsOnClose: function() {
      console.log("close")
      this.lost.dispatch()

      setTimeout(this.connect.bind(this), this.connectAttemptCount * 1000)
      this.connectAttemptCount++
    },

    wsOnError: function(err) {
      console.log("error", err)
    },

    wsOnOpen: function() {
      console.log("open")
      this.connectAttemptCount = 0
      this.opened.dispatch()
    },

    onResponse: function(response) {
      var id = response.id
      var callbacks = this.pendingRequests[id]
      if(callbacks) {
        if(response.result && callbacks.succeeded) {
          callbacks.succeeded(response.result)
        }
        if(response.error && callbacks.failed) {
          callbacks.failed(response.error)
        }
      }

      delete this.pendingRequests[id]

      if(Object.keys(this.pendingRequests).length === 0) {
        this.lastRequestFinished.dispatch()
      }
    },

    onNotification: function(notification) {
      switch(notification.method) {
        case "channel_created":
          this.channelCreated.dispatch(notification.params)
          break
        case "message_posted":
          this.messagePosted.dispatch(notification.params)
          break
        case "user_created":
          this.userCreated.dispatch(notification.params)
          break
        default:
          console.log("Unknown notification:", notification)
      }
    },

    sendNotification: function(method, params) {
      var msg = {method: method, params: params}
      this.ws.send(JSON.stringify(msg))
    },

    sendRequest: function(method, params, callbacks) {
      if(Object.keys(this.pendingRequests).length === 0) {
        this.firstRequestStarted.dispatch()
      }

      var requestID = this.nextRequestID++
      var msg = {method: method, params: params, id: requestID}
      this.pendingRequests[requestID] = callbacks

      this.ws.send(JSON.stringify(msg))
    },

    onSessionStart: function(data) {
      this.userID = data.userID
      this.sessionID = data.sessionID
      localStorage.setItem("sessionID", this.sessionID)
    },

    login: function(credentials, callbacks) {
      if(typeof callbacks === 'undefined') {
        callbacks = {}
      }
      if(callbacks.succeeded) {
        var f = callbacks.succeeded
        callbacks.succeeded = function(data) {
          this.onSessionStart(data)
          f(data)
        }.bind(this)
      } else {
        callbacks.succeeded = this.onSessionStart.bind(this)
      }
      this.sendRequest("login", credentials, callbacks)
    },

    logout: function() {
      this.sendNotification("logout")
      delete this.userID
      delete this.sessionID
      localStorage.removeItem("sessionID")
    },

    register: function(registration, callbacks) {
      this.sendRequest("register", registration, callbacks)
    },

    resumeSession: function(sessionID, callbacks) {
      this.sendRequest("resume_session", {session_id: sessionID}, callbacks)
    },

    requestPasswordReset: function(email, callbacks) {
      this.sendRequest("request_password_reset", {"email": email}, callbacks)
    },

    resetPassword: function(reset, callbacks) {
      this.sendRequest("reset_password", reset, callbacks)
    },

    initChat: function(callbacks) {
      this.sendRequest("init_chat", {}, callbacks)
    },

    sendMessage: function(message, callbacks) {
      this.sendRequest("post_message", message, callbacks)
    },

    createChannel: function(channel, callbacks) {
      this.sendRequest("create_channel", channel, callbacks)
    }
  }
})()
