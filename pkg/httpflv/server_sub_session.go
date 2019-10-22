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
	url2 "net/url"
	"strings"
	"time"

	"github.com/q191201771/naza/pkg/connection"

	log "github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

var flvHTTPResponseHeaderStr = "HTTP/1.1 200 OK\r\n" +
	"Cache-Control: no-cache\r\n" +
	"Content-Type: video/x-flv\r\n" +
	"Connection: close\r\n" +
	"Expires: -1\r\n" +
	"Pragma: no-cache\r\n" +
	"\r\n"

var flvHTTPResponseHeader = []byte(flvHTTPResponseHeaderStr)

var flvHeaderBuf13 = []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x0, 0x0, 0x0, 0x09, 0x0, 0x0, 0x0, 0x0}

type SubSession struct {
	UniqueKey string

	StartTick  int64
	StreamName string
	AppName    string
	URI        string
	Headers    map[string]string

	IsFresh     bool
	WaitKeyNalu bool

	conn connection.Connection
}

func NewSubSession(conn net.Conn) *SubSession {
	uk := unique.GenUniqueKey("FLVSUB")
	log.Infof("lifecycle new SubSession. [%s] remoteAddr=%s", uk, conn.RemoteAddr().String())
	return &SubSession{
		UniqueKey:   uk,
		IsFresh:     true,
		WaitKeyNalu: true,
		conn: connection.New(conn, func(option *connection.Option) {
			option.ReadBufSize = readBufSize
			option.WriteChanSize = wChanSize
			option.WriteTimeoutMS = subSessionWriteTimeoutMS
		}),
	}
}

// TODO chef: read request timeout
func (session *SubSession) ReadRequest() (err error) {
	session.StartTick = time.Now().Unix()

	defer func() {
		if err != nil {
			session.Dispose(err)
		}
	}()

	var firstLine string
	_, firstLine, session.Headers, err = parseHTTPHeader(session.conn)
	if err != nil {
		return
	}

	items := strings.Split(string(firstLine), " ")
	if len(items) != 3 || items[0] != "GET" {
		err = ErrHTTPFLV
		return
	}

	session.URI = items[1]
	var urlObj *url2.URL
	urlObj, err = url2.Parse(session.URI)
	if err != nil {
		return
	}
	if !strings.HasSuffix(urlObj.Path, ".flv") {
		err = ErrHTTPFLV
		return
	}

	items = strings.Split(urlObj.Path, "/")
	if len(items) != 3 {
		err = ErrHTTPFLV
		return
	}
	session.AppName = items[1]
	items = strings.Split(items[2], ".")
	if len(items) < 2 {
		err = ErrHTTPFLV
		return
	}
	session.StreamName = items[0]

	return nil
}

func (session *SubSession) RunLoop() error {
	buf := make([]byte, 128)
	_, err := session.conn.Read(buf)
	return err
}

func (session *SubSession) WriteHTTPResponseHeader() {
	log.Infof("<----- http response header. [%s]", session.UniqueKey)
	session.WriteRawPacket(flvHTTPResponseHeader)
}

func (session *SubSession) WriteFlvHeader() {
	log.Infof("<----- http flv header. [%s]", session.UniqueKey)
	session.WriteRawPacket(flvHeaderBuf13)
}

func (session *SubSession) WriteTag(tag *Tag) {
	session.WriteRawPacket(tag.Raw)
}

func (session *SubSession) WriteRawPacket(pkt []byte) {
	_, _ = session.conn.Write(pkt)
}

func (session *SubSession) Dispose(err error) {
	_ = session.conn.Close()
}
