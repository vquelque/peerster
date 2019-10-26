$(document).ready(function() {
  $.getJSON("/id", function(data) {
    $("#peerID").html(data);
  });
  fetchMessages();
  getPeers();

  function getPeers() {
    $.getJSON("/peers", function(data) {
      var items = [];
      $.each(data, function(key, val) {
        items.push("<li id='" + key + "' class='peerItem'>" + val + "</li>");
      });
      $(".peerList").html(items.join(""));
    });
  }

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

  $("#chat").submit(function(event) {
    event.preventDefault(); //prevent default action
    $.ajax({
      url: "/message",
      type: "POST",
      data: $(this).serialize()
    }).done(function(response) {
      $("#chat")[0].reset();
      fetchMessages();
      getPeers();
    });
  });

  function fetchMessages() {
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

      $(".msgList").html(items.join(""));
    });
  }

  setInterval(fetchMessages, 1000); //fetch messages every seconds
});
