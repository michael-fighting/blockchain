package apiserver

import (
	"Dapp/db"
	localutil "Dapp/local_util"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (srv *ApiServer) GetVoteDetail(ginContext *gin.Context) {
	var (
		totalCount     int
		offset         int //偏移指定在开始返回记录之前要跳过的记录数量
		limit          int
		voteId         int
		voteDetailList []*db.VoteDetail
	)

	//判定投票详情参数是否合法
	getVoteDetailBody := BindGetVoteDetailBodyHandler(ginContext)
	if getVoteDetailBody == nil || !getVoteDetailBody.IsLegal() {
		srv.Log.Error("BindGetVoteListHandler fail")
		ginContext.JSONP(http.StatusOK, gin.H{
			"code": localutil.CodeIllegalVoteDetailParamam,
			"msg":  localutil.IllegalVoteDetailParamam,
		})
		return
	}
	offset = getVoteDetailBody.PageNum * getVoteDetailBody.PageSize
	limit = getVoteDetailBody.PageSize
	totalCount, voteDetailList, err := db.GetVoteDetail(offset, limit)
	if err != nil {
		srv.Log.Error("GetVoteList fail")
		ginContext.JSONP(http.StatusOK, gin.H{
			"code": localutil.CodeGetVoteDetailFail,
			"msg":  localutil.GetVoteDetailFail,
		})
		return
	}
	//由voteId获取投票的基础项，如创建时间、结束时间等
	voteId = getVoteDetailBody.voteId
	vote, err := db.GetVoteByUserId(voteId)
	if err != nil {
		srv.Log.Error("GetVoteByUserId fail")
		ginContext.JSONP(http.StatusOK, gin.H{
			"code": localutil.CodeGetVoteByUserIdFail,
			"msg":  localutil.GetVoteByUserIdFail,
		})
		return
	}
	ginContext.JSONP(http.StatusOK, gin.H{
		"code":           localutil.CodeSuccess,
		"msg":            localutil.SuccessDealed,
		"totalCount":     totalCount,
		"voteDetailList": voteDetailList,
		"title":          vote.Title,
		"start_time":     vote.Start_time,
		"end_time":       vote.End_time,
		"rule":           vote.Rule,
		"scenery":        vote.Scenery,
	})

}
