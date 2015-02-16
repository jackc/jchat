(function() {
  "use strict"

  App.Views.ChannelsPage = function(options) {
    view.View.call(this, "div")
    this.el.className = "channels"

    this.chat = options.chat

    this.header = this.createChild(App.Views.LoggedInHeader)
    this.header.render()

    this.channelForm = this.createChild(App.Views.ChannelForm, {chat: this.chat})
    this.channelForm.render()
  }

  App.Views.ChannelsPage.prototype = Object.create(view.View.prototype)

  var p = App.Views.ChannelsPage.prototype
  p.template = JST["templates/channels_page"]

  p.render = function() {
    this.el.innerHTML = ""
    this.el.appendChild(this.header.el)
    this.el.appendChild(this.channelForm.el)
    return this.el
  }


  App.Views.ChannelForm = function(options) {
    view.View.call(this, "form")
    this.el.className = "channel"

    this.chat = options.chat

    this.save = this.save.bind(this)
    this.onSaveSuccess = this.onSaveSuccess.bind(this)
    this.onSaveFailure = this.onSaveFailure.bind(this)
  }

  App.Views.ChannelForm.prototype = Object.create(view.View.prototype)

  var p = App.Views.ChannelForm.prototype

  p.template = JST["templates/channel_form"]

  p.render = function() {
    this.el.innerHTML = this.template()
    this.listen()
    return this.el
  }

  p.listen = function() {
    this.el.addEventListener("submit", this.save )
  }

  p.save = function(e) {
    e.preventDefault()
    var form = e.currentTarget
    var attrs = {
      name: form.elements.name.value
    }
    this.chat.createChannel(attrs, {
      succeeded: this.onSaveSuccess,
      failed: this.onSaveFailure
    })
  }

  p.onSaveSuccess = function(data) {
    window.router.navigate('home')
  }

  p.onSaveFailure = function(error) {
    alert(error.message + " - " + error.data)
  }
})()
