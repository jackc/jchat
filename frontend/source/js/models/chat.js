(function() {
  "use strict"

  App.Models.Channel = function(attrs) {
    this.name = attrs.name
    this.messages = attrs.messages
  }

  App.Models.Channel.prototype = {
    unreadMessagesCount: function() {
      return 0
    }
  }

  App.Models.Chat = function(attrs) {
    this.channels = attrs.channels.map(function(c) {
      return new App.Models.Channel(c)
    })

    this.openChannel = this.channels[0]
  }

  App.Models.Chat.prototype = {

  }
})();
