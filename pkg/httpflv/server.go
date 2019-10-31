// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"net"
	"sync"

	log "github.com/q191201771/naza/pkg/nazalog"
)

type ServerObserver interface {
	// 通知上层有新的拉流者
	// 返回值： true则允许拉流，false则关闭连接
	NewHTTPFLVSubSessionCB(session *SubSession) bool

	DelHTTPFLVSubSessionCB(session *SubSession)
}

type Server struct {
	obs  ServerObserver
	addr string

	m  sync.Mutex
	ln net.Listener
}

func NewServer(obs ServerObserver, addr string) *Server {
	return &Server{
		obs:  obs,
		addr: addr,
	}
}

func (server *Server) RunLoop() error {
	var err error

	server.m.Lock()
	server.ln, err = net.Listen("tcp", server.addr)
	server.m.Unlock()

	if err != nil {
		return err
	}

	log.Infof("start httpflv listen. addr=%s", server.addr)
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			return err
		}
		go server.handleConnect(conn)
	}
}

func (server *Server) Dispose() {
	server.m.Lock()
	defer server.m.Unlock()
	if server.ln == nil {
		return
	}
	if err := server.ln.Close(); err != nil {
		log.Error(err)
	}
}

func (server *Server) handleConnect(conn net.Conn) {
	log.Infof("accept a httpflv connection. remoteAddr=%v", conn.RemoteAddr())
	session := NewSubSession(conn)
	if err := session.ReadRequest(); err != nil {
		log.Errorf("read httpflv SubSession request error. [%s]", session.UniqueKey)
		return
	}
	log.Infof("-----> http request. [%s] uri=%s", session.UniqueKey, session.URI)

	if !server.obs.NewHTTPFLVSubSessionCB(session) {
		session.Dispose()
	}

	err := session.RunLoop()
	log.Debugf("httpflv sub session loop done. err=%v", err)
	server.obs.DelHTTPFLVSubSessionCB(session)
}
