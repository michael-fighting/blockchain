package apiserver

import (
	"Dapp/db"
	localutil "Dapp/local_util"
	"Dapp/processor"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"time"
)

//使用协程订阅项目发布合约事件
func IssueSubject(vpr *processor.VoteProcessor, ginContext *gin.Context){

	//从智能合约订阅事件中获取参数,发布项目
	c, err := vpr.AdminClient.SubscribeContractEvent(ginContext, 0, -1, "contract1", "issue project")
	if err != nil {
		ginContext.JSONP(http.StatusOK, gin.H{
			"code": localutil.CodeSubscribeContractEventFail,
			"msg":  localutil.SubscribeContractEventFail,
		})
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case event, ok := <-c:
			if !ok {
				fmt.Println("chan is close!")
				return
			}
			if event == nil {
				log.Fatalln("require not nil")
			}
			//逐个从合约事件中取所需的字段
			contractEventInfo, ok := event.(*common.ContractEventInfo)
			if !ok {
				log.Fatalln("require true")
			}
			id, err := strconv.Atoi(contractEventInfo.EventData[0])
			if err != nil {
				ginContext.JSONP(http.StatusOK, gin.H{
					"code": localutil.CodeIllegalIdParams,
					"msg":  localutil.IllegalIdParams,
				})
				return
			}
			picUrl := contractEventInfo.EventData[1]
			title := contractEventInfo.EventData[2]
			//开始时间戳转换为时间
			startTime := contractEventInfo.EventData[3]
			iStartTime,_ := strconv.Atoi(startTime)
			stm := time.Unix(int64(iStartTime), 0)
			//结束时间戳转换为时间
			endTime := contractEventInfo.EventData[4]
			iEndTime,_ := strconv.Atoi(endTime)
			etm := time.Unix(int64(iEndTime), 0)
			scenery := contractEventInfo.EventData[5]
			//获取投票详情参数,并将其反序列化
			strVoteDetail := contractEventInfo.EventData[6]
			var items     []*ItemInfo
			err = json.Unmarshal([]byte(strVoteDetail), items)
			if err != nil {
				ginContext.JSONP(http.StatusOK, gin.H{
					"code": localutil.CodeUnmarshalFail,
					"msg":  localutil.UnmarshalFail,
				})
				return
			}
			//将投票写入数据库
			dbVote := &db.Vote{
				CommonField: db.CommonField{
					CreatedAt: time.Now(),
					Id:        int64(id),
				},
				Title:      title,
				State:      "投票中",
				PicUrl:     picUrl,
				Start_time: stm,
				End_time: etm,
				Rule: "单选",
				Scenery: scenery,
			}
			// 将投票写入数据库
			err = db.CreateVote(dbVote)
			if err != nil {
				ginContext.JSONP(http.StatusOK, gin.H{
					"code": localutil.CodeSaveVoteFail,
					"msg":  localutil.SaveVoteFail,
				})
				return
			}
			for _, v := range items {
				dbVoteDetail := &db.VoteDetail{
					CommonField: db.CommonField{
						CreatedAt: time.Now(),
						Id:        v.Id,
					},
					VoteId: int64(id),
					PicItemUrl: v.PicUrl,
					Description: v.Desc,
				}
				// 将投票项详情写入数据库
				err = db.CreateVoteDetail(dbVoteDetail)
				if err != nil {
					ginContext.JSONP(http.StatusOK, gin.H{
						"code": localutil.CodeSaveVoteDetailFail,
						"msg":  localutil.SaveVoteDetailFail,
					})
					return
				}
			}


			fmt.Printf("recv contract event [%d] => %+v\n", contractEventInfo.BlockHeight, contractEventInfo)

			//if err := client.Stop(); err != nil {
			//	return
			//}
			//return
		case <-ctx.Done():
			return
		}
	}
}


//使用协程订阅项目发布合约事件
func Vote(vpr *processor.VoteProcessor, ginContext *gin.Context){

	//从智能合约订阅事件中获取参数,发布项目
	c, err := vpr.AdminClient.SubscribeContractEvent(ginContext, 0, -1, "contract2", "vote")
	if err != nil {
		ginContext.JSONP(http.StatusOK, gin.H{
			"code": localutil.CodeSubscribeContractEventFail,
			"msg":  localutil.SubscribeContractEventFail,
		})
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case event, ok := <-c:
			if !ok {
				fmt.Println("chan is close!")
				return
			}
			if event == nil {
				log.Fatalln("require not nil")
			}
			//逐个从合约事件中取所需的字段
			contractEventInfo, ok := event.(*common.ContractEventInfo)
			if !ok {
				log.Fatalln("require true")
			}
			projectId, err := strconv.Atoi(contractEventInfo.EventData[0])
			if err != nil {
				ginContext.JSONP(http.StatusOK, gin.H{
					"code": localutil.CodeIllegalIdParams,
					"msg":  localutil.IllegalIdParams,
				})
				return
			}
			projectItemId, err := strconv.Atoi(contractEventInfo.EventData[1])
			if err != nil {
				ginContext.JSONP(http.StatusOK, gin.H{
					"code": localutil.CodeIllegalIdParams,
					"msg":  localutil.IllegalIdParams,
				})
				return
			}
			//更新投票数量
			err = db.UpdateVoteDetailNumber(int64(projectId), int64(projectItemId))
			if err != nil {
				ginContext.JSONP(http.StatusOK, gin.H{
					"code": localutil.CodeUpdateVoteDetailNumberFail,
					"msg":  localutil.UpdateVoteDetailNumberFail,
				})
				return
			}



			fmt.Printf("recv contract event [%d] => %+v\n", contractEventInfo.BlockHeight, contractEventInfo)

			//if err := client.Stop(); err != nil {
			//	return
			//}
			//return
		case <-ctx.Done():
			return
		}
	}
}
