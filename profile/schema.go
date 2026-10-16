package profile

type Profile struct {
	Name          string `json:"name" example:"hk"`
	RemoteAddress string `json:"remoteAddress" example:"tcp://host:port"`
	Password      string `json:"password" example:"change-me"`
}
