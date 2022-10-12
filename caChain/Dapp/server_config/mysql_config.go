package serverconfig

type MySqlConfig struct {
	UserName       string `mapstructure:"user_name"`
	Password       string `mapstructure:"password"`
	DbAddress      string `mapstructure:"db_address"`
	Database       string `mapstructure:"database"`
	MaxOpenConns   int    `mapstructure:"max_open"`
	MaxIdleConns   int    `mapstructure:"max_idle"`
	MaxIdleSeconds int    `mapstructure:"max_idle_seconds"`
	MaxLifeSeconds int    `mapstructure:"max_life_seconds"`
}
