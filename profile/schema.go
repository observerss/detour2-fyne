package profile

type Profile struct {
	Name          string `json:"name" example:"hk"`
	RemoteAddress string `json:"remoteAddress" example:"tcp://example.com:7777"`
	Password      string `json:"password" example:"change-me"`
}
