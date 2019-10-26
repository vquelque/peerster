$(document).ready(function() {
  $.getJSON("/id", function(data) {
    $("#peerID").html(data);
  });

  $.getJSON("/peers", function(data) {
    var items = [];
    $.each(data, function(key, val) {
      items.push("<li id='" + key + "' class='peerItem'>" + val + "</li>");
    });

    $("<ul/>", {
      class: "peerList",
      html: items.join("")
    }).appendTo(".peers");
  });

  $.getJSON("/contacts", function(data) {
    $.each(data, function(key, val) {
      req = "/private.html?peer=" + key;
      $(".contactList").append(
        "<li class='contactItem'>" +
          "<a target='popup' rel='noopener noreferrer' href=" +
          req +
          ">" +
          key +
          "</a>" +
          " (via " +
          val +
          ")" +
          "</li>"
      );
    });
  });

  $.getJSON("/message", function(data) {
    var items = [];
    $.each(data, function(key, val) {
      var mID = parseInt(val.ID);
      var str =
        "<strong> Rumor ID </strong> " +
        mID +
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
});
