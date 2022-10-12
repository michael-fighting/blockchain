package apiserver

import (
	"Dapp/db"
	localutil "Dapp/local_util"
	"github.com/gin-gonic/gin"
	"net/http"
)

//获取投票列表
func (srv *ApiServer) GetVoteList(ginContext *gin.Context) {
	var (

		totalCount int
		offset     int //偏移指定在开始返回记录之前要跳过的记录数量
		limit      int
		voteList     []*db.Vote
	)

	//判定投票列表参数是否合法
	getVoteListBody := BindGetVoteListBodyHandler(ginContext)
	if getVoteListBody == nil || !getVoteListBody.IsLegal() {
		srv.Log.Error("BindGetVoteListHandler fail")
		ginContext.JSONP(http.StatusOK, gin.H{
			"code": localutil.CodeIllegalVoteListParamam,
			"msg":  localutil.IllegalVoteListParamam,
		})
		return
	}
	offset = getVoteListBody.PageNum * getVoteListBody.PageSize
	limit = getVoteListBody.PageSize
	totalCount, voteList, err := db.GetVoteList(offset, limit)
	if err != nil {
		srv.Log.Error("GetVoteList fail")
		ginContext.JSONP(http.StatusOK, gin.H{
			"code": localutil.CodeGetVoteListFail,
			"msg":  localutil.GetVoteListFail,
		})
		return
	}
	ginContext.JSONP(http.StatusOK, gin.H{
		"code": localutil.CodeSuccess,
		"msg":  localutil.SuccessDealed,
		"totalCount": totalCount,
		"voteList": voteList,
	})





}
