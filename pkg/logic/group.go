// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	log "github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

type Group struct {
	UniqueKey string

	appName    string
	streamName string

	exitChan chan struct{}

	mutex                sync.Mutex
	pubSession           *rtmp.ServerSession
	pullSession          *rtmp.PullSession
	rtmpSubSessionSet    map[*rtmp.ServerSession]struct{}
	httpflvSubSessionSet map[*httpflv.SubSession]struct{}
	// rtmp chunk格式
	metadata        []byte
	avcKeySeqHeader []byte
	aacSeqHeader    []byte
	// httpflv tag格式
	// TODO chef: 如果没有开启httpflv监听，可以不做格式转换，节约CPU资源
	metadataTag        *httpflv.Tag
	avcKeySeqHeaderTag *httpflv.Tag
	aacSeqHeaderTag    *httpflv.Tag
}

var _ rtmp.PubSessionObserver = &Group{}

func NewGroup(appName string, streamName string) *Group {
	uk := unique.GenUniqueKey("RTMPGROUP")
	log.Infof("lifecycle new group. [%s] appName=%s, streamName=%s", uk, appName, streamName)
	return &Group{
		UniqueKey:            uk,
		appName:              appName,
		streamName:           streamName,
		exitChan:             make(chan struct{}, 1),
		rtmpSubSessionSet:    make(map[*rtmp.ServerSession]struct{}),
		httpflvSubSessionSet: make(map[*httpflv.SubSession]struct{}),
	}
}

func (group *Group) RunLoop() {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-group.exitChan:
			break
		case <-t.C:
			//noop
		}
	}
}

func (group *Group) Dispose(err error) {
	log.Infof("lifecycle dispose group. [%s]", group.UniqueKey)
	group.exitChan <- struct{}{}

	group.mutex.Lock()
	defer group.mutex.Unlock()
	if group.pubSession != nil {
		group.pubSession.Dispose()
	}
	for session := range group.rtmpSubSessionSet {
		session.Dispose()
	}
}

func (group *Group) AddRTMPPubSession(session *rtmp.ServerSession) bool {
	log.Debugf("add PubSession into group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	if group.pubSession != nil {
		log.Errorf("PubSession already exist in group. [%s] old=%s, new=%s", group.UniqueKey, group.pubSession.UniqueKey, session.UniqueKey)
		return false
	}

	group.pubSession = session
	group.mutex.Unlock()
	session.SetPubSessionObserver(group)
	return true
}

