$(document).ready(function() {
  $.getJSON("/id", function(data) {
    $("#peerID").html(data);
  });
  update();

  function getPeers() {
    $.getJSON("/peers", function(data) {
      var items = [];
      $.each(data, function(key, val) {
        items.push("<li id='" + key + "' class='peerItem'>" + val + "</li>");
      });
      $(".peerList").html(items.join(""));
    });
  }

  function getContacts() {
    $.getJSON("/contacts", function(data) {
      var contacts = [];
      var pNames = [];
      $.each(data, function(key, val) {
        req = "/private.html?peer=" + key;
        contacts.push(
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
        pNames.push("<option value='" + key + "'>" + key + "</option>");
      });
      $(".contactList").html(contacts.join(""));
      $(".fileRequestPeer").html(pNames.join(""));
    });
  }

  $("#chat").submit(function(event) {
    event.preventDefault(); //prevent default action
    $.ajax({
      url: "/message",
      type: "POST",
      data: $(this).serialize()
    }).done(function(response) {
      $("#chat")[0].reset();
      update();
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

  function downloadSearchedFile(metahash, filename) {
    $.post("/downloadFile", {
      metahash: metahash,
      filename: filename,
      peer: ""
    });
  }

  function searchResults() {
    $.getJSON("/searchResults", function(data) {
      var items = [];
      $.each(data, function(key, val) {
        items.push(
          "<li class='searchResultItem' id='" + key + "'>" + val + "</li>"
        );
      });
      $(".downloadableList").html(items.join(""));
      $(".searchResultItem").each(function(index) {
        $(this).on("click", function() {
          var name = $(this).text();
          var metahash = $(this).attr("id");
          downloadSearchedFile(metahash, name);
        });
      });
    });
  }

  function update() {
    fetchMessages();
    getContacts();
    getPeers();
    searchResults();
  }

  setInterval(update, 10000); //fetch messages every 10 second
});
