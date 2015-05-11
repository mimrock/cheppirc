(function($) {
$("#connect").click(function() {
  $("#status-box").html("Connecting...")
  $.ajax({
    type: "POST",
    url: "/connect",
    data: $("#connect-form").serialize(),
    success: function(data)
      {
        var response = jQuery.parseJSON(data);
        $("#status-box").html("Connected: " + response.success + " Message: " + response.message + " raw: " + data)
        //alert("debug")
        window.location.replace("/chat?session=" + response.message)
      },
    //dataType: dataType
  });
});  
})(jQuery);
