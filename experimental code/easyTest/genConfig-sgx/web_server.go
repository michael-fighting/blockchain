package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Disable console color, you don't need console color when writing the logs to file.
	gin.DisableConsoleColor()

	// Use a file for logging
	f, _ := os.Create("/root/gin.log")
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.POST("/setConf", func(c *gin.Context) {
		if err := c.ShouldBindJSON(&MyNet); err != nil {
			panic(err)
		}
		GenConfigV2()
		c.JSON(200, map[string]interface{}{
			"code": 0,
			"msg":  "ok",
		})
	})
	r.GET("/getConf", func(c *gin.Context) {
		fmt.Printf("remoteIp:%v\n", c.ClientIP())
		c.JSON(200, map[string]interface{}{
			"code": 0,
			"data": ConfMap[c.RemoteIP()],
			"msg":  "ok",
		})
	})
	r.GET("/getAllConf", func(c *gin.Context) {
		fmt.Printf("remoteIp:%v\n", c.ClientIP())
		c.JSON(200, map[string]interface{}{
			"code": 0,
			"data": ConfMap,
			"msg":  "ok",
		})
	})
	// Listen and serve on 0.0.0.0:8090
	r.Run(":8090")
}
