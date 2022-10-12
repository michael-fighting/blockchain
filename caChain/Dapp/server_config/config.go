package serverconfig

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	GlobalCFG *ServerConfig
)

type ServerConfig struct {
	ApiCFG    *ApiConfig    `mapstructure:"api_server"`
	LogCFG    *LogConfig    `mapstructure:"log"`
	DBCFG     *MySqlConfig  `mapstructure:"store_mysql"`
	//WeChatCFG *WechatConfig `mapstructure:"wechat"`
	//NFTList   []NFTItem     `mapstructure:"nft_list"`
}

func initViper(confPath string) (*viper.Viper, error) {
	cmViper := viper.New()
	cmViper.SetConfigFile(confPath)
	err := cmViper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	return cmViper, nil
}

func ReadConfigFile(confPath string) error {
	var (
		err       error
		confViper *viper.Viper
	)
	if confViper, err = initViper(confPath); err != nil {
		return fmt.Errorf("load sdk config failed, %s", err)
	}
	GlobalCFG = &ServerConfig{}
	if err = confViper.Unmarshal(GlobalCFG); err != nil {
		return fmt.Errorf("unmarshal config file failed, %s", err)
	}
	fmt.Fprintf(os.Stdout, "config is : %+v ", GlobalCFG)
	return nil
}
