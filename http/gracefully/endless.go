// fork from github.com/fvbock/endless@v0.0.0-20170109170031-447134032cb6
// modified:
// 1. support windows signals
// 2. log content
// 3. close by http.Serve.ShutDown() rather than listener.Close()
// 4. make sure Serve() exit after Shutdown() triggered by signals
// 5. implement *net.TcpConn all public methods

package gracefully

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/routine"
)

const (
	PreSignal = iota
	PostSignal

	StateInit
	StateRunning
	StateShuttingDown
	StateTerminate
)

var (
	DefaultReadTimeOut    time.Duration
	DefaultWriteTimeOut   time.Duration
	DefaultMaxHeaderBytes int
	DefaultHammerTime     time.Duration

	runningServerReg     sync.RWMutex
	runningServers       map[string]*endlessServer
	runningServersOrder  []string
	socketPtrOffsetMap   map[string]uint
	runningServersForked bool

	isChild     bool
	socketOrder string
)

func init() {
	runningServerReg = sync.RWMutex{}
	runningServers = make(map[string]*endlessServer)
	runningServersOrder = []string{}
	socketPtrOffsetMap = make(map[string]uint)

	DefaultMaxHeaderBytes = 0 // use http.DefaultMaxHeaderBytes - which currently is 1 << 20 (1MB)

	// after a restart the parent will finish ongoing requests before
	// shutting down. set to a negative value to disable
	DefaultHammerTime = 60 * time.Second
}

type endlessServer struct {
	*http.Server
	SignalHooks map[int]map[os.Signal][]func()
	BeforeBegin func(addr string)
	AppName     string

	endlessListener  net.Listener
	tlsInnerListener *endlessListener
	close            chan struct{}
	wg               *sync.WaitGroup
	sigChan          chan os.Signal
	isChild          bool
	state            uint8
	lock             *sync.RWMutex
}

// NewServer returns an initialized endlessServer Object. Calling Serve on it will
// actually "start" the server.
func NewServer(appName string, handler http.Handler, addr string, nextProtos []string) (srv *endlessServer) {
	runningServerReg.Lock()
	defer runningServerReg.Unlock()

	socketOrder = os.Getenv("ENDLESS_SOCKET_ORDER")
	isChild = os.Getenv("ENDLESS_CONTINUE") != ""

	if len(socketOrder) > 0 {
		for i, addr := range strings.Split(socketOrder, ",") {
			socketPtrOffsetMap[addr] = uint(i)
		}
	} else {
		socketPtrOffsetMap[addr] = uint(len(runningServersOrder))
	}

	srv = &endlessServer{
		AppName: appName,
		Server: &http.Server{
			Addr:           addr,
			ReadTimeout:    DefaultReadTimeOut,
			WriteTimeout:   DefaultWriteTimeOut,
			MaxHeaderBytes: DefaultMaxHeaderBytes,
			Handler:        handler,
			TLSConfig:      &tls.Config{NextProtos: nextProtos},
		},
		close:       make(chan struct{}),
		wg:          new(sync.WaitGroup),
		sigChan:     make(chan os.Signal),
		isChild:     isChild,
		SignalHooks: newSignalHookFunc(),
		state:       StateInit,
		lock:        new(sync.RWMutex),
	}

	runningServersOrder = append(runningServersOrder, addr)
	runningServers[addr] = srv

	return
}

