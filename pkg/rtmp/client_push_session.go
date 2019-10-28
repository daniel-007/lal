// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

type PushSession struct {
	core *ClientSession
}

type PushSessionTimeout struct {
	ConnectTimeoutMS int
	PushTimeoutMS    int
	WriteAVTimeoutMS int
}

func NewPushSession(timeout PushSessionTimeout) *PushSession {
	return &PushSession{
		core: NewClientSession(CSTPushSession, func(option *ClientSessionOption) {
			option.ConnectTimeoutMS = timeout.ConnectTimeoutMS
			option.DoTimeoutMS = timeout.PushTimeoutMS
			option.WriteAVTimeoutMS = timeout.WriteAVTimeoutMS
		}),
	}
}

// 阻塞直到收到服务端返回的 rtmp publish 对应结果的信令或发生错误
func (s *PushSession) Push(rawURL string) error {
	return s.core.doWithTimeout(rawURL)
}

func (s *PushSession) AsyncWrite(msg []byte) error {
	return s.core.AsyncWrite(msg)
}

func (s *PushSession) Flush() error {
	return s.core.Flush()
}

func (s *PushSession) Dispose() {
	s.core.Dispose()
}

// TODO chef: ClientSession WaitLoop 接口也可以暴露出来
