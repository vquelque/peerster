$(document).ready(function() {
  $(".peerID").load("/id");

  $.getJSON("/peers", function(data) {
    var items = [];
    $.each(data, function(key, val) {
      items.push("<li id='" + key + "'>" + val + "</li>");
    });

    $("<ul/>", {
      class: "peerList",
      html: items.join("")
    }).appendTo(".peers");
  });

  $.getJSON("/messages", function(data) {
    var items = [];
    $.each(data, function(key, val) {
      var str =
        "Rumor ID " +
        val.ID +
        " from " +
        val.Origin +
        "<br>" +
        "MESSAGE : " +
        val.Text;
      items.push("<li id='" + key + "' class='msgItem'>" + str + "</li>");
    });

    $("<ul/>", {
      class: "msgList",
      html: items.join("")
    }).appendTo("#chatbox");
  });
});
