package amf

import (
	"bytes"
	"encoding/binary"
)

// GetType GetType
func GetType(i byte) string {
	switch i {
	case 0x00:
		return "number"
	case 0x01:
		return "boolean"
	case 0x02:
		return "string"
	case 0x03:
		// 需要再次处理
		return "object"
	case 0x04:
		return "movieclip"
	case 0x05:
		return "null"
	case 0x06:
		return "undefined"
	case 0x07:
		return "reference"
	case 0x08:
		// 需要再次处理
		return "ecma-array"
	case 0x09:
		return "object-end"
	case 0x0A:
		return "strict-array"
	case 0x0B:
		return "date"
	}
	return ""
}

// GetValue GetValue
func GetValue(buf []byte) ([]byte, interface{}) {
	var case1 = buf[0]
	buf = buf[1:]
	switch case1 {
	case 0x00: //number 8
		var result float64
		binary.Read(bytes.NewReader(buf), binary.BigEndian, &result)
		if uint32(len(buf)) == 8 {
			buf = []byte{}
		} else {
			buf = buf[8:]
		}
		return buf, result
	case 0x01: //boolean
		if buf[0] == 1 {
			return buf[1:], true
		} else {
			return buf[1:], false
		}
	case 0x02: //string
		var len1 = binary.BigEndian.Uint32(append([]byte{0, 0}, buf[0:2]...))
		var str = string(buf[2:][:len1])
		if uint32(len(buf)) == 2+len1 {
			buf = []byte{}
		} else {
			buf = buf[2+len1:]
		}
		return buf, str
	case 0x03: //object
		var map1 = make(map[string]interface{})
		for {
			var val interface{}
			// key len
			var len1 = binary.BigEndian.Uint32(append([]byte{0, 0}, buf[0:2]...))
			buf = buf[2:] // 去除
			// key
			var key = string(buf[:len1])
			buf = buf[len1:] // 去除
			buf, val = GetValue(buf)
			map1[key] = val
			if string(buf) == string([]byte{0, 0, 9}) {
				buf = []byte{}
				break
			}
		}
		return buf, map1
	case 0x04:
	case 0x05:
	case 0x06:
	case 0x07:
	case 0x08:
	case 0x09: //object-end

	case 0x0A:
	case 0x0B: //date
	default:
		return buf, nil
	}
	return buf, nil
}
