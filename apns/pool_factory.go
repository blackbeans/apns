package apns

type ConnPool struct {
}

func (self *ConnPool) get() *ApnsConnection {
	return nil
}

func (self *ConnPool) release(conn *ApnsConnection) {
	//do nothing
}

func (self *ConnPool) close() {

}
