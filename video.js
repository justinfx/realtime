RT.createChannel("video",{
	onReceive : function(e) {
		var action = e.data.msg.action;
		var video = document.getElementById("video");
		switch(action) {
			case "play":
			case "pause":
				video[action]();
				break;
			case "seek":
				video.currentTime = e.data.msg.time;
				break;
			default:
				break;		
		}	
		
	}
});