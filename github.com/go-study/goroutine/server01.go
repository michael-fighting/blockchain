// Clock1 is a TCP server that periodically writes the time.
package main

//func main() {
//	listener, err := net.Listen("tcp", "localhost:8000")  //定义一个在本地网络地址localhost:8000监听的listener，即创建一个服务端
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for {
//		conn, err := listener.Accept()  //建立一个连接到该接口的连接
//		if err != nil {
//			log.Print(err) // e.g., connection aborted
//			continue
//		}
//		handleConn(conn) // handle one connection at a time
//	}
//}
//
//func handleConn(c net.Conn) {
//	defer c.Close()   //关闭连接
//	for {
//		_, err := io.WriteString(c, time.Now().Format("15:04:05\n"))  //将time.Now().Format("15:04:05\n")的内容写到连接中
//		if err != nil {
//			return // e.g., client disconnected
//		}
//		time.Sleep(1 * time.Second)
//	}
//}