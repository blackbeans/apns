package server

import (
	"net/http"
	"testing"
	"time"
)

func TestMomoHttpServer(t *testing.T) {
	http.HandleFunc("/test", func(out http.ResponseWriter, req *http.Request) {
		out.Write([]byte("dbc"))
	})
	httpserver := NewMomoHttpServer(":7070", nil)
	go func() {
		httpserver.ListenAndServe()
	}()

	time.Sleep(10 * time.Second)

	httpserver.Shutdonw()

}
