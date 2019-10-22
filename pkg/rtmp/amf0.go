// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

// amf0.go
// @pure
// 提供amf0格式的编码与解码的操作

import (
	"bytes"
	"errors"
	"io"

	"github.com/q191201771/naza/pkg/bele"
	log "github.com/q191201771/naza/pkg/nazalog"
)

var (
	ErrAMFInvalidType = errors.New("lal.rtmp: invalid amf0 type")
	ErrAMFTooShort    = errors.New("lal.rtmp: too short to unmarshal amf0 data")
)

const (
	AMF0TypeMarkerNumber     = uint8(0x00)
	AMF0TypeMarkerBoolean    = uint8(0x01)
	AMF0TypeMarkerString     = uint8(0x02)
	AMF0TypeMarkerObject     = uint8(0x03)
	AMF0TypeMarkerNull       = uint8(0x05)
	AMF0TypeMarkerObjectEnd  = uint8(0x09)
	AMF0TypeMarkerLongString = uint8(0x0c)

	// 还没用到的类型
	//AMF0TypeMarkerMovieclip   = uint8(0x04)
	//AMF0TypeMarkerUndefined   = uint8(0x06)
	//AMF0TypeMarkerReference   = uint8(0x07)
	//AMF0TypeMarkerEcmaArray   = uint8(0x08)
	//AMF0TypeMarkerStrictArray = uint8(0x0a)
	//AMF0TypeMarkerData        = uint8(0x0b)
	//AMF0TypeMarkerUnsupported = uint8(0x0d)
	//AMF0TypeMarkerRecordset   = uint8(0x0e)
	//AMF0TypeMarkerXmlDocument = uint8(0x0f)
	//AMF0TypeMarkerTypedObject = uint8(0x10)
)

var AMF0TypeMarkerObjectEndBytes = []byte{0, 0, AMF0TypeMarkerObjectEnd}

type ObjectPair struct {
	Key   string
	Value interface{}
}

type amf0 struct{}

var AMF0 amf0

func (amf0) WriteNumber(writer io.Writer, val float64) error {
	if _, err := writer.Write([]byte{AMF0TypeMarkerNumber}); err != nil {
		return err
	}
	return bele.WriteBE(writer, val)
}

func (amf0) WriteString(writer io.Writer, val string) error {
	if len(val) < 65536 {
		if _, err := writer.Write([]byte{AMF0TypeMarkerString}); err != nil {
			return err
		}
		if err := bele.WriteBE(writer, uint16(len(val))); err != nil {
			return err
		}
	} else {
		if _, err := writer.Write([]byte{AMF0TypeMarkerLongString}); err != nil {
			return err
		}
		if err := bele.WriteBE(writer, uint32(len(val))); err != nil {
			return err
		}
	}
	_, err := writer.Write([]byte(val))
	return err
}

func (amf0) WriteNull(writer io.Writer) error {
	_, err := writer.Write([]byte{AMF0TypeMarkerNull})
	return err
}

func (amf0) WriteBoolean(writer io.Writer, b bool) error {
	if _, err := writer.Write([]byte{AMF0TypeMarkerBoolean}); err != nil {
		return err
	}
	v := uint8(0)
	if b {
		v = 1
	}
	_, err := writer.Write([]byte{v})
	return err
}

func (amf0) WriteObject(writer io.Writer, objs []ObjectPair) error {
	if _, err := writer.Write([]byte{AMF0TypeMarkerObject}); err != nil {
		return err
	}
	for i := 0; i < len(objs); i++ {
		if err := bele.WriteBE(writer, uint16(len(objs[i].Key))); err != nil {
			return err
		}
		if _, err := writer.Write([]byte(objs[i].Key)); err != nil {
			return err
		}
		switch objs[i].Value.(type) {
		case string:
			if err := AMF0.WriteString(writer, objs[i].Value.(string)); err != nil {
				return err
			}
		case int:
			if err := AMF0.WriteNumber(writer, float64(objs[i].Value.(int))); err != nil {
				return err
			}
		case bool:
			if err := AMF0.WriteBoolean(writer, objs[i].Value.(bool)); err != nil {
				return err
			}
		default:
			log.Panicf("unknown value type. i=%d, v=%v", i, objs[i].Value)
		}
	}
	_, err := writer.Write(AMF0TypeMarkerObjectEndBytes)
	return err
}

