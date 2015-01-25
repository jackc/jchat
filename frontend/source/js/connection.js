(function() {
  "use strict";

  window.Connection = function() {
    this.opened = new signals.Signal();
    this.lost = new signals.Signal();

    this.firstRequestStarted = new signals.Signal()
    this.lastRequestFinished = new signals.Signal()
    this.messagePosted = new signals.Signal()
    this.userCreated = new signals.Signal()

    this.ws = new WebSocket(this.hostRelativeWsURI("/ws"))
    this.ws.onmessage = this.wsOnMessage.bind(this)
    this.ws.onclose = function() { console.log("socket closed"); }
    this.ws.onerror = this.wsOnError.bind(this)
    this.ws.onopen = this.wsOnOpen.bind(this)

    this.pendingRequests = {}
  };

  Connection.prototype = {
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

    wsOnMessage: function(evt) {
      var data = JSON.parse(evt.data)

      if(typeof data.id !== 'undefined') {
        this.onResponse(data)
      } else {
        this.onNotification(data)
      }
    },

    wsOnError: function() {
      this.lost.dispatch()
    },

    wsOnOpen: function() {
      var sessionID = localStorage.getItem("sessionID")
      if(sessionID) {
      } {
        this.opened.dispatch()
      }
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
      this.ws.close()
      window.conn = new Connection;
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
    }
  }
})();
