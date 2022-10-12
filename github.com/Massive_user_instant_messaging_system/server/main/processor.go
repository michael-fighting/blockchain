package main

import (
	"fmt"
	"github.com/Massive_user_instant_messaging_system/common/message"
	"github.com/Massive_user_instant_messaging_system/server/process"
	"github.com/Massive_user_instant_messaging_system/server/utils"
	"io"
	"net"
)

//先创建一个Processor的结构体
type Processor struct {
	Conn net.Conn
}

//编写一个ServerProcessMes函数
//功能：根据客户端发送消息种类不同，决定调用哪个函数来处理
func (this *Processor) serverProcessMes(mes *message.Message) (err error) {
	switch mes.Type {
	case message.LoginMesType:
		//处理登录的逻辑
		//创建一个UserProcess实例
		up := &process.UserProcess{
			Conn: this.Conn,
		}
		err = up.ServerProcessLogin(mes)
	case message.RegisterMesType:
	//处理注册
		up := &process.UserProcess{
			Conn: this.Conn,
		}
		err = up.ServerProcessRegister(mes)
	default:
		fmt.Println("消息类型不存在，无法处理...")

	}
	return
}

func (this *Processor) process2() (err error){
	//读取客户端发送的消息
	for {

		//这里我们将读取数据包，直接封装一个函数readPkg(),返回Message, Err
		//创建一个Transfer实例完成读包任务
		tf := &utils.Transfer{Conn: this.Conn}
		mes, err := tf.ReadPkg()

		if err != nil {
			if err == io.EOF {
				fmt.Println("客户端退出，服务器端也退出...")
				return err
			} else {
				fmt.Println("readPkg err=", err)
				return err
			}
		}
		//fmt.Println("mes=", mes)

		err = this.serverProcessMes( &mes)
		if err != nil {
			return err
		}

	}

}
