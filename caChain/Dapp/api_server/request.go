package apiserver

import (
	"github.com/gin-gonic/gin"
)

// 查询投票列表参数
type GetVoteListParams struct {
	PageNum  int
	PageSize int
}

// 查询投票列表参数
type GetVoteDetailParams struct {
	PageNum  int
	PageSize int
	voteId   int
}

type RequestBody interface {
	// IsLegal 是否合法
	IsLegal() bool
}

type ItemInfo struct {
	Id     int64
	PicUrl string
	Desc   string
}

func BindGetVoteListBodyHandler(ctx *gin.Context) *GetVoteListParams {
	var body = &GetVoteListParams{}
	if err := BindGetVoteListBody(ctx, body); err != nil {
		return nil
	}
	return body
}

func BindGetVoteListBody(ctx *gin.Context, body RequestBody) error {
	if err := ctx.ShouldBindJSON(body); err != nil {
		return err
	}
	return nil
}

func (getVoteListParams *GetVoteListParams) IsLegal() bool {
	if getVoteListParams.PageNum < 0 || getVoteListParams.PageSize == 0 {
		return false
	}
	return true
}

func BindGetVoteDetailBodyHandler(ctx *gin.Context) *GetVoteDetailParams {
	var body = &GetVoteDetailParams{}
	if err := BindGetVoteDetailBody(ctx, body); err != nil {
		return nil
	}
	return body
}

func BindGetVoteDetailBody(ctx *gin.Context, body RequestBody) error {
	if err := ctx.ShouldBindJSON(body); err != nil {
		return err
	}
	return nil
}

func (getVoteDetailParams *GetVoteDetailParams) IsLegal() bool {
	if getVoteDetailParams.PageNum < 0 || getVoteDetailParams.PageSize == 0 {
		return false
	}
	return true
}
