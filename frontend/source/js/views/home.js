(function() {
  "use strict"

  App.Views.HomePage = function(options) {
    view.View.call(this, "div")
    this.el.className = "home"

    this.chat = options.chat

    this.header = this.createChild(App.Views.LoggedInHeader)
    this.header.render()

    this.channelList = this.createChild(App.Views.ChannelList, {chat: this.chat})
    this.channelList.render()

    this.openChannel = this.createChild(App.Views.OpenChannel, {chat: this.chat})
    this.openChannel.render()
  }

  App.Views.HomePage.prototype = Object.create(view.View.prototype)

  var p = App.Views.HomePage.prototype
  p.template = JST["templates/home_page"]

  p.render = function() {
    this.el.innerHTML = ""
    this.el.appendChild(this.header.el)
    this.el.appendChild(this.channelList.el)
    this.el.appendChild(this.openChannel.el)
    return this.el
  }


  App.Views.ChannelList = function(options) {
    view.View.call(this, "ol")
    this.el.className = "channels"
    this.chat = options.chat

    this.channelSelected = this.channelSelected.bind(this)

    this.channelViews = this.chat.channels.map(function(c) {
      return this.createChild(App.Views.Channel, {chat: this.chat, channel: c})
    }, this)

    this.channelViews.forEach(function(cv) {
      cv.selected.add(this.channelSelected)
      this.el.appendChild(cv.render())
    }, this)
  }

  App.Views.ChannelList.prototype = Object.create(view.View.prototype)

  var p = App.Views.ChannelList.prototype

  p.render = function() {
    this.el.innerHTML = ""

    this.channelViews.forEach(function(cv) {
      this.el.appendChild(cv.render())
    }, this)

    return this.el
  }

  p.channelSelected = function(channelView) {
    this.chat.changeChannel(channelView.channel)
  }


  App.Views.Channel = function(options) {
    view.View.call(this, "li")

    this.chat = options.chat
    this.channel = options.channel

    this.selected = new signals.Signal()

    this.onChannelChanged = this.onChannelChanged.bind(this)
    this.chat.channelChanged.add(this.onChannelChanged)

    this.listen()
  }

  App.Views.Channel.prototype = Object.create(view.View.prototype)

  var p = App.Views.Channel.prototype

  p.template = JST["templates/channel"]

  p.render = function() {
    this.el.innerHTML = this.template(this.channel)
    if(this.channel == this.chat.selectedChannel) {
      this.el.classList.add("selected")
    }

    return this.el
  }

  p.listen = function() {
    this.el.addEventListener("click", function() { this.selected.dispatch(this) }.bind(this) )
  }

  p.onChannelChanged = function(channel) {
    if(this.channel == channel) {
      console.log("adding class")
      this.el.classList.add("selected")
    } else {
      console.log("removing class")
      this.el.classList.remove("selected")
    }
  }


  App.Views.OpenChannel = function(options) {
    view.View.call(this, "div")
    this.el.className = "openChannel"
    this.chat = options.chat
    this.users = this.chat.users
    this.channel = this.chat.selectedChannel

    this.onChannelChanged = this.onChannelChanged.bind(this)
    this.chat.channelChanged.add(this.onChannelChanged)

    this.messagesView = this.createChild(App.Views.OpenChannelMessages, {channel: this.channel, users: this.users})
    this.messagesView.render()

    this.composerView = this.createChild(App.Views.Composer, {channel: this.channel})
    this.composerView.render()
  }

  App.Views.OpenChannel.prototype = Object.create(view.View.prototype)

  var p = App.Views.OpenChannel.prototype

  p.render = function() {
    this.el.innerHTML = ""

    this.el.appendChild(this.messagesView.el)
    this.el.appendChild(this.composerView.el)

    return this.el
  }

  p.onChannelChanged = function(channel) {
    this.channel = channel

    this.removeChild(this.messagesView)
    this.removeChild(this.composerView)

    this.messagesView = this.createChild(App.Views.OpenChannelMessages, {channel: this.channel, users: this.users})
    this.messagesView.render()

    this.composerView = this.createChild(App.Views.Composer, {channel: this.channel})
    this.composerView.render()

    this.render()
  }

  App.Views.OpenChannelMessages = function(options) {
    view.View.call(this, "div")
    this.el.className = "messages"
    this.users = options.users
    this.channel = options.channel

    this.messageReceived = this.messageReceived.bind(this)
    this.channel.messageReceived.add(this.messageReceived)
  }

  App.Views.OpenChannelMessages.prototype = Object.create(view.View.prototype)

  var p = App.Views.OpenChannelMessages.prototype

  p.destructor = function() {
    // TODO - call this from somewhere...
    this.channel.messageReceived.remove(this.messageReceived)
  }

  p.render = function() {
    this.el.innerHTML = ""

    var lastMessageTime = new Date(0)

    this.channel.messages.forEach(function(m) {
      var post_time = new Date(m.creation_time * 1000)
      if(lastMessageTime.toDateString() != post_time.toDateString()) {
        var v = new App.Views.Daybreak({date: post_time})
        this.el.appendChild(v.render())
        lastMessageTime = post_time
      }

      var v = new App.Views.Message({model: m, users: this.users})
      this.el.appendChild(v.render())
    }, this)

    // run after this is put on the page
    setTimeout(function() { this.el.scrollTop = this.el.scrollHeight }.bind(this), 0);

    return this.el
  }

  p.messageReceived = function() {
    this.render()
  }

  App.Views.Daybreak = function(options) {
    view.View.call(this, "div")
    this.el.className = "daybreak"
    this.date = options.date
  }

  App.Views.Daybreak.prototype = Object.create(view.View.prototype)

  var p = App.Views.Daybreak.prototype

  p.render = function() {
    var today = new Date()
    var yesterday = new Date()
    yesterday.setDate(today.getDate() - 1)

    var daybreakString
    if(today.toDateString() == this.date.toDateString()) {
      daybreakString = "Today"
    } else if(yesterday.toDateString() == this.date.toDateString()) {
      daybreakString = "Yesterday"
    } else {
      daybreakString = this.date.toDaybreakString()
    }

    this.el.innerHTML = daybreakString
    return this.el
  }

  App.Views.Message = function(options) {
    view.View.call(this, "div")
    this.el.className = "message"
    this.model = options.model
    this.users = options.users
  }

  App.Views.Message.prototype = Object.create(view.View.prototype)

  var p = App.Views.Message.prototype

  p.template = JST["templates/message"]

  p.render = function() {
    var user
    for(var i = 0; i < this.users.length; i++) {
      if(this.users[i].id == this.model.author_id) {
        user = this.users[i]
        break
      }
    }

    var attrs = {
      author_name: user.name,
      post_time: new Date(this.model.creation_time * 1000),
      body: this.model.body
    }

    this.el.innerHTML = this.template(attrs)
    return this.el
  }

  App.Views.Composer = function(options) {
    view.View.call(this, "div")
    this.el.className = "composer"

    this.channel = options.channel

    this.onSubmit = this.onSubmit.bind(this)
    this.onKeyPress = this.onKeyPress.bind(this)
  }

  App.Views.Composer.prototype = Object.create(view.View.prototype)

  var p = App.Views.Composer.prototype

  p.template = JST["templates/composer"]

  p.render = function() {
    this.el.innerHTML = this.template(this.model)
    this.listen()
    return this.el
  }

  p.listen = function() {
    this.el.querySelector("form").addEventListener("submit", this.onSubmit)
    this.el.querySelector("form textarea").addEventListener("keypress", this.onKeyPress)
  }

  p.onSubmit = function(e) {
    e.preventDefault()
    this.submit()
  }

  p.submit = function() {
    var textarea = this.el.querySelector("textarea")
    this.channel.sendMessage(textarea.value)
    textarea.value = ""
  }

  p.onKeyPress = function(e) {
    if(e.keyCode == 13) {
      e.preventDefault()
      this.submit()
    }
  }
})()
