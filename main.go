package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// conectar con el backend
		backendConn, err := net.Dial("tcp", "localhost:8500")
		if err != nil {
			log.Fatalf("Error connecting to backend: %v", err)
		}
		defer backendConn.Close()

		// manejar las solicitudes HTTP
		r.Write(backendConn)

		// 3. Leer la respuesta del backend y copiarla al cliente
		backendResp, err := http.ReadResponse(
			bufio.NewReader(backendConn),
			r,
		)
		if err != nil {
			http.Error(w, "bad response", http.StatusBadGateway)
			return
		}
		defer backendResp.Body.Close()

		// 4. Copiar cabeceras y status
		for key, values := range backendResp.Header {
			for _, v := range values {
				w.Header().Add(key, v)
			}
		}
		w.WriteHeader(backendResp.StatusCode)

		// 5. Copiar el body
		io.Copy(w, backendResp.Body)
	})

	log.Println("FW-Proxy listening on :8080")
	http.ListenAndServe(":8080", nil)
}
