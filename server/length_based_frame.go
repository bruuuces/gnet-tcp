package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

type LengthFieldPrepender struct {
	lenFieldLen LenFieldLen
}

func (p *LengthFieldPrepender) Encode(_ *TCPSession, msgData []byte) ([]byte, error) {
	lenFieldBytes, err := p.encodeMsgLen(msgData)
	if err != nil {
		return nil, err
	}
	return append(lenFieldBytes, msgData...), nil
}

func (p *LengthFieldPrepender) encodeMsgLen(msgData []byte) ([]byte, error) {
	msgDataLen := len(msgData)
	lenFieldBytes := make([]byte, p.lenFieldLen)
	switch p.lenFieldLen {
	case LenFieldLenInt8:
		if msgDataLen > math.MaxInt8 {
			return nil, fmt.Errorf("msgData length overflow, len: %v, max: %v", msgDataLen, math.MaxInt8)
		}
		lenFieldBytes[0] = byte(msgDataLen)
	case LenFieldLenInt16:
		if msgDataLen > math.MaxInt16 {
			return nil, fmt.Errorf("msgData length overflow, len: %v, max: %v", msgDataLen, math.MaxInt16)
		}
		binary.BigEndian.PutUint16(lenFieldBytes, uint16(msgDataLen))
	case LenFieldLenInt32:
		if msgDataLen > math.MaxInt32 {
			return nil, fmt.Errorf("msgData length overflow, len: %v, max: %v", msgDataLen, math.MaxInt32)
		}
		binary.BigEndian.PutUint32(lenFieldBytes, uint32(msgDataLen))
	}
	return lenFieldBytes, nil
}

func NewLengthFieldPrepender(lenFieldLen int) (*LengthFieldPrepender, error) {
	l, err := validateLenFieldLen(lenFieldLen)
	if err != nil {
		return nil, err
	}
	return &LengthFieldPrepender{
		lenFieldLen: l,
	}, nil
}

type LengthFieldBasedFrameDecoder struct {
	lenFieldLen LenFieldLen
	maxFrameLen int
}

func (d *LengthFieldBasedFrameDecoder) Decode(_ *TCPSession, reader io.Reader) ([]byte, error) {
	msgDataLen, err := d.decodeMsgLen(reader)
	if err != nil {
		return nil, err
	}
	msgData, err := d.decodeMsgData(reader, msgDataLen)
	if err != nil {
		return nil, err
	}
	return msgData, nil
}

func (d *LengthFieldBasedFrameDecoder) decodeMsgLen(reader io.Reader) (int, error) {
	lenFieldBytes := make([]byte, d.lenFieldLen)
	readLen, err := io.ReadFull(reader, lenFieldBytes)
	if err == io.ErrUnexpectedEOF {
		return 0, err
	} else if err != nil {
		return 0, fmt.Errorf("decode message length field error, readLen: %v, err: %v", readLen, err)
	} else if readLen != int(d.lenFieldLen) {
		return 0, fmt.Errorf("decode message length field length error, actual: %v, expect: %v", readLen, d.lenFieldLen)
	}
	var msgDataLen int
	switch d.lenFieldLen {
	case LenFieldLenInt8:
		msgDataLen = int(lenFieldBytes[0])
	case LenFieldLenInt16:
		msgDataLen = int(binary.BigEndian.Uint16(lenFieldBytes))
	case LenFieldLenInt32:
		msgDataLen = int(binary.BigEndian.Uint32(lenFieldBytes))
	}
	if msgDataLen < 0 {
		return 0, fmt.Errorf("msgDataLen cannot be a negative number, msgDataLen: %v", msgDataLen)
	} else if msgDataLen > d.maxFrameLen {
		return 0, fmt.Errorf("message too long, msgDataLen: %v, maxFrameLen: %v", msgDataLen, d.maxFrameLen)
	}
	return msgDataLen, nil
}

func (d *LengthFieldBasedFrameDecoder) decodeMsgData(reader io.Reader, msgDataLen int) ([]byte, error) {
	msgData := make([]byte, msgDataLen)
	readLen, err := io.ReadFull(reader, msgData)
	if err == io.ErrUnexpectedEOF {
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("decode message data error, readLen: %v, err: %v", readLen, err)
	} else if readLen != msgDataLen {
		return nil, fmt.Errorf("decode message length field length error, actual: %v, expect: %v", readLen, msgDataLen)
	}
	return msgData, nil
}

func NewLengthFieldBasedFrameDecoder(lenFieldLen int, maxFrameLen int) (*LengthFieldBasedFrameDecoder, error) {
	l, err := validateLenFieldLen(lenFieldLen)
	if err != nil {
		return nil, err
	}
	return &LengthFieldBasedFrameDecoder{
		lenFieldLen: l,
		maxFrameLen: maxFrameLen,
	}, nil
}

var ErrMessageLenFieldLen = errors.New("invalid `lenFieldLen`")

// validateLenFieldLen 验证消息长度字段的字节长度是否有效
func validateLenFieldLen(lenFieldLen int) (LenFieldLen, error) {
	l := LenFieldLen(lenFieldLen)
	if l.IsValid() {
		return l, nil
	}
	return 0, ErrMessageLenFieldLen
}

// LenFieldLen 消息长度字段的字节长度
type LenFieldLen int

const (
	LenFieldLenInt8  LenFieldLen = 1
	LenFieldLenInt16 LenFieldLen = 2
	LenFieldLenInt32 LenFieldLen = 4
)

// IsValid 获取消息长度字段的字节长度是否有效
func (t LenFieldLen) IsValid() bool {
	return t == LenFieldLenInt8 || t == LenFieldLenInt16 || t == LenFieldLenInt32
}
