// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

type PullSession struct {
	*ClientSession
}

type PullSessionTimeout struct {
	ConnectTimeoutMS int
	PullTimeoutMS    int
	ReadAVTimeoutMS  int
}

func NewPullSession(timeout PullSessionTimeout) *PullSession {
	return &PullSession{
		ClientSession: NewClientSession(CSTPullSession, ClientSessionTimeout{
			ConnectTimeoutMS: timeout.ConnectTimeoutMS,
			DoTimeoutMS:      timeout.PullTimeoutMS,
			ReadAVTimeoutMS:  timeout.ReadAVTimeoutMS,
		}),
	}
}

// 阻塞直到连接断开或发生错误
//
// @param onReadAVMsg: 回调结束后，内存块会被 PullSession 重复使用
func (s *PullSession) Pull(rawURL string, onReadAVMsg OnReadAVMsg) error {
	s.onReadAVMsg = onReadAVMsg
	if err := s.doWithTimeout(rawURL); err != nil {
		return err
	}
	return s.WaitLoop()
}
