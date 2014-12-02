(function() {
  "use strict"

  App.Views.HomePage = function() {
    view.View.call(this, "div")
    this.el.className = "home"

    this.header = this.createChild(App.Views.LoggedInHeader)
    this.header.render()

    this.channels = this.createChild(App.Views.Channels)
    this.channels.render()

    this.openChannel = this.createChild(App.Views.OpenChannel)
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


  App.Views.Channels = function() {
    view.View.call(this, "ol")
    this.el.className = "channels"
  }

  App.Views.Channels.prototype = Object.create(view.View.prototype)

  var p = App.Views.Channels.prototype

  p.render = function() {
    this.el.innerHTML = ""

    window.chat.channels.forEach(function(c) {
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
    return this.el
  }


  App.Views.OpenChannel = function() {
    view.View.call(this, "div")
    this.el.className = "openChannel"
  }

  App.Views.OpenChannel.prototype = Object.create(view.View.prototype)

  var p = App.Views.OpenChannel.prototype

  p.render = function() {
    this.el.innerHTML = ""

    window.chat.openChannel.messages.forEach(function(m) {
      var v = new App.Views.Message({model: m})
      this.el.appendChild(v.render())
    }, this)

    return this.el
  }

  App.Views.Message = function(options) {
    view.View.call(this, "div")
    this.el.className = "message"
    this.model = options.model
  }

  App.Views.Message.prototype = Object.create(view.View.prototype)

  var p = App.Views.Message.prototype

  p.template = JST["templates/message"]

  p.render = function() {
    this.el.innerHTML = this.template(this.model)
    return this.el
  }
})()
