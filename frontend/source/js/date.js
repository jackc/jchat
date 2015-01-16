(function() {
  "use strict"

  var monthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
  var dayNames = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"]

  var ordinalSuffix = function(n) {
    n = n % 100
    if (11 <= n && n < 13) {
      return "th"
    }

    n = n % 10;
    if (n == 1) return "st"
    if (n == 2) return "nd"
    if (n == 3) return "rd"
    return "th"
  };

  var hour12 = function(h) {
    h = (h % 12);
    return h == 0 ? 12 : h
  }

  var min = function(m) {
    return m < 10 ? "0" + m : m
  }

  var xm = function(h) {
    return h < 12 ? "am" : "pm"
  }

  Date.prototype.toDaybreakString = function() {
    var t = this,
        y = t.getFullYear(),
        m = monthNames[t.getMonth()],
        d = t.getDate(),
        o = ordinalSuffix(d),
        dn = dayNames[t.getDay()];

    return dn + ", " + m + " " + d + o + ", " + y
  }


  Date.prototype.toPostTimeString = function() {
    var t = this,
        h = t.getHours(),
        mm = t.getMinutes();

    return hour12(h) + ":" + min(mm) + " " + xm(h)
  }
})()
