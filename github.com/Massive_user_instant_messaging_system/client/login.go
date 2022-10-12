package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/Massive_user_instant_messaging_system/common/message"
	"net"
)

//写一个函数，完成登录
func login(userId int, userPwd string) (err error){

	//下一个就要开始定协议
	//fmt.Printf("userId = %d userPwd = %s\n", userId, userPwd)
	//return nil

	//1.链接到服务器
	conn, err := net.Dial("tcp", "localhost:8889")
	if err != nil{
		fmt.Println("net.Dial err=", err)
		return
	}
	//延时关闭
	defer conn.Close()
	//2.准备通过conn发送消息给服务
	var mes message.Message
	mes.Type = message.LoginMesType
	//3. 创建一个LoginMes 结构体
	var loginMes message.LoginMes
	loginMes.UserId = userId
	loginMes.UserPwd = userPwd

	//4. 将loginMes 序列化
	data, err := json.Marshal(loginMes)
	if err != nil{
		fmt.Println("json.Marshal err=", err)
		return
	}
	//5. 把data赋给mes.Data字段
	mes.Data = string(data)

	//6. 将mes进行序列化
	data, err = json.Marshal(mes)
	if err != nil{
		fmt.Println("json.Marshal err=", err)
		return
	}

	// 7.到这个时候 data就是我们要发送的消息
	// 7.1 先把data的长度发送给服务器
	// 先获取到data的长度->转成一个表示长度的byte切片
	var pkgLen uint32
	pkgLen = uint32(len(data))
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[0:4], pkgLen)  //因为uint32是4个字节，接口返回的是发送成功的字节数
	//现在发送长度
	n, err := conn.Write(buf[:4])
	if n != 4 || err != nil{
		fmt.Println("conn.Write(bytes) err=" ,err)
	}
	//fmt.Printf("客户端 发送消息长度=%d\n 内容=%s", len(data), string(data))

	//发送消息本身
	_, err = conn.Write(data)
	if n != 4 || err != nil{
		fmt.Println("conn.Write(data) err=" ,err)
	}

	//休眠20s
	//time.Sleep(20 * time.Second)
	//fmt.Println("休眠了20..")
	//这里还需要处理服务器端返回的消息
	mes, err = readPkg(conn)

	if err != nil{
		fmt.Println("readPkg(conn) err=", err)
		return
	}

	//将mes的Data部分反序列化成 LoginResMes
	var loginResMes message.LoginResMes
	 err = json.Unmarshal([]byte(mes.Data), &loginResMes)
	if loginResMes.Code == 200{
		fmt.Println("登录成功")
	}else if loginResMes.Code == 500{
		fmt.Println(loginResMes.Error)
	}

	return
}
