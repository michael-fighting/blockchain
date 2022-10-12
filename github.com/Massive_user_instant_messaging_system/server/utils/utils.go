package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/Massive_user_instant_messaging_system/common/message"
	"net"
)

//这里将这些方法关联到结构体中
type Transfer struct {
	//分析它应该有哪些字段
	Conn net.Conn
	Buf [8096]byte   //传输时使用的缓冲
}

func (this *Transfer) ReadPkg() (mes message.Message, err error) {

	//buf := make([]byte, 8096)
	fmt.Println("读取客户端发送的数据...")
	//conn.Read在conn没有被关闭的情况下，才会zus
	//如果客户端关闭了conn则不会阻塞
	_, err = this.Conn.Read(this.Buf[:4])
	if err != nil {
		//err = errors.New("read pkg header error")
		return
	}
	//fmt.Println("读到的buf=", buf[:4])

	//根据buf[:4]转成一个uint32类型
	var pkgLen uint32
	pkgLen = binary.BigEndian.Uint32(this.Buf[0:4]) //因为uint32是4个字节，接口返回的是发送成功的字节数

	//根据pkgLen读取conn消息内容到buf中去
	n, err := this.Conn.Read(this.Buf[:pkgLen])
	if n != int(pkgLen) || err != nil {
		//err = errors.New("read pkg body error")
		return
	}

	//把buf反序列化成-> message.Message
	//技术就是一层窗户纸
	err = json.Unmarshal(this.Buf[:pkgLen], &mes)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err)
		return
	}
	return
}

func (this *Transfer) WritePkg(data []byte) (err error) {
	//先发送一个长度给对方
	var pkgLen uint32
	pkgLen = uint32(len(data))
	//var buf [4]byte
	binary.BigEndian.PutUint32(this.Buf[0:4], pkgLen)  //因为uint32是4个字节，接口返回的是发送成功的字节数
	//现在发送长度
	n, err := this.Conn.Write(this.Buf[:4])
	if n != 4 || err != nil{
		fmt.Println("conn.Write(bytes) err=" ,err)
	}

	//发送data本身
	n, err = this.Conn.Write(data)
	if n != int(pkgLen) || err != nil{
		fmt.Println("conn.Write(bytes) err=" ,err)
		return
	}
	return
}
