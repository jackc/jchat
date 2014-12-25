(function() {
  "use strict"

  App.Models.Channel = function(conn, attrs) {
    this.conn = conn

    this.id = attrs.id
    this.name = attrs.name
    this.messages = attrs.messages

    this.messageReceived = new signals.Signal()

    this.sendMessage = this.sendMessage.bind(this)
  }

  App.Models.Channel.prototype = {
    unreadMessagesCount: function() {
      return 0
    },

    sendMessage: function(text) {
      this.conn.sendMessage({channel_id: this.id, text: text})
    }
  }

  App.Models.Chat = function(conn, attrs) {
    this.conn = conn

    this.channels = attrs.channels.map(function(c) {
      return new App.Models.Channel(this.conn, c)
    }.bind(this))
  }

  App.Models.Chat.prototype = {

  }
})();
