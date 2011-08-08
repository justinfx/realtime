RT.createChannel("video",{
	onReceive : function(e) {
		var action = e.data.msg.action;
		var video = document.getElementById("video");
		switch(action) {
			case "play":
			case "pause":
				RT.publish("video",{action : "noop"});
				video[action]();
				break;
			case "seek":
				video.currentTime = e.data.msg.time;
				break;
			case "noop":
				console.log("got noop");
				break;	
			default:
				break;		
		}	
	}
});