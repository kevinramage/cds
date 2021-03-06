package sdk

// DefaultValues contains default user values for init DB
type DefaultValues struct {
	DefaultGroupName string
}

// ConfigUser struct.
type ConfigUser struct {
	URLUI  string `json:"url.ui"`
	URLAPI string `json:"url.api"`
}

type TCPServer struct {
	Addr               string  `toml:"addr" default:"" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
	Port               int     `toml:"port" default:"8090" json:"port"`
	GlobalTCPRateLimit float64 `toml:"globalTCPRateLimit" default:"2048000" comment:"Rate limit (B/s) for incoming logs" json:"globalTCPRateLimit"`
}
