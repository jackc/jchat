(function() {
  "use strict"

  App.Models.Channel = function(chat, attrs) {
    this.chat = chat

    this.id = attrs.id
    this.name = attrs.name
    this.messages = attrs.messages

    this.messageReceived = new signals.Signal()

    this.sendMessage = this.sendMessage.bind(this)
    this.onMessagePosted = this.onMessagePosted.bind(this)
  }

  App.Models.Channel.prototype = {
    unreadMessagesCount: function() {
      return 0
    },

    sendMessage: function(text) {
      this.chat.postMessage({channel_id: this.id, text: text})
    },

    onMessagePosted: function(message) {
      this.messages.push(message)
      this.messageReceived.dispatch()
    }
  }

  App.Models.Chat = function(conn, attrs) {
    this.conn = conn

    this.users = attrs.users

    this.channels = attrs.channels.map(function(c) {
      return new App.Models.Channel(this, c)
    }, this)

    this.selectedChannel = this.channels[0]

    this.channelChanged = new signals.Signal()

    this.onUserCreated = this.onUserCreated.bind(this)
    this.conn.userCreated.add(this.onUserCreated)

    this.onMessagePosted = this.onMessagePosted.bind(this)
    this.conn.messagePosted.add(this.onMessagePosted)
  }

  App.Models.Chat.prototype = {
    createChannel: function(attrs, callbacks) {
      this.conn.createChannel(attrs, callbacks)
    },

    changeChannel: function(channel) {
      if(channel == this.selectedChannel) {
        return
      }

      this.selectedChannel = channel
      this.channelChanged.dispatch(channel)
    },

    postMessage: function(message) {
      this.conn.sendMessage(message)
    },

    onUserCreated: function(user) {
      this.users.push(user)
    },

    onMessagePosted: function(message) {
      for(var i = 0; i < this.channels.length; i++) {
        var c = this.channels[i]
        if(message.channel_id == c.id) {
          c.onMessagePosted(message)
          return
        }
      }
    }
  }
})();
