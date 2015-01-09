(function() {
  "use strict"

  App.Views.HomePage = function(options) {
    view.View.call(this, "div")
    this.el.className = "home"

    this.chat = options.chat

    this.header = this.createChild(App.Views.LoggedInHeader)
    this.header.render()

    this.channels = this.createChild(App.Views.Channels, {chat: this.chat})
    this.channels.render()

    this.openChannel = this.createChild(App.Views.OpenChannel, {channel: this.chat.channels[0], users: this.chat.users})
    this.openChannel.render()
  }

  App.Views.HomePage.prototype = Object.create(view.View.prototype)

  var p = App.Views.HomePage.prototype
  p.template = JST["templates/home_page"]

  p.render = function() {
    this.el.innerHTML = ""
    this.el.appendChild(this.header.el)
    this.el.appendChild(this.channels.el)
    this.el.appendChild(this.openChannel.el)
    return this.el
  }


  App.Views.Channels = function(options) {
    view.View.call(this, "ol")
    this.el.className = "channels"
    this.chat = options.chat
  }

  App.Views.Channels.prototype = Object.create(view.View.prototype)

  var p = App.Views.Channels.prototype

  p.render = function() {
    this.el.innerHTML = ""

    this.chat.channels.forEach(function(c) {
      var v = new App.Views.Channel({model: c})
      this.el.appendChild(v.render())
    }, this)
    return this.el
  }


  App.Views.Channel = function(options) {
    view.View.call(this, "li")

    this.model = options.model
  }

  App.Views.Channel.prototype = Object.create(view.View.prototype)

  var p = App.Views.Channel.prototype

  p.template = JST["templates/channel"]

  p.render = function() {
    this.el.innerHTML = this.template(this.model)
    this.listen()
    return this.el
  }

  p.listen = function() {
    this.el.addEventListener("click", function() { console.log("Hello")} )
  }


  App.Views.OpenChannel = function(options) {
    view.View.call(this, "div")
    this.el.className = "openChannel"
    this.users = options.users
    this.channel = options.channel

    this.composer = this.createChild(App.Views.Composer, {channel: this.channel})
    this.composer.render()

    this.messageReceived = this.messageReceived.bind(this)

    this.channel.messageReceived.add(this.messageReceived)
  }

  App.Views.OpenChannel.prototype = Object.create(view.View.prototype)

  var p = App.Views.OpenChannel.prototype

  p.destructor = function() {
    // TODO - call this from somewhere...
    this.channel.messageReceived.remove(this.messageReceived)
  }

  p.render = function() {
    this.el.innerHTML = ""

    this.channel.messages.forEach(function(m) {
      var v = new App.Views.Message({model: m, users: this.users})
      this.el.appendChild(v.render())
    }, this)

    this.el.appendChild(this.composer.el)

    return this.el
  }

  p.messageReceived = function() {
    this.render()
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
      if(this.users[i].id == this.model.user_id) {
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