// read类型的方法集合
//
// 从输入参数<b>切片中读取函数名所指定的amf类型数据
// 注意，方法内部不会修改输入参数<b>切片的内容
//
// 返回值如无特殊说明，则
// 第1个参数为读取出的所指定类型的数据
// 第2个参数为读取时从<b>消耗的字节大小
// 第3个参数error，如果不等于nil，表示读取失败

func (amf0) ReadStringWithoutType(b []byte) (string, int, error) {
	if len(b) < 2 {
		return "", 0, ErrAMFTooShort
	}
	l := int(bele.BEUint16(b))
	if l > len(b)-2 {
		return "", 0, ErrAMFTooShort
	}
	return string(b[2 : 2+l]), 2 + l, nil
}

func (amf0) ReadLongStringWithoutType(b []byte) (string, int, error) {
	if len(b) < 4 {
		return "", 0, ErrAMFTooShort
	}
	l := int(bele.BEUint32(b))
	if l > len(b)-4 {
		return "", 0, ErrAMFTooShort
	}
	return string(b[4 : 4+l]), 4 + l, nil
}

func (amf0) ReadString(b []byte) (val string, l int, err error) {
	if len(b) < 1 {
		return "", 0, ErrAMFTooShort
	}
	switch b[0] {
	case AMF0TypeMarkerString:
		val, l, err = AMF0.ReadStringWithoutType(b[1:])
		l++
	case AMF0TypeMarkerLongString:
		val, l, err = AMF0.ReadLongStringWithoutType(b[1:])
		l++
	default:
		err = ErrAMFInvalidType
	}
	return
}

func (amf0) ReadNumber(b []byte) (float64, int, error) {
	if len(b) < 9 {
		return 0, 0, ErrAMFTooShort
	}
	if b[0] != AMF0TypeMarkerNumber {
		return 0, 0, ErrAMFInvalidType
	}
	return bele.BEFloat64(b[1:]), 9, nil
}

func (amf0) ReadBoolean(b []byte) (bool, int, error) {
	if len(b) < 2 {
		return false, 0, ErrAMFTooShort
	}
	if b[0] != AMF0TypeMarkerBoolean {
		return false, 0, ErrAMFInvalidType
	}
	return b[1] != 0x0, 2, nil
}

func (amf0) ReadNull(b []byte) (int, error) {
	if len(b) < 1 {
		return 0, ErrAMFTooShort
	}
	if b[0] != AMF0TypeMarkerNull {
		return 0, ErrAMFInvalidType
	}
	return 1, nil
}

func (amf0) ReadObject(b []byte) (map[string]interface{}, int, error) {
	if len(b) < 1 {
		return nil, 0, ErrAMFTooShort
	}
	if b[0] != AMF0TypeMarkerObject {
		return nil, 0, ErrAMFInvalidType
	}

	index := 1
	obj := make(map[string]interface{})
	for {
		if len(b)-index >= 3 && bytes.Equal(b[index:index+3], AMF0TypeMarkerObjectEndBytes) {
			return obj, index + 3, nil
		}

		k, l, err := AMF0.ReadStringWithoutType(b[index:])
		if err != nil {
			return nil, 0, err
		}
		index += l
		if len(b)-index < 1 {
			return nil, 0, ErrAMFTooShort
		}
		vt := b[index]
		switch vt {
		case AMF0TypeMarkerString:
			v, l, err := AMF0.ReadString(b[index:])
			if err != nil {
				return nil, 0, err
			}
			obj[k] = v
			index += l
		case AMF0TypeMarkerBoolean:
			v, l, err := AMF0.ReadBoolean(b[index:])
			if err != nil {
				return nil, 0, err
			}
			obj[k] = v
			index += l
		case AMF0TypeMarkerNumber:
			v, l, err := AMF0.ReadNumber(b[index:])
			if err != nil {
				return nil, 0, err
			}
			obj[k] = v
			index += l
		default:
			log.Panicf("unknown type. vt=%d", vt)
		}
	}

	//panic("should not reach here.")
	//return nil, 0, nil
}