// ListenAndServe listens on the TCP network address addr and then calls Serve
// with handler to handle requests on incoming connections. Handler is typically
// nil, in which case the DefaultServeMux is used.
func ListenAndServe(appName string, handler http.Handler, addr string, nextProtos []string) error {
	server := NewServer(appName, handler, addr, nextProtos)
	return server.ListenAndServe()
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it expects
// HTTPS connections. Additionally, files containing a certificate and matching
// private key for the server must be provided. If the certificate is signed by a
// certificate authority, the certFile should be the concatenation of the server's
// certificate followed by the CA's certificate.
func ListenAndServeTLS(appName string, handler http.Handler, addr, certFile, keyFile string,
	nextProtos []string) error {
	server := NewServer(appName, handler, addr, nextProtos)
	return server.ListenAndServeTLS(certFile, keyFile)
}

// Serve accepts incoming HTTP connections on the listener l, creating a new
// service goroutine for each. The service goroutines read requests and then call
// handler to reply to them. Handler is typically nil, in which case the
// DefaultServeMux is used.
//
// In addition to the stl Serve behaviour each connection is added to a
// sync.WaitGroup so that all outstanding connections can be served before shutting
// down the server.
func (e *endlessServer) Serve() (err error) {
	defer log.Println(syscall.Getpid(), "[Common] endless exited.")

	e.setState(StateRunning)
	log.Println(syscall.Getpid(), "[Common] endless listening", e.endlessListener.Addr())

	// ignore server closed error because it happened when we call Server.Shutdown or Server.Close
	if err = e.Server.Serve(e.endlessListener); err != nil {
		// http: Server closed
		// use of closed network connection
		if errors.Is(err, http.ErrServerClosed) || isClosedConnError(err) {
			err = nil
		}
	}
	log.Println(syscall.Getpid(), "[Common] endless waiting for connections to finish...")
	e.wg.Wait()
	e.setState(StateTerminate)

	<-e.close

	return
}

// ListenAndServe listens on the TCP network address srv.Addr and then calls Serve
// to handle requests on incoming connections. If srv.Addr is blank, ":http" is
// used.
func (e *endlessServer) ListenAndServe() (err error) {
	addr := e.Addr
	if addr == "" {
		addr = ":http"
	}

	if err = setupHTTP2_Serve(e.Server); err != nil {
		return
	}

	routine.Go(e.handleSignals, routine.AppName(e.AppName))

	l, err := e.getListener(addr)
	if err != nil {
		log.Println(syscall.Getpid(), "[Common] endless", err)
		return
	}

	e.endlessListener = newEndlessListener(l, e)
	if e.isChild {
		_ = syscallKill(syscall.Getppid())
	}

	if e.BeforeBegin != nil {
		e.BeforeBegin(e.Addr)
	}

	return e.Serve()
}

// ListenAndServeTLS listens on the TCP network address srv.Addr and then calls
// Serve to handle requests on incoming TLS connections.
//
// Filenames containing a certificate and matching private key for the server must
// be provided. If the certificate is signed by a certificate authority, the
// certFile should be the concatenation of the server's certificate followed by the
// CA's certificate.
//
// If srv.Addr is blank, ":https" is used.
func (e *endlessServer) ListenAndServeTLS(certFile, keyFile string) (err error) {
	addr := e.Addr
	if addr == "" {
		addr = ":https"
	}

	// Setup HTTP/2 before srv.Serve, to initialize srv.TLSConfig
	// before we clone it and create the TLS Listener.
	if err = setupHTTP2_ServeTLS(e.Server); err != nil {
		return
	}

	config := new(tls.Config)
	if e.Server.TLSConfig != nil {
		*config = *e.Server.TLSConfig.Clone()
	}
	if !utils.NewSet(config.NextProtos...).Contains("http/1.1") {
		config.NextProtos = append(config.NextProtos, "http/1.1")
	}

	configHasCert := len(config.Certificates) > 0 || config.GetCertificate != nil
	if !configHasCert || certFile != "" || keyFile != "" {
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
	}

	routine.Go(e.handleSignals, routine.AppName(e.AppName))

	l, err := e.getListener(addr)
	if err != nil {
		log.Println(syscall.Getpid(), "[Common] endless error occur when get listener:", err)
		return
	}

	e.tlsInnerListener = newEndlessListener(l, e)
	e.endlessListener = tls.NewListener(e.tlsInnerListener, config)
	if e.isChild {
		_ = syscallKill(syscall.Getppid())
	}

	return e.Serve()
}

// Shutdown closes the listener so that none new connections are accepted. it also
// starts a goroutine that will hammer (stop all running requests) the server
// after DefaultHammerTime.
func (e *endlessServer) Shutdown() {
	// make sure server Shutdown & log printed before Serve() return
	defer func() {
		e.lock.Lock()
		defer e.lock.Unlock()
		if _, ok := utils.IsChannelClosed(e.close); ok {
			return
		}
		if e.close != nil {
			close(e.close)
		}
	}()

	if e.getState() != StateRunning {
		return
	}

	e.setState(StateShuttingDown)
	if DefaultHammerTime >= 0 {
		routine.Loop(e.hammerTime, routine.Args(DefaultHammerTime), routine.AppName(e.AppName))
	}
	// disable keep-alive on existing connections
	e.Server.SetKeepAlivesEnabled(false)

	// TODO: new context with timeout because system may forcefully kill the program
	if err := e.Server.Shutdown(context.TODO()); err != nil {
		log.Println(syscall.Getpid(), "[Common] endless close listener error:", err)
	} else {
		log.Println(syscall.Getpid(), "[Common] endless", e.endlessListener.Addr(), "listener closed.")
	}
}

// RegisterSignalHook registers a function to be run PreSignal or PostSignal for
// a given signal. PRE or POST in this case means before or after the signal
// related code endless itself runs
func (e *endlessServer) RegisterSignalHook(prePost int, sig os.Signal, f func()) (err error) {
	if prePost != PreSignal && prePost != PostSignal {
		err = fmt.Errorf("cannot use %v for prePost arg. Must be endless.PRE_SIGNAL or endless.POST_SIGNAL", sig)
		return
	}
	for _, s := range hookableSignals {
		if s == sig {
			e.SignalHooks[prePost][sig] = append(e.SignalHooks[prePost][sig], f)
			return
		}
	}
	err = fmt.Errorf("signal %v is not supported", sig)
	return
}

// getListener either opens a new socket to listen on, or takes the acceptor socket
// it got passed when restarted.
func (e *endlessServer) getListener(addr string) (l net.Listener, err error) {
	if e.isChild {
		ptrOffset := uint(0)
		runningServerReg.RLock()
		defer runningServerReg.RUnlock()
		if len(socketPtrOffsetMap) > 0 {
			ptrOffset = socketPtrOffsetMap[addr]
			log.Println(syscall.Getpid(), "[Common] endless addr:", addr, "ptr offset:", socketPtrOffsetMap[addr])
		}

		f := os.NewFile(uintptr(3+ptrOffset), "")
		l, err = net.FileListener(f)
		if err != nil {
			err = fmt.Errorf("net.FileListener error: %v", err)
			return
		}
	} else {
		l, err = net.Listen("tcp", addr)
		if err != nil {
			err = fmt.Errorf("net.Listen error: %v", err)
			return
		}
	}
	return
}

func (e *endlessServer) signalHooks(ppFlag int, sig os.Signal) {
	if _, notSet := e.SignalHooks[ppFlag][sig]; !notSet {
		return
	}
	for _, f := range e.SignalHooks[ppFlag][sig] {
		f()
	}
}

// hammerTime forces the server to shut down in a given timeout - whether it
// finished outstanding requests or not. if Read/WriteTimeout are not set or the
// max header size is very big a connection could hang...
//
// srv.Serve() will not return until all connections are served. this will
// unblock the srv.wg.Wait() in Serve() thus causing ListenAndServe(TLS) to
// return.
func (e *endlessServer) hammerTime(d time.Duration) {
	defer func() {
		// we are calling e.wg.Done() until it panics which means we called
		// Done() when the counter was already at 0, and we're done.
		// (and thus Serve() will return and the parent will exit)
		if r := recover(); r != nil {
			log.Println(syscall.Getpid(), "[Common] endless wait group at 0", r)
		}
	}()
	if e.getState() != StateShuttingDown {
		return
	}
	time.Sleep(d)
	log.Println(syscall.Getpid(), "[Common] endless harmerTime() forcefully shutting down parent")
	for {
		if e.getState() == StateTerminate {
			break
		}
		e.wg.Done()
		runtime.Gosched()
	}
}

func (e *endlessServer) fork() (err error) {
	runningServerReg.Lock()
	defer runningServerReg.Unlock()

	// only one server instance should fork!
	if runningServersForked {
		return errors.New("another process already forked, ignoring this one")
	}

	runningServersForked = true

	var files = make([]*os.File, len(runningServers))
	var orderArgs = make([]string, len(runningServers))
	// get the accessor socket fds for _all_ server instances
	for _, srvPtr := range runningServers {
		// introspect.PrintTypeDump(srvPtr.endlessListener)
		switch srvPtr.endlessListener.(type) {
		case *endlessListener:
			// normal listener
			files[socketPtrOffsetMap[srvPtr.Server.Addr]] = srvPtr.endlessListener.(*endlessListener).File()
		default:
			// tls listener
			files[socketPtrOffsetMap[srvPtr.Server.Addr]] = srvPtr.tlsInnerListener.File()
		}
		orderArgs[socketPtrOffsetMap[srvPtr.Server.Addr]] = srvPtr.Server.Addr
	}

	env := append(
		os.Environ(),
		"ENDLESS_CONTINUE=1",
	)
	if len(runningServers) > 1 {
		env = append(env, fmt.Sprintf(`ENDLESS_SOCKET_ORDER=%s`, strings.Join(orderArgs, ",")))
	}

	path := os.Args[0]
	var args []string
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}

	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = files
	cmd.Env = env

	if err = cmd.Start(); err != nil {
		log.Fatalf("%v [Common] endless restart: failed to launch, error: %v", syscall.Getpid(), err)
	}

	return
}

