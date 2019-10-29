// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
)

var Trans trans

type trans struct {
}

// 注意，tag -> message [nocopy]
func (t trans) FLVTag2RTMPMsg(tag httpflv.Tag) (msg rtmp.AVMsg) {
	msg.Header.MsgLen = tag.Header.DataSize
	msg.Header.MsgTypeID = tag.Header.T
	msg.Header.MsgStreamID = rtmp.MSID1
	switch tag.Header.T {
	case httpflv.TagTypeMetadata:
		msg.Header.CSID = rtmp.CSIDAMF
	case httpflv.TagTypeAudio:
		msg.Header.CSID = rtmp.CSIDAudio
	case httpflv.TagTypeVideo:
		msg.Header.CSID = rtmp.CSIDVideo
	}
	msg.Header.Timestamp = tag.Header.Timestamp
	msg.Header.TimestampAbs = tag.Header.Timestamp
	msg.Message = tag.Raw[11 : 11+msg.Header.MsgLen]
	return
}

// 注意，message -> tag [copy]
func (t trans) RTMPMsg2FLVTag(msg rtmp.AVMsg) *httpflv.Tag {
	var tag httpflv.Tag
	tag.Header.T = msg.Header.MsgTypeID
	tag.Header.DataSize = msg.Header.MsgLen
	tag.Header.Timestamp = msg.Header.TimestampAbs
	tag.Raw = httpflv.PackHTTPFLVTag(msg.Header.MsgTypeID, msg.Header.TimestampAbs, msg.Message)
	return &tag
}
