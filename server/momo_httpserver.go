package server

import (
	// "errors"
	"errors"
	"net"
	"net/http"
	"time"
)

type MomoHttpServer struct {
	http.Server
	stop chan int //Channel used only to indicate listener should shutdown
}

func NewMomoHttpServer(addr string, handler http.Handler) *MomoHttpServer {
	server := &MomoHttpServer{}
	server.Addr = addr
	server.Handler = handler
	server.stop = make(chan int)
	return server
}

// ListenAndServe listens on the TCP network address srv.Addr and then
// calls Serve to handle requests on incoming connections.  If
// srv.Addr is blank, ":http" is used.
func (self *MomoHttpServer) ListenAndServe() error {
	addr := self.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for i := 0; i < 200; i++ {
		go self.Serve(stoppableListener{ln.(*net.TCPListener), self.stop})
	}
	return nil
}

func (self *MomoHttpServer) Shutdonw() {
	close(self.stop)
}

//连接Listener
type stoppableListener struct {
	*net.TCPListener          //Wrapped listener
	stop             chan int //Channel used only to indicate listener should shutdown

}

func (sl stoppableListener) Accept() (c net.Conn, err error) {

	for {

		tc, err := sl.TCPListener.AcceptTCP()

		//Check for the channel being closed
		select {
		case <-sl.stop:
			return nil, errors.New("STOP LISTEN!")
		default:
			//If the channel is still open, continue as normal
		}
		if nil == err {
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(3 * time.Minute)
		} else {
			return nil, err
		}

		return tc, err
	}

}
