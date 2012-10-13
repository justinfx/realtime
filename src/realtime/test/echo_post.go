package main

import (
	"fmt"
	"net/http"
)

func main() {

	http.HandleFunc("/api/realtime/monitor", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			buf := make([]byte, 4096)
			_, err := r.Body.Read(buf)
			if err != nil {
				fmt.Println("Error reading POST request", err)
				return
			}
			fmt.Printf("Content-Type: %s\n", r.Header.Get("Content-Type"))
			fmt.Printf("%s\n", buf)
		}
	})

	http.ListenAndServe(":8080", nil)

}
