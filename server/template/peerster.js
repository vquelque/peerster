$(document).ready(function(){ 
        $.when(
            $.get("/id"),
            $.get("/node"),
            $.get("/message"),
            $.get("/routes")
        )
        .then(function(id, nodes, messages, routes) {
            const name = JSON.parse(id[0])
            $(".peerID").text(name)
        });
})