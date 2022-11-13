package run

type Run struct {
	ProfileName    string `json:"profileName,omitempty"`
	LocalPort      string `json:"localPort,omitempty"`
	RunOnStartup   bool   `json:"runOnStartup,omitempty"`
	UseGlobalProxy bool   `json:"useGlobalProxy,omitempty"`
}
