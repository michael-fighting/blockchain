package serverconfig

type LogConfig struct {
	LogInConsole bool   `mapstructure:"log_in_console"`
	ShowColor    bool   `mapstructure:"show_color"`
	LogLevel     string `mapstructure:"log_level"`
	LogPath      string `mapstructure:"log_path"`
}
