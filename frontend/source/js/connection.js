(function() {
  "use strict";

  window.Connection = function() {
    this.readyForLogin = new signals.Signal();
    this.restoredConnection = new signals.Signal();

    this.firstRequestStarted = new signals.Signal();
    this.lastRequestFinished = new signals.Signal();

    this.ws = new WebSocket(this.hostRelativeWsURI("/ws"));
    this.ws.onmessage = this.wsOnMessage.bind(this);
    this.ws.onclose = function() { console.log("socket closed"); };
    this.ws.onerror = function() { console.log("error"); };
    this.ws.onopen = this.wsOnOpen.bind(this);

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

    wsOnOpen: function() {
      this.readyForLogin.dispatch()
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

        delete this.pendingRequests[id]

        if(Object.keys(this.pendingRequests).length === 0) {
          this.lastRequestFinished.dispatch()
        }
      }
    },

    onNotification: function(notification) {
      console.log(notification)
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

    login: function(credentials, callbacks) {
      if(typeof callbacks === 'undefined') {
        callbacks = {}
      }
      if(callbacks.succeeded) {
        var f = callbacks.succeeded
        callbacks.succeeded = function(data) {
          this.userID = data.userID
          f(data)
        }.bind(this)
      } else {
        callbacks.succeeded = function(data) {
          this.userID = data.userID
        }.bind(this)
      }
      this.sendRequest("login", credentials, callbacks)
    },

    logout: function() {
      this.ws.close()
      window.conn = new Connection;
      window.router.navigate('login')
    },

    register: function(registration, callbacks) {
      this.sendRequest("register", registration, callbacks)
    },

    requestPasswordReset: function(email, callbacks) {
      this.sendRequest("request_password_reset", {"email": email}, callbacks)
    },

    resetPassword: function(reset, callbacks) {
      this.sendRequest("reset_password", reset, callbacks)
    }
  }
})();