func (e *endlessServer) getState() uint8 {
	e.lock.RLock()
	defer e.lock.RUnlock()

	return e.state
}

func (e *endlessServer) setState(st uint8) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.state = st
}

type endlessListener struct {
	net.Listener
	stopped bool
	server  *endlessServer
}

func newEndlessListener(l net.Listener, srv *endlessServer) (el *endlessListener) {
	return &endlessListener{
		Listener: l,
		server:   srv,
	}
}

func (e *endlessListener) Accept() (c net.Conn, err error) {
	tc, err := e.Listener.(*net.TCPListener).AcceptTCP()
	if err != nil {
		return
	}

	// see net/http.tcpKeepAliveListener
	_ = tc.SetKeepAlive(true)
	// see net/http.tcpKeepAliveListener
	_ = tc.SetKeepAlivePeriod(3 * time.Minute)

	c = &endlessConn{
		Conn:   tc,
		server: e.server,
	}

	e.server.wg.Add(1)
	return
}

func (e *endlessListener) File() *os.File {
	// returns a dup(2) - FD_CLOEXEC flag *not* set
	tl := e.Listener.(*net.TCPListener)
	fl, _ := tl.File()
	return fl
}

type endlessConn struct {
	net.Conn
	doneOnce sync.Once
	server   *endlessServer
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
func (e *endlessConn) Read(b []byte) (n int, err error) { return e.Conn.Read(b) }

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
func (e *endlessConn) Write(b []byte) (n int, err error) { return e.Conn.Write(b) }

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (e *endlessConn) Close() (err error) {
	defer e.doneOnce.Do(func() {
		e.server.wg.Done()
	})
	return e.Conn.Close()
}

// LocalAddr returns the local network address, if known.
func (e *endlessConn) LocalAddr() net.Addr { return e.Conn.LocalAddr() }

// RemoteAddr returns the remote network address, if known.
func (e *endlessConn) RemoteAddr() net.Addr { return e.Conn.RemoteAddr() }

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail instead of blocking. The deadline applies to all future
// and pending I/O, not just the immediately following call to
// Read or Write. After a deadline has been exceeded, the
// connection can be refreshed by setting a deadline in the future.
//
// If the deadline is exceeded a call to Read or Write or to other
// I/O methods will return an error that wraps os.ErrDeadlineExceeded.
// This can be tested using errors.Is(err, os.ErrDeadlineExceeded).
// The error's Timeout method will return true, but note that there
// are other possible errors for which the Timeout method will
// return true even if the deadline has not been exceeded.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (e *endlessConn) SetDeadline(t time.Time) error { return e.Conn.SetDeadline(t) }

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (e *endlessConn) SetReadDeadline(t time.Time) error { return e.Conn.SetReadDeadline(t) }

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (e *endlessConn) SetWriteDeadline(t time.Time) error { return e.Conn.SetWriteDeadline(t) }

// SyscallConn returns a raw network connection.
// This implements the syscall.Conn interface.
func (e *endlessConn) SyscallConn() (syscall.RawConn, error) {
	return e.Conn.(*net.TCPConn).SyscallConn()
}

// ReadFrom implements the io.ReaderFrom ReadFrom method.
func (e *endlessConn) ReadFrom(r io.Reader) (int64, error) {
	return e.Conn.(*net.TCPConn).ReadFrom(r)
}

// SetLinger sets the behavior of Close on a connection which still
// has data waiting to be sent or to be acknowledged.
//
// If sec < 0 (the default), the operating system finishes sending the
// data in the background.
//
// If sec == 0, the operating system discards any unsent or
// unacknowledged data.
//
// If sec > 0, the data is sent in the background as with sec < 0. On
// some operating systems after sec seconds have elapsed any remaining
// unsent data may be discarded.
func (e *endlessConn) SetLinger(sec int) error {
	return e.Conn.(*net.TCPConn).SetLinger(sec)
}

// SetKeepAlive sets whether the operating system should send
// keep-alive messages on the connection.
func (e *endlessConn) SetKeepAlive(keepalive bool) error {
	return e.Conn.(*net.TCPConn).SetKeepAlive(keepalive)
}

// SetKeepAlivePeriod sets period between keep-alives.
func (e *endlessConn) SetKeepAlivePeriod(d time.Duration) error {
	return e.Conn.(*net.TCPConn).SetKeepAlivePeriod(d)
}

// SetNoDelay controls whether the operating system should delay
// packet transmission in hopes of sending fewer packets (Nagle's
// algorithm).  The default is true (no delay), meaning that data is
// sent as soon as possible after a Write.
func (e *endlessConn) SetNoDelay(noDelay bool) error {
	return e.Conn.(*net.TCPConn).SetNoDelay(noDelay)
}
