(function() {
  "use strict";

  var gotoLogin = function() {
    window.router = new App.Router
    router.navigate("login")
    router.start()
  }

  document.addEventListener("DOMContentLoaded", function() {
    window.conn = new Connection
    conn.opened.add(function() {
      var sessionID = localStorage.getItem("sessionID")
      if(sessionID) {
        conn.resumeSession(sessionID, {
          succeeded: function() {
            conn.initChat({
              succeeded: function(data) {
                window.chat = new App.Models.Chat(conn, data)

                window.router = new App.Router
                router.navigate('home')
                router.start()
              }
            })
          },
          failed: gotoLogin
        })
      } else {
        gotoLogin()
      }
    })

    return new App.Views.WorkingNotice;
  });
})();
