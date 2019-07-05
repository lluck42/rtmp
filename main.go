package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"tcp-server/amf"
	"tcp-server/chunk"
	"tcp-server/command"
	"time"
)

func main() {
	var err error
	_ = err
	fmt.Println("server has been start===>")
	tcpAddr, _ := net.ResolveTCPAddr("tcp", ":1937")
	//服务器端一般不定位具体的客户端套接字
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	// ConnMap := make(map[string]*net.TCPConn)
	// 一个链接可以有多个 chunk
	var Chunks = make(chunk.Chunks)

	for {
		tcpConn, _ := tcpListener.AcceptTCP()
		defer tcpConn.Close()
		go func() {
			fmt.Println("连接的客户端信息：", tcpConn.RemoteAddr().String())
			if handShake(tcpConn) != nil {
				fmt.Println("=========验证失败=========")
				return //退出协程
			}
			var ChunkSize uint32 = 4096
			// 拼接 chunk 包
			for {

				// chunk
				var chunk1 = Chunks.GetChunks(tcpConn, ChunkSize)
				// 判断完整性
				fmt.Println("bodysizie:", binary.BigEndian.Uint32(append([]byte{0}, chunk1.BodySize[:]...)))

				if chunk1.RemainBodySize != 0 {
					continue
				}

				// 响应一些事件
				switch int(chunk1.TypeID[0]) {
				case 0x01:
					// 设置 ChunkSize
					ChunkSize = binary.BigEndian.Uint32(chunk1.Body)
					fmt.Println("设置chunksize", ChunkSize)
				case 0x14:
					// command
					var val interface{}
					chunk1.Body, val = amf.GetValue(chunk1.Body)
					fmt.Println(val)
					switch val {
					case "connect":
						command.SendWindowAcknowledgementSize(tcpConn, 5000000)
						command.SendChunkSize(tcpConn)
						command.SendBandWidth(tcpConn)
						command.SendSuccess(tcpConn)
					case "createStream":
						command.SendResult(tcpConn)
					case "publish":
						command.SendOnstatus(tcpConn)
						// read
					}
					// AMF0编码
				default:
				}
				// 取值
				// var val interface{}
				// for len(chunk1.Body) != 0 {
				// 	chunk1.Body, val = amf.GetValue(chunk1.Body)
				// 	fmt.Println(val)
				// 	_ = val
				// }
			}
		}()
		// ConnMap[tcpConn.RemoteAddr().String()] = tcpConn
		fmt.Println("================================================================================================")
	}
}

// handShake 握手
func handShake(tcpConn *net.TCPConn) error {
	var err error
	// 读取版本号
	var version = make([]byte, 1)
	_, err = tcpConn.Read(version)
	if err != nil {
		return err
	}
	// 写入版本号
	_, err = tcpConn.Write([]byte{3})
	if err != nil {
		return err
	}
	// 读取 c1
	var c1Bytes = make([]byte, 1528+8)
	_, err = tcpConn.Read(c1Bytes)
	if err != nil {
		return err
	}
	var c1 = BytesToC1(c1Bytes)
	_ = c1
	// 写入 s1
	var s1 = makeS1()
	_, err = tcpConn.Write(C1ToBytes(s1))
	if err != nil {
		return err
	}
	// 读取 c2 并验证
	var c2Bytes = make([]byte, 1536)
	_, err = tcpConn.Read(c2Bytes)
	// fmt.Println(c2Bytes[len(c2Bytes)-10:])
	if err != nil {
		return err
	}
	var c2 = BytesToC1(c2Bytes)
	if c2.Random != s1.Random {
		return errors.New("c2 验证失败！")
	}
	// 写入s2
	var s2 = makeC2(c1)
	_, err = tcpConn.Write(C1ToBytes(s2))
	if err != nil {
		return err
	}
	return nil
}

// Int32ToBytes int32 转 bytes
func Int32ToBytes(i int32) []byte {
	var buf = make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

// BytesToInt32 bytes 转 int32
func BytesToInt32(buf []byte) int32 {
	return int32(binary.BigEndian.Uint32(buf))
}

// 返回 s1
func makeS1() C1 {
	var s1 C1
	copy(s1.Time1[:4], makeTime4Bytes())
	copy(s1.Time2[:4], []byte{0, 0, 0, 0})
	copy(s1.Random[:1528], CreateRandomBytes(1528))
	return s1
}

// 返回 c2 s2
func makeC2(c1 C1) C1 {
	var c2 C1
	copy(c2.Time1[:4], makeTime4Bytes())
	c2.Time2 = c1.Time1
	c2.Random = c1.Random
	return c2
}

// 当前时间戳 截取 为 4 byte
func makeTime4Bytes() []byte {
	i64 := time.Now().UnixNano() / 1e6 // 微秒
	s := fmt.Sprintf("%013d", i64)     // 保留13位数字
	// 转为字符串
	i, _ := strconv.Atoi(s[3:])   // 转为数字
	return Int32ToBytes(int32(i)) // 相对时间戳
}

// CreateRandomBytes 随机 []byte
func CreateRandomBytes(len int) []byte {
	rand.Seed(time.Now().UnixNano()) // 真随机
	var by []byte
	for i := 0; i < len; i++ {
		randomInt := rand.Intn(1 << 8) //  一个字节 8 bit
		by = append(by, byte(randomInt))
	}
	return by
}

// C1 s1 c2 s2 struct
type C1 struct {
	Time1  [4]byte
	Time2  [4]byte
	Random [1528]byte
}

// C1ToBytes C1 转为 bytes
func C1ToBytes(c1 C1) []byte {
	var rt []byte
	rt = append(rt, c1.Time1[:]...)
	rt = append(rt, c1.Time2[:]...)
	rt = append(rt, c1.Random[:]...)
	return rt
}

// BytesToC1 bytes 转 C1
func BytesToC1(b []byte) C1 {
	var c1 C1
	copy(c1.Time1[:4], b[0:4])
	copy(c1.Time2[:4], b[4:8])
	copy(c1.Random[:1528], b[8:1528+8])
	return c1
}

// =====================================================

//
