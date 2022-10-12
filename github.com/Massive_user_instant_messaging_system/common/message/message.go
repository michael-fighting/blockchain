package message

const (
	LoginMesType   =  "LoginMes"
	LoginResMesType    = "LoginResMes"
	RegisterMesType = "RegisterMes"
	RegisterResMesType = "RegisterResMes"
)

type Message struct {
	Type string  `json:"type"`//消息的类型
	Data string  `json:"data"`//消息的内容
}

//先定义两个具体的消息...需要再增加

type LoginMes struct {
	UserId int       `json:"userId"`//用户id
	UserPwd string      `json:"userPwd"`//用户密码
	UserName string  `json:"userName"`//用户名
}

type LoginResMes struct {
	Code int  `json:"code"`//返回状态码 500 表示该用户未注册 200表示登录成功
	Error string   `json:"error"`//返回错误信息

}

type RegisterMes struct {
	User User `json:"user"` //类型就是User结构体
}

type RegisterResMes struct {
	Code int  `json:"code"`//返回状态码 400 表示该用户已经占用 200表示注册成功
	Error string   `json:"error"`//返回错误信息
}
