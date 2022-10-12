package serverconfig

type ApiConfig struct {
	Port          int    `mapstructure:"port"`
	JWTSign       string `mapstructure:"jwt_sign"`
	JWTIssue      string `mapstructure:"jwt_issue_user"`
	TokenExpire   int    `mapstructure:"token_expire"`

	// chainconfig
	SDKPath      string `mapstructure:"sdk_path"`
	SysPrivate   string `mapstructure:"sys_private"`
	SysPublic    string `mapstructure:"sys_public"`
}
