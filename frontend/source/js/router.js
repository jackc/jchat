(function() {
  "use strict";

  App.Router = function() {};

  App.Router.prototype = {
    routes: {
      login: "login",
      register: "register",
      lostPassword: "lostPassword",
      resetPassword: "resetPassword",
      home: "home"
    },

    login: function() {
      this.changePage(App.Views.LoginPage);
    },

    register: function() {
      this.changePage(App.Views.RegisterPage);
    },

    lostPassword: function() {
      this.changePage(App.Views.LostPasswordPage);
    },

    resetPassword: function() {
      this.changePage(App.Views.ResetPasswordPage);
    },

    home: function() {
      this.changePage(App.Views.HomePage);
    },

    changePage: function(pageClass, options) {
      if(this.currentPage) {
        this.currentPage.remove();
      }

      this.currentPage = new pageClass(options);
      var view = document.getElementById("view");
      view.innerHTML = "";
      view.appendChild(this.currentPage.render());
    },

    start: function() {
      var self = this;
      window.addEventListener("hashchange",
        function() { self.change() },
        false);
      this.change();
    },

    change: function() {
      var hash = window.location.hash.slice(1)
      var route = hash.split("?")[0]
      var handler = this.routes[route]
      if(!handler) {
        this.navigate("login")
        return
      }

      return this[handler]()
    },

    navigate: function(route) {
      window.location.hash = "#" + route
    }
  }
})();
