package profile

type Region struct {
	Name    string `json:"name,omitempty"`
	Display string `json:"display,omitempty"`
}

type CloudProvider struct {
	Name    string `json:"name,omitempty"`
	Display string `json:"display,omitempty"`
}

type Profile struct {
	Name            string         `json:"name" example:"default"`
	CloudProvider   *CloudProvider `json:"cloudProvider" example:"aliyun"`
	AccessKeyId     string         `json:"accessKeyId"`
	AccessKeySecret string         `json:"accessKeySecret"`
	AccountId       string         `json:"accountId"`
	Region          *Region        `json:"region" example:"{Name:cn-hongkong,Display:中国香港}"`
	ServiceName     string         `json:"serviceName" example:"api2"`
	FunctionName    string         `json:"functionName" example:"dt2"`
	TriggerName     string         `json:"triggerName" example:"ws2"`
	Password        string         `json:"password" example:"pass123"`
	Image           string         `json:"image" example:"registry-vpc.cn-hongkong.aliyuncs.com/hjcrocks/detour2:latest"`
}
