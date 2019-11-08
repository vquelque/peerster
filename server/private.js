$(document).ready(function() {
  var peer = getUrlVars()["peer"];
  var reqURL = "/privateMsg?peer=" + peer;
  $("#peerID").html(peer);
  fetchMessages();

  function fetchMessages() {
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

      $(".msgList").html(items.join(""));
    });
  }

  $("#chat").submit(function(event) {
    event.preventDefault(); //prevent default action
    $.ajax({
      url: reqURL,
      type: "POST",
      data: $(this).serialize()
    }).done(function(response) {
      $("#chat")[0].reset();
      fetchMessages();
    });
  });

  setInterval(fetchMessages, 10000); //fetch messages every 10 seconds
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
