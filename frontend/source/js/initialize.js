(function() {
  "use strict";

  var gotoLogin = function() {
    window.router = new App.Router
    router.navigate("login")
    router.start()
  }

  document.addEventListener("DOMContentLoaded", function() {
    window.conn = new Connection
    conn.opened.addOnce(function() {
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

    conn.lost.addOnce(function() {
      console.log("Lost websocket connection")
    })

    return new App.Views.WorkingNotice;
  });
})();
