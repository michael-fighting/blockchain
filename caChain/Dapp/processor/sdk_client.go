package processor

import (
	"sync"

	sdk "chainmaker.org/chainmaker/sdk-go/v2"

	"go.uber.org/zap"
)

var (
	sdkLogger        *zap.SugaredLogger
	createClientLock sync.Mutex
)

//// 创建sdk，userprivatekey为空，则使用系统密钥,系统密钥保存在
//func CreateChainClientWithKeyConf(sdkConfPath string) (*sdk.ChainClient, error) {
//	createClientLock.Lock()
//	defer createClientLock.Unlock()
//	var cc *sdk.ChainClient
//	var err error
//
//	cc, err = sdk.NewChainClient(
//		sdk.WithConfPath(sdkConfPath),
//		sdk.WithChainClientLogger(sdkLogger),
//	)
//
//	if err != nil {
//		return nil, err
//	}
//	// Enable certificate compression
//	if cc.GetAuthType() == sdk.PermissionedWithCert {
//		err = cc.EnableCertHash()
//	}
//	if err != nil {
//		return nil, err
//	}
//	return cc, nil
//}

// CreateChainClientWithSDKConf create a chain client with sdk config file path
func CreateChainClientWithSDKConf(sdkConfPath string) (*sdk.ChainClient, error) {
	createClientLock.Lock()
	defer createClientLock.Unlock()
	cc, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfPath),
	)
	if err != nil {
		return nil, err
	}

	// Enable certificate compression
	if cc.GetAuthType() == sdk.PermissionedWithCert {
		if err := cc.EnableCertHash(); err != nil {
			return nil, err
		}
	}
	return cc, nil
}