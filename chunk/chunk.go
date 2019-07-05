package chunk

import (
	"encoding/binary"
	"net"

	"github.com/imroc/biu"
)

// Chunk 1
type Chunk struct {
	RemainBodySize    uint32 // 剩余 bodysize
	ChunkStreamID     int
	Timestamp         []byte
	BodySize          []byte
	TypeID            []byte
	StreamID          []byte
	ExtendedTimeStamp []byte
	Body              []byte
}

// 上一个 chunk
var lastChunk Chunk

// Chunks 1
type Chunks map[int]Chunk

// GetChunkFmt GetChunkFmt
func GetChunkFmt(buf byte) string {
	return biu.ByteToBinaryString(buf)[0:2]
}

// GetMessageHeaderLength GetMessageHeaderLength
func GetMessageHeaderLength(buf byte) int {
	switch GetChunkFmt(buf) {
	case "00":
		return 11
	case "01":
		return 7
	case "10":
		return 3
	case "11":
		return 0
	default:
		return 0
	}
}

// GetChunkStreamIDAndLen GetChunkStreamIDAndLen
func GetChunkStreamIDAndLen(tcpConn *net.TCPConn) (cid int, l int) {
	var err error
	_ = err

	var buf = make([]byte, 1)
	_, err = tcpConn.Read(buf)
	l = GetMessageHeaderLength(buf[0])

	var str = biu.ByteToBinaryString(buf[0])[2:8]
	if str == "000001" {
		buf = make([]byte, 2)
		_, err = tcpConn.Read(buf)
		i := binary.LittleEndian.Uint16(buf)
		cid = (int)(i) + 64
	} else if str == "000000" {
		_, err = tcpConn.Read(buf)
		buf = append([]byte{0}, buf...)
		i := binary.BigEndian.Uint16(buf)
		cid = (int)(i) + 64
	} else {
		str = "[00" + str + "]"
		bs := biu.BinaryStringToBytes(str) //一个字节
		bs = append([]byte{0}, bs...)
		i := binary.BigEndian.Uint16(bs)
		cid = (int)(i)
	}

	return cid, l
}

// GetChunks byte 转 结构体
// 因为 header 长度不定，所以要直接读取 io
func (ch *Chunks) GetChunks(tcpConn *net.TCPConn, chunkSize uint32) (h Chunk) {
	//
	var err error
	_ = err
	var len1 int
	h.ChunkStreamID, len1 = GetChunkStreamIDAndLen(tcpConn)
	var buf []byte
	switch len1 {
	case 11:
		buf = make([]byte, len1)
		_, err = tcpConn.Read(buf)

		h.Timestamp = buf[0:3]
		h.BodySize = buf[3:6]
		h.TypeID = buf[6:7]
		h.StreamID = buf[7:11]
		break
	case 7:
		buf = make([]byte, 7)
		_, err = tcpConn.Read(buf)
		h.Timestamp = buf[0:3]
		h.BodySize = buf[3:6]
		h.TypeID = buf[6:7]
		break
	case 3:
		buf = make([]byte, len1)
		_, err = tcpConn.Read(buf)

		h.Timestamp = buf[0:3]
		break
	case 0:

	}

	if len(h.Timestamp) > 0 && binary.BigEndian.Uint32(append([]byte{0}, h.Timestamp[:]...)) == 0x00ffffff {
		buf = make([]byte, 4)
		_, err = tcpConn.Read(buf)
	}

	// var bodysize = binary.BigEndian.Uint32(append([]byte{0}, h.BodySize...))
	// 满足长度不因该加入队列
	// body

	if _, ok := (*ch)[h.ChunkStreamID]; ok {
		if chunkSize >= (*ch)[h.ChunkStreamID].RemainBodySize {
			chunkSize = (*ch)[h.ChunkStreamID].RemainBodySize
			h.RemainBodySize = 0
		} else {
			h.RemainBodySize = (*ch)[h.ChunkStreamID].RemainBodySize - chunkSize
		}

		buf = make([]byte, chunkSize)
		_, err = tcpConn.Read(buf)
		var chunkTmp = (*ch)[h.ChunkStreamID]
		chunkTmp.Body = append((*ch)[h.ChunkStreamID].Body, buf...)
		h = chunkTmp
	} else {
		// bodysize 省略取上一个 bodysize
		if len(h.BodySize) == 0 {
			h.BodySize = lastChunk.BodySize
		}
		// TypeID 省略取上一个 TypeID
		if len(h.TypeID) == 0 {
			h.TypeID = lastChunk.TypeID
		}

		var bodySize = binary.BigEndian.Uint32(append([]byte{0}, h.BodySize[:]...))
		if chunkSize >= bodySize {
			chunkSize = bodySize
			h.RemainBodySize = 0
			buf = make([]byte, chunkSize)
			_, err = tcpConn.Read(buf)
			h.Body = buf
		} else {
			h.RemainBodySize = bodySize - chunkSize
			buf = make([]byte, chunkSize)
			_, err = tcpConn.Read(buf)
			h.Body = buf
			(*ch)[h.ChunkStreamID] = h
		}
	}
	lastChunk = h // 保存上一个 chunk
	return h
}

// GetChunkBody 11
func GetChunkBody() {

}
