package processor

import (
	"fmt"
	"Dapp/logger"
	serverconfig "Dapp/server_config"

	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"go.uber.org/zap"
)

var (
	GlobalVoteProcessor *VoteProcessor
)

type VoteProcessor struct {
	Log *zap.SugaredLogger
	//UserClientMap sync.Map         // 用户id 到client的映射
	AdminClient   *sdk.ChainClient // admin client
	//SystemUser    *models.User     // 系统用户
	//SystemUserKey *models.UserKey  // 系统用户的key
	//NFTList       []models.NFTInfo
}

// 根据用户id 和用户key 来构造client
// func (bp *BusinessProcessor) GetUserClient(userid int64, userKey *models.UserKey) (*sdk.ChainClient, error) {
// 	var chainClient *sdk.ChainClient
// 	// 查一下指定的链有没有正常运行的grpc客户端
// 	chainO, chainOk := bp.UserClientMap.Load(userid)
// 	if !chainOk {
// 		bp.Log.Warnf("GetUserClient no userid(%d) ", userid)
// 		// 根据链的sdk配置构造一个grpc客户端对象
// 		sdkCli, sdkCliErr := CreateChainClientWithKeyConf(serverconfig.GlobalCFG.ApiCFG.SDKPath, userKey.PrivateKey)
// 		if sdkCliErr != nil {
// 			bp.Log.Errorf("GetUserClient create chain error(%s),userid(%d)  ", sdkCliErr.Error(), userid)
// 			return nil, sdkCliErr
// 		}
// 		chainClient = sdkCli

// 	} else {
// 		chainClient, _ = chainO.(*sdk.ChainClient)
// 	}
// 	//  todo 暂时保留 ，ping一下，看看是否是通的
// 	// chainClient.GetChainMakerServerVersion()
// 	_, rpcErr := chainClient.GetChainConfig()
// 	if rpcErr != nil {
// 		bp.Log.Errorf("GetUserClient  get chain cfg error(%s) ", rpcErr.Error())
// 		_ = chainClient.Stop()          // 需要停掉
// 		bp.UserClientMap.Delete(userid) // ping不通，删除
// 		return nil, rpcErr
// 	}
// 	bp.Log.Debugf("GetUserClient userid(%d) ", userid)
// 	if !chainOk {
// 		bp.UserClientMap.Store(userid, chainClient)
// 	}
// 	return chainClient, nil
// }

func InitProcessor() *VoteProcessor {

	sdkLogger = logger.NewLogger("sdk", serverconfig.GlobalCFG.LogCFG)
	GlobalVoteProcessor := &VoteProcessor{
		Log:           logger.NewLogger("processor", serverconfig.GlobalCFG.LogCFG),
	}

	adminClient, adminError := CreateChainClientWithSDKConf(serverconfig.GlobalCFG.ApiCFG.SDKPath)
	if adminError != nil {
		panic(adminError.Error())
	}

	rpcCfg, rpcError := adminClient.GetChainConfig()
	if rpcError != nil {
		panic(fmt.Sprintf("server lost connection with chain , error : %s", rpcError.Error()))
	}
	GlobalVoteProcessor.Log.Infof("admin get chain config ,", zap.Any("chaincfg", rpcCfg))
	GlobalVoteProcessor.AdminClient = adminClient
	return GlobalVoteProcessor
}
