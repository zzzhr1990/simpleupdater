package simpleupdater

//simpleupdater listeners and connections allow graceful
//restarts by tracking when all connections from a listener
//have been closed

import (
	"net"
	"os"
	"sync"
	"time"
)

func newsimpleupdaterListener(l net.Listener) *simpleupdaterListener {
	return &simpleupdaterListener{
		Listener:     l,
		closeByForce: make(chan bool),
	}
}

// gracefully closing net.Listener
type simpleupdaterListener struct {
	net.Listener
	closeError   error
	closeByForce chan bool
	wg           sync.WaitGroup
}

func (l *simpleupdaterListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.(*net.TCPListener).AcceptTCP()
	if err != nil {
		return nil, err
	}
	conn.SetKeepAlive(true)                  // see http.tcpKeepAliveListener
	conn.SetKeepAlivePeriod(3 * time.Minute) // see http.tcpKeepAliveListener
	uconn := simpleupdaterConn{
		Conn:   conn,
		wg:     &l.wg,
		closed: make(chan bool),
	}
	go func() {
		//connection watcher
		select {
		case <-l.closeByForce:
			uconn.Close()
		case <-uconn.closed:
			//closed manually
		}
	}()
	l.wg.Add(1)
	return uconn, nil
}

// non-blocking trigger close
func (l *simpleupdaterListener) release(timeout time.Duration) {
	//stop accepting connections - release fd
	l.closeError = l.Listener.Close()
	//start timer, close by force if deadline not met
	waited := make(chan bool)
	go func() {
		l.wg.Wait()
		waited <- true
	}()
	go func() {
		select {
		case <-time.After(timeout):
			close(l.closeByForce)
		case <-waited:
			//no need to force close
		}
	}()
}

// blocking wait for close
func (l *simpleupdaterListener) Close() error {
	l.wg.Wait()
	return l.closeError
}

func (l *simpleupdaterListener) File() *os.File {
	// returns a dup(2) - FD_CLOEXEC flag *not* set
	tl := l.Listener.(*net.TCPListener)
	fl, _ := tl.File()
	return fl
}

// notifying on close net.Conn
type simpleupdaterConn struct {
	net.Conn
	wg     *sync.WaitGroup
	closed chan bool
}

func (o simpleupdaterConn) Close() error {
	err := o.Conn.Close()
	if err == nil {
		o.wg.Done()
		o.closed <- true
	}
	return err
}