(function() {
  "use strict";

  window.Connection = function() {
    this.firstAjaxStarted = new signals.Signal();
    this.lastAjaxFinished = new signals.Signal();
  };

  Connection.prototype = {
    pendingCount: 0,

    incAjax: function() {
      this.pendingCount++;
      if (this.pendingCount === 1) {
        this.firstAjaxStarted.dispatch();
      }
    },

    decAjax: function() {
      this.pendingCount--;
      if (this.pendingCount === 0) {
        this.lastAjaxFinished.dispatch();
      }
    },

    ajax: function(url, method, options) {
      var self = this;

      if (options == null) {
        options = {};
      }

      var req = new XMLHttpRequest();
      req.open(method, url, true);

      if (State.Session.id) {
        req.setRequestHeader("X-Authentication", State.Session.id);
      }

      if (options.contentType) {
        req.setRequestHeader("Content-Type", options.contentType);
      }

      if (options.headers) {
        var headers = options.headers;
        for (k in headers) {
          v = headers[k];
          req.setRequestHeader(k, v);
        }
      }

      req.onload = function() {
        self.decAjax();
        var data = req.getResponseHeader("Content-Type") === "application/json" ? JSON.parse(req.responseText) : req.responseText;

        if (200 <= req.status && req.status <= 299 && options.succeeded) {
          options.succeeded(data, req);
          return;
        }

        if (req.status === 403 && req.responseText === "Bad or missing X-Authentication header") {
          State.Session.clear();
          window.router.navigate('login');
          return;
        }

        if (options.failed) {
          options.failed(data, req);
        }
      };

      req.onerror = function() {
        self.decAjax();
        options.failed(undefined, req);
      };

      this.incAjax();

      req.send(options.data);
    },

    get: function(url, options) {
      this.ajax(url, "GET", options);
    },

    post: function(url, options) {
      this.ajax(url, "POST", options);
    },

    patch: function(url, options) {
      this.ajax(url, "PATCH", options);
    },

    delete: function(url, options) {
      this.ajax(url, "DELETE", options);
    },

    mergeCallbacks: function(options, callbacks) {
      if (callbacks) {
        options.succeeded = callbacks.succeeded;
        options.failed = callbacks.failed;
      }

      return options;
    },

    login: function(credentials, callbacks) {
      var options = {
        contentType: "application/json",
        data: JSON.stringify(credentials)
      };

      options = this.mergeCallbacks(options, callbacks);

      this.post("/api/sessions", options);
    },

    logout: function() {
      return this.delete("/api/sessions/" + State.Session.id);
    },

    register: function(registration, callbacks) {
      var options = {
        data: JSON.stringify(registration)
      };

      options = this.mergeCallbacks(options, callbacks);

      return this.post("/api/register", options);
    },

    requestPasswordReset: function(email, callbacks) {
      var options = {
        contentType: "application/json",
        data: JSON.stringify({"email": email})
      };

      options = this.mergeCallbacks(options, callbacks);

      return this.post("/api/request_password_reset", options);
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
