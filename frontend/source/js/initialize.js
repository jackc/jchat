(function() {
  "use strict";

  document.addEventListener("DOMContentLoaded", function() {
    window.conn = new Connection
    window.conn.readyForLogin.add(function() {
      window.router = new App.Router
      window.router.navigate("login")
      window.router.start()
    })

    return new App.Views.WorkingNotice;
  });
})();
