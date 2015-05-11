(function($) {
  console.log("loaded")
  function wsurl(s) {
    var l = window.location;
    return ((l.protocol === "https:") ? "wss://" : "ws://") + l.hostname + (((l.port != 80) && (l.port != 443)) ? ":" + l.port : "") + s;
  }

  var url = wsurl("/ws?session=" + $("#session").attr("data-id"))
  var serversocket = new WebSocket(url);
  console.log("add onmessage:", url)
  serversocket.onmessage = function (e) {
    console.log("onmessage trigger")
    var message = jQuery.parseJSON(e.data);
    var target = message.Target;
    if (target.charAt(0) != "#") {
      console.log("Not a channel: " + target);
      //return;
    }

    console.log("Replace target:", target);
    // Temporary solution to load users.
    // TODO: make it work with websockets.
    if (message.Mtype === 'hidden' && message.Text === 'reload') {
      location.reload();
    }
    else {
      $(".channel-boxes ." + target + "-channel")
          .append("<div class=\"message " + message.Mtype + "\"><div class=\"sender\">" + message.Sender.replace("<", "&lt;").replace(">", "&gt;") + "</div><div class=\"text\">" + message.Text.replace("<", "&lt;").replace(">", "&gt;") + "</div></div>");
    }
  };

  var sendsocket = new WebSocket(wsurl("/sendws?session=" + $("#session").attr("data-id")));

  $("#message-input").keyup(function(e){
    console.log("Pressed:", e.which);
    if (e.which === 13) {
      console.log("SEND:", $(".channel-tab.active").attr("data-channel") + "||" + $(this).val());
      sendsocket.send($(".channel-tab.active").attr("data-channel") + "||" + $(this).val());
      $(this).val("");
    }
  });
  
  $(document).ready(function(){
    $('.channel-tab:first').addClass('active');
    $('.channel-box:first').addClass('active');
    $('.user-box:first').addClass('active');    
  });
})(jQuery);