func (group *Group) DelRTMPPubSession(session *rtmp.ServerSession) {
	log.Debugf("del PubSession from group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.pubSession = nil
	group.metadata = nil
	group.avcKeySeqHeader = nil
	group.aacSeqHeader = nil
	group.metadataTag = nil
	group.avcKeySeqHeaderTag = nil
	group.aacSeqHeaderTag = nil
}

func (group *Group) AddRTMPSubSession(session *rtmp.ServerSession) {
	log.Debugf("add SubSession into group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.rtmpSubSessionSet[session] = struct{}{}

	// TODO chef: 多长没有拉流session存在的功能
	//group.turnToEmptyTick = 0
}

func (group *Group) DelRTMPSubSession(session *rtmp.ServerSession) {
	log.Debugf("del SubSession from group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	delete(group.rtmpSubSessionSet, session)
}

func (group *Group) AddHTTPFLVSubSession(session *httpflv.SubSession) {
	log.Debugf("add httpflv SubSession into group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	session.WriteHTTPResponseHeader()
	session.WriteFLVHeader()

	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.httpflvSubSessionSet[session] = struct{}{}
	log.Debugf("< AddHTTPFLVSubSession. [%s] [%s] %d", group.UniqueKey, session.UniqueKey, len(group.httpflvSubSessionSet))
}

func (group *Group) DelHTTPFLVSubSession(session *httpflv.SubSession) {
	log.Debugf("del httpflv SubSession from group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	delete(group.httpflvSubSessionSet, session)
}

func (group *Group) IsTotalEmpty() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.pubSession == nil && len(group.rtmpSubSessionSet) == 0 && len(group.httpflvSubSessionSet) == 0
}

func (group *Group) IsInExist() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.pubSession != nil
}

// PubSession or PullSession
func (group *Group) ReadRTMPAVMsgCB(header rtmp.Header, timestampAbs uint32, message []byte) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	// TODO chef: 这两个函数部分功能重合，需要重构
	group.broadcastRTMP2RTMP(header, timestampAbs, message)
	group.broadcastRTMP2FLV(header, timestampAbs, message)
}

func (group *Group) broadcastRTMP2RTMP(header rtmp.Header, timestampAbs uint32, message []byte) {
	//log.Infof("%+v", header)
	// # 1. 设置好头部信息
	var currHeader rtmp.Header
	currHeader.MsgLen = uint32(len(message))
	currHeader.Timestamp = timestampAbs
	currHeader.MsgTypeID = header.MsgTypeID
	currHeader.MsgStreamID = rtmp.MSID1
	switch header.MsgTypeID {
	case rtmp.TypeidDataMessageAMF0:
		currHeader.CSID = rtmp.CSIDAMF
		//prevHeader = nil
	case rtmp.TypeidAudio:
		currHeader.CSID = rtmp.CSIDAudio
		//prevHeader = group.prevAudioHeader
	case rtmp.TypeidVideo:
		currHeader.CSID = rtmp.CSIDVideo
		//prevHeader = group.prevVideoHeader
	}

	var absChunks []byte

	// # 2. 广播。遍历所有sub session，决定是否转发
	for session := range group.rtmpSubSessionSet {
		// ## 2.1. 一个message广播给多个sub session时，只做一次chunk切割
		if absChunks == nil {
			absChunks = rtmp.Message2Chunks(message, &currHeader)
		}

		// ## 2.2. 如果是新的sub session，发送已缓存的信息
		if session.IsFresh {
			// 发送缓存的头部信息
			if group.metadata != nil {
				session.AsyncWrite(group.metadata)
			}
			if group.avcKeySeqHeader != nil {
				session.AsyncWrite(group.avcKeySeqHeader)
			}
			if group.aacSeqHeader != nil {
				session.AsyncWrite(group.aacSeqHeader)
			}
			session.IsFresh = false
		}

		// ## 2.3. 判断当前包的类型，以及sub session的状态，决定是否发送并更新sub session的状态
		switch header.MsgTypeID {
		case rtmp.TypeidDataMessageAMF0:
			session.AsyncWrite(absChunks)
		case rtmp.TypeidAudio:
			session.AsyncWrite(absChunks)
		case rtmp.TypeidVideo:
			if session.WaitKeyNalu {
				if message[0] == 0x17 && message[1] == 0x0 {
					session.AsyncWrite(absChunks)
				}
				if message[0] == 0x17 && message[1] == 0x1 {
					session.AsyncWrite(absChunks)
					session.WaitKeyNalu = false
				}
			} else {
				session.AsyncWrite(absChunks)
			}

		}

	}

	// # 3. 缓存 metadata 和 avc key seq header 和 aac seq header
	// 由于可能没有订阅者，所以message可能还没做chunk切割，所以这里要做判断是否做chunk切割
	switch header.MsgTypeID {
	case rtmp.TypeidDataMessageAMF0:
		if absChunks == nil {
			absChunks = rtmp.Message2Chunks(message, &currHeader)
		}
		log.Debugf("cache metadata. [%s]", group.UniqueKey)
		group.metadata = absChunks
		group.metadataTag = Trans.RTMPMsg2FLVTag(header, timestampAbs, message)
	case rtmp.TypeidVideo:
		// TODO chef: magic number
		if message[0] == 0x17 && message[1] == 0x0 {
			if absChunks == nil {
				absChunks = rtmp.Message2Chunks(message, &currHeader)
			}
			log.Debugf("cache avc key seq header. [%s]", group.UniqueKey)
			group.avcKeySeqHeader = absChunks
			group.avcKeySeqHeaderTag = Trans.RTMPMsg2FLVTag(header, timestampAbs, message)
		}
	case rtmp.TypeidAudio:
		if (message[0]>>4) == 0x0a && message[1] == 0x0 {
			if absChunks == nil {
				absChunks = rtmp.Message2Chunks(message, &currHeader)
			}
			log.Debugf("cache aac seq header. [%s]", group.UniqueKey)
			group.aacSeqHeader = absChunks
			group.aacSeqHeaderTag = Trans.RTMPMsg2FLVTag(header, timestampAbs, message)
		}
	}
}

func (group *Group) broadcastRTMP2FLV(header rtmp.Header, timestampAbs uint32, message []byte) {
	var currTag *httpflv.Tag
	for session := range group.httpflvSubSessionSet {
		// ## 1.1. 将当前 message 转换成 tag 格式
		if currTag == nil {
			currTag = Trans.RTMPMsg2FLVTag(header, timestampAbs, message)
		}

		// ## 2.2. 如果是新的sub session，发送已缓存的信息
		if session.IsFresh {
			// 发送缓存的头部信息
			if group.metadataTag != nil {
				session.WriteTag(group.metadataTag)
			}
			if group.avcKeySeqHeaderTag != nil {
				session.WriteTag(group.avcKeySeqHeaderTag)
			}
			if group.aacSeqHeaderTag != nil {
				session.WriteTag(group.aacSeqHeaderTag)
			}
			session.IsFresh = false
		}

		// ## 2.3. 判断当前包的类型，以及sub session的状态，决定是否发送并更新sub session的状态
		switch header.MsgTypeID {
		case rtmp.TypeidDataMessageAMF0:
			session.WriteTag(currTag)
		case rtmp.TypeidAudio:
			session.WriteTag(currTag)
		case rtmp.TypeidVideo:
			if session.WaitKeyNalu {
				if message[0] == 0x17 && message[1] == 0x0 {
					session.WriteTag(currTag)
				}
				if message[0] == 0x17 && message[1] == 0x1 {
					session.WriteTag(currTag)
					session.WaitKeyNalu = false
				}
			} else {
				session.WriteTag(currTag)
			}

		}
	}
}
