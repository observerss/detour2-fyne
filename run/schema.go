package run

type Run struct {
	ProfileName    string `json:"profileName,omitempty"`
	Socks5Port     string `json:"socks5Port,omitempty"`
	HTTPPort       string `json:"httpPort,omitempty"`
	PoolSize       int    `json:"poolSize,omitempty"`
	LocalPort      string `json:"localPort,omitempty"`
	RunOnStartup   bool   `json:"runOnStartup,omitempty"`
	UseGlobalProxy bool   `json:"useGlobalProxy,omitempty"`
}
