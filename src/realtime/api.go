// api.go
package main

import (
	"http"
	"fmt"
)

// The handler function for accepting and publishing messages
// via a POST request. Request Body must be a valid JSON message
// structure.
func HandlePostAPIPublish(writer http.ResponseWriter, req *http.Request) {

	if !LICENSE.CheckHttpRequest(req) {
		writer.WriteHeader(http.StatusUnauthorized)
		writer.Write([]byte("Error: Domain name origin is not licensed for this server\n"))
		return
	} else if req.ContentLength == -1 {
		writer.WriteHeader(http.StatusLengthRequired)
		return
	}

	length := int(req.ContentLength)

	buf := make([]byte, length)
	if nr, _ := req.Body.Read(buf); nr != length {
		Debugln("api/HandlePostAPIReq: Error reading Body. read %d bytes but expected %d",
			nr, length)
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte("Error: Internal error reading request Body\n"))
		return
	}

	msg, err := NewJsonMessage(buf)
	if err != nil {
		Debugf("api/HandlePostAPIReq: Bad message format in POST request: (message) %v, (error) %v",
			string(buf), err)
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
		return
	}

	Debugln("api/HandlePostAPIReq: Message received:", msg.String())

	err = SERVER.publish(nil, msg)
	if err != nil {
		Debugf("api/HandlePostAPIReq: Bad message format in POST request: (message) %v, (error) %v",
			msg.String(), err)
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
		return
	}

	writer.WriteHeader(http.StatusOK)
}
