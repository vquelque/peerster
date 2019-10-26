$(document).ready(function() {
  var peer = getUrlVars()["peer"];
  var reqURL = "/privateMsg?peer=" + peer;
  $("#peerID").html(peer);

  $.getJSON(reqURL, function(data) {
    var items = [];
    $.each(data, function(key, val) {
      var str =
        "<strong> PRIVATE MESSAGE </strong> " +
        " <strong> from </strong> " +
        val.Origin +
        "<br>" +
        "<strong> MESSAGE : </strong>" +
        val.Text;
      items.push("<li id='" + key + "' class='msgItem'>" + str + "</li>");
    });

    $("<ul/>", {
      class: "msgList",
      html: items.join("")
    }).appendTo("#chatbox");
  });

  $("#send").click(function() {
    $.post(reqURL, $("#chat").serialize(), function(data) {
      $(location).attr("href", "private.html?peer=" + peer);
    });
  });
});

// Read a page's GET URL variables and return them as an associative array.
function getUrlVars() {
  var vars = [],
    hash;
  var hashes = window.location.href
    .slice(window.location.href.indexOf("?") + 1)
    .split("&");
  for (var i = 0; i < hashes.length; i++) {
    hash = hashes[i].split("=");
    vars.push(hash[0]);
    vars[hash[0]] = hash[1];
  }
  return vars;
}
