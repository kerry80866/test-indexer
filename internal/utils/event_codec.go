package utils

import (
	"encoding/binary"
	"fmt"
	"google.golang.org/protobuf/proto"
)

// EncodeEvent 将 protobuf 消息编码为带事件类型前缀的二进制数据：
// - 前 4 字节为事件类型（uint32，小端序）
// - 后续为 protobuf 序列化数据（使用 MarshalAppend）
func EncodeEvent(eventType uint32, msg proto.Message) ([]byte, error) {
	const extraBuffer = 32 // 多预留一些空间，降低 MarshalAppend 触发扩容的概率

	// 预估 protobuf 编码大小
	size := proto.Size(msg)

	// 分配缓冲区：
	// - 前 4 字节写入事件类型
	// - 后续用于 protobuf 编码追加
	buf := make([]byte, 4, 4+size+extraBuffer)
	binary.LittleEndian.PutUint32(buf[:4], eventType)

	opts := proto.MarshalOptions{Deterministic: true}
	result, err := opts.MarshalAppend(buf, msg)
	if err != nil {
		return nil, fmt.Errorf("EncodeEvent: marshal %T: %w", msg, err)
	}

	// 极小概率下 MarshalAppend 会触发扩容，导致前缀丢失，做防御性修正
	if &result[0] != &buf[0] {
		newBuf := make([]byte, 4, len(result)+4)
		binary.LittleEndian.PutUint32(newBuf[:4], eventType)
		result = append(newBuf, result[4:]...)
	}

	return result, nil
}
