(function() {
  "use strict"

  App.Views.HomePage = function() {
    view.View.call(this, "div")
    this.el.className = "home"

    this.header = this.createChild(App.Views.LoggedInHeader)
    this.header.render()

    this.channels = this.createChild(App.Views.Channels)
    this.channels.render()
  }

  App.Views.HomePage.prototype = Object.create(view.View.prototype)

  var p = App.Views.HomePage.prototype
  p.template = JST["templates/home_page"]

  p.render = function() {
    this.el.innerHTML = ""
    this.el.appendChild(this.header.el)
    this.el.appendChild(this.channels.el)
    return this.el
  }


  App.Views.Channels = function() {
    view.View.call(this, "ol")
    this.el.className = "channels"
  }

  App.Views.Channels.prototype = Object.create(view.View.prototype)

  var p = App.Views.Channels.prototype

  p.render = function() {
    this.el.innerHTML = "<li>general</li><li>random</li>"
    return this.el
  }
})()
