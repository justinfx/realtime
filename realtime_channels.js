
// creat the functionality for the chat
RT.createChannel("chat_test",{

	// handle on receive (required)
	onReceive : function(data) {
		
		// append this message to the chat
		$("#chat").append("<div>"+data.identity+" - " + data.msg + "</div>");
		
		// scroll to bottom
		$("#chat")[0].scrollTop = $("#chat")[0].scrollHeight;
	},
	
	// handle on subscribe (not required);
	onSubscribe : function(data) {
		console.log("user defined on subscribe",data);
	},
	
	// handle a custom event, not required.
	customMethod : function(data) {
		console.log("custom",data);
		$("#chat").append("<div>CUSTOM: " + data.msg + "</div>");
	}
});

RT.createChannel("global",{
	onReceive : function(data) {
		$("body").prepend("<div class='global'>"+data.msg+"</div>");
	}
});	