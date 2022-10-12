package apiserver

import (
	"Dapp/logger"
	"Dapp/processor"
	serverconfig "Dapp/server_config"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	GlobalWaitGroup sync.WaitGroup
)

type ApiServer struct {
	Router         *gin.Engine
	Log            *zap.SugaredLogger
	ProxyProcessor *processor.VoteProcessor
	//SystemKey      *models.UserKey
	//PayPrivateKey  *ecdsa.PrivateKey
}

func NewApiServer(proxy *processor.VoteProcessor) *ApiServer {
	return &ApiServer{
		Router:         gin.New(),
		Log:            logger.NewLogger("api", serverconfig.GlobalCFG.LogCFG),
		ProxyProcessor: proxy,
		//SystemKey:      systemKey,
		//PayPrivateKey:  payp,
	}
}

func (srv *ApiServer) Listen() {
	//gin.SetMode(gin.ReleaseMode)
	srv.Router.Use(gin.Recovery())
	srv.registerRouters()
	httpSRV := &http.Server{
		Addr:    fmt.Sprintf(":%d", serverconfig.GlobalCFG.ApiCFG.Port),
		Handler: srv.Router,
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		// 监听http端口
		srv.Log.Infof("http service start , port %s ", httpSRV.Addr)
		if err := httpSRV.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srv.Log.Errorf("http server start err , %s ", err.Error())
		}
	}()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	// 优雅退出
	if err := httpSRV.Shutdown(ctx); err != nil {
		srv.Log.Errorf("http service failed shutdown ,err %s ", err.Error())
		return
	}
	GlobalWaitGroup.Wait()
	srv.Log.Info("http service success shutdown")
}
