package apiserver

import tokenmiddleware "Dapp/api_server/token_middleware"

func (srv *ApiServer) registerRouters() {
	//srv.Router.GET("/v1/hello", srv.TestCode)        //srv.Hello)
	//srv.Router.POST("/v1/login", srv.LogIn)          //注册登录接口

	authApis := srv.Router.Group("/v1/log", tokenmiddleware.JWTAuth())
	//var authApis *gin.RouterGroup
	authApis.POST("/get_voteList", srv.GetVoteList)               // 获取投票列表
	authApis.POST("/get_voteDetail", srv.GetVoteDetail)         // 获取投票详情
	//srv.Router.POST("/upload", srv.UploadFile)         // 上传文件
}
