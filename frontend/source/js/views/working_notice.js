(function() {
  "use strict";

  App.Views.WorkingNotice = function() {
    var self = this;
    this.el = document.getElementById("working_notice");
    this.el.style.display = "none";

    conn.firstRequestStarted.add(function() {
      self.el.style.display = "";
    });
    conn.lastRequestFinished.add(function() {
      self.el.style.display = "none";
    });
  };
})();
