self.onmessage = function(e) {

	var ws = new WebSocket("ws://localhost:8001/realtime/websocket");
	ws.send("~m~72~m~~j~{\"type\":\"message\",\"channel\":\"video\",\"data\":{\"msg\":{\"action\":\"play\"}}}");

};