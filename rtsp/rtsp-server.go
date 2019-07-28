package rtsp

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

// Server rtsp server
type Server struct {
	SessionLogger
	TCPListener    *net.TCPListener
	TCPPort        int
	Stoped         bool
	pushers        map[string]*Pusher // Path <-> Pusher
	pushersLock    sync.RWMutex
	addPusherCh    chan *Pusher
	removePusherCh chan *Pusher
}

// Instance server instance
var instance = &Server{
	SessionLogger:  SessionLogger{log.New(os.Stdout, "[RTSPServer]", log.LstdFlags|log.Lshortfile)},
	Stoped:         true,
	TCPPort:        554,
	pushers:        make(map[string]*Pusher),
	addPusherCh:    make(chan *Pusher),
	removePusherCh: make(chan *Pusher),
}

// GetServer get instance
func GetServer() *Server {
	return instance
}

// Start server
func (server *Server) Start() (err error) {
	logger := server.logger
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", server.TCPPort))
	if err != nil {
		return
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return
	}

	go func() {
		for {
			select {
			case pusher, addChnOk := <-server.addPusherCh:
				if addChnOk {
					logger.Printf("Pusher Added, path: %s\n", pusher.Path())
				}
			case pusher, removeChnOk := <-server.removePusherCh:
				if removeChnOk {
					logger.Printf("Pusher Removed, path: %s\n", pusher.Path())
				}
			}
		}
	}()

	server.Stoped = false
	server.TCPListener = listener
	logger.Println("started on", server.TCPPort)
	networkBuffer := 1048576 //Key("network_buffer").MustInt(1048576)
	for !server.Stoped {
		conn, err := server.TCPListener.Accept()
		if err != nil {
			logger.Println(err)
			continue
		}
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			if err := tcpConn.SetReadBuffer(networkBuffer); err != nil {
				logger.Printf("rtsp server conn set read buffer error, %v", err)
			}
			if err := tcpConn.SetWriteBuffer(networkBuffer); err != nil {
				logger.Printf("rtsp server conn set write buffer error, %v", err)
			}
		}

		session := NewSession(server, conn)
		go session.Start()
	}
	return
}

// Stop server
func (server *Server) Stop() {
	logger := server.logger
	logger.Println("rtsp server stop on", server.TCPPort)
	server.Stoped = true
	if server.TCPListener != nil {
		server.TCPListener.Close()
		server.TCPListener = nil
	}
	server.pushersLock.Lock()
	server.pushers = make(map[string]*Pusher)
	server.pushersLock.Unlock()

	close(server.addPusherCh)
	close(server.removePusherCh)
}

//AddPusher adds pusher
func (server *Server) AddPusher(pusher *Pusher) bool {
	logger := server.logger
	logger.Printf("AddPusher %s\n", pusher.Path())
	added := false
	server.pushersLock.Lock()
	oldPusher, ok := server.pushers[pusher.Path()]
	if !ok {
		server.pushers[pusher.Path()] = pusher
		logger.Printf("%v start, now pusher size[%d]", pusher, len(server.pushers))
		added = true
		server.pushersLock.Unlock()
	} else {
		logger.Println("Removing pusher")
		server.pushersLock.Unlock()
		removed := server.RemovePusher(oldPusher)
		if removed {
			logger.Println("Removed pusher")
			return server.AddPusher(pusher)
		}
		added = false
	}

	if added {
		go pusher.Start()
		server.addPusherCh <- pusher
	}
	return added
}

//TryAttachToPusher attach to existing pusher
func (server *Server) TryAttachToPusher(session *Session) (int, *Pusher) {
	server.pushersLock.Lock()
	attached := 0
	var pusher *Pusher
	if _pusher, ok := server.pushers[session.Path]; ok {
		if _pusher.RebindSession(session) {
			session.logger.Printf("Attached to a pusher")
			attached = 1
			pusher = _pusher
		} else {
			attached = -1
		}
	}
	server.pushersLock.Unlock()
	return attached, pusher
}

//RemovePusher removes pusher
func (server *Server) RemovePusher(pusher *Pusher) bool {
	logger := server.logger
	logger.Printf("RemovePusher %s\n", pusher.Path())
	removed := false
	server.pushersLock.Lock()
	if _pusher, ok := server.pushers[pusher.Path()]; ok && pusher.ID() == _pusher.ID() {
		delete(server.pushers, pusher.Path())
		logger.Printf("%v end, now pusher size[%d]\n", pusher, len(server.pushers))
		removed = true
	}
	server.pushersLock.Unlock()
	if removed {
		server.removePusherCh <- pusher
	}
	return removed
}

// GetPusher gets pusher
func (server *Server) GetPusher(path string) (pusher *Pusher) {
	server.pushersLock.RLock()
	pusher = server.pushers[path]
	server.pushersLock.RUnlock()
	return
}

//GetPushers gets all pushers
func (server *Server) GetPushers() (pushers map[string]*Pusher) {
	pushers = make(map[string]*Pusher)
	server.pushersLock.RLock()
	for k, v := range server.pushers {
		pushers[k] = v
	}
	server.pushersLock.RUnlock()
	return
}

//GetPusherSize get size
func (server *Server) GetPusherSize() (size int) {
	server.pushersLock.RLock()
	size = len(server.pushers)
	server.pushersLock.RUnlock()
	return
}
