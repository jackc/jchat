(function() {
  "use strict";

  window.Connection = function() {
    this.firstRequestStarted = new signals.Signal();
    this.lastRequestFinished = new signals.Signal();

    this.ws = new WebSocket(this.hostRelativeWsURI("/ws"));
    this.ws.onmessage = this.wsOnMessage.bind(this);
    this.ws.onclose = function() { console.log("socket closed"); };
    this.ws.onerror = function() { console.log("error"); };
    this.ws.onopen = function() {
      console.log("connected...");
    }.bind(this);

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
        var id = data.id
        var callbacks = this.pendingRequests[id]
        if(callbacks) {
          if(data.result && callbacks.succeeded) {
            callbacks.succeeded(data.result)
          }
          if(data.error && callbacks.failed) {
            callbacks.failed(data.error)
          }

          delete this.pendingRequests[id]

          if(Object.keys(this.pendingRequests).length === 0) {
            this.lastRequestFinished.dispatch()
          }
        }
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

    login: function(credentials, callbacks) {
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
      var options = {
        contentType: "application/json",
        data: JSON.stringify(reset)
      };

      options = this.mergeCallbacks(options, callbacks);

      this.post("/api/reset_password", options);
    }
  }
})();
