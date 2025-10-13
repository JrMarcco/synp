package compression

import "github.com/gobwas/ws/wsflate"

// Config 为压缩配置。
type Config struct {
	Enabled                 bool `yaml:"enabled"`
	ServerMaxWindowBits     int  `yaml:"server_max_window_bits"`
	ServerNoContextTakeover bool `yaml:"server_no_context_takeover"`
	ClientMaxWindowBits     int  `yaml:"client_max_window_bits"`
	ClientNoContextTakeover bool `yaml:"client_no_context_takeover"`
	Level                   int  `yaml:"level"`
}

// ToParamters 将 Config 转换为 wsflate 参数。
func (cfg *Config) ToParamters() wsflate.Parameters {
	return wsflate.Parameters{
		ServerMaxWindowBits:     wsflate.WindowBits(cfg.ServerMaxWindowBits),
		ServerNoContextTakeover: cfg.ServerNoContextTakeover,
		ClientMaxWindowBits:     wsflate.WindowBits(cfg.ClientMaxWindowBits),
		ClientNoContextTakeover: cfg.ClientNoContextTakeover,
	}
}

// State 为压缩状态，包含协商后的扩展信息和压缩参数。
type State struct {
	Enabled bool

	Ext    *wsflate.Extension
	Params wsflate.Parameters
}
