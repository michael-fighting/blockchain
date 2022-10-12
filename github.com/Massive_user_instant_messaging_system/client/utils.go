package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/Massive_user_instant_messaging_system/common/message"
	"net"
)

func readPkg(conn net.Conn) (mes message.Message, err error) {

	buf := make([]byte, 8096)
	fmt.Println("读取客户端发送的数据...")
	//conn.Read在conn没有被关闭的情况下，才会zus
	//如果客户端关闭了conn则不会阻塞
	_, err = conn.Read(buf[:4])
	if err != nil {
		//err = errors.New("read pkg header error")
		return
	}
	//fmt.Println("读到的buf=", buf[:4])

	//根据buf[:4]转成一个uint32类型
	var pkgLen uint32
	pkgLen = binary.BigEndian.Uint32(buf[0:4]) //因为uint32是4个字节，接口返回的是发送成功的字节数

	//根据pkgLen读取conn消息内容到buf中去
	n, err := conn.Read(buf[:pkgLen])
	if n != int(pkgLen) || err != nil {
		//err = errors.New("read pkg body error")
		return
	}

	//把buf反序列化成-> message.Message
	//技术就是一层窗户纸
	err = json.Unmarshal(buf[:pkgLen], &mes)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err)
		return
	}
	return
}

func writePkg(conn net.Conn, data []byte) (err error) {
	//先发送一个长度给对方
	var pkgLen uint32
	pkgLen = uint32(len(data))
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[0:4], pkgLen)  //因为uint32是4个字节，接口返回的是发送成功的字节数
	//现在发送长度
	n, err := conn.Write(buf[:4])
	if n != 4 || err != nil{
		fmt.Println("conn.Write(bytes) err=" ,err)
	}

	//发送data本身
	n, err = conn.Write(data)
	if n != int(pkgLen) || err != nil{
		fmt.Println("conn.Write(bytes) err=" ,err)
		return
	}
	return
}
