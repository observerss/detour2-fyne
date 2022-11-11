package profile

import "fmt"

var AliyunRegions = []*Region{
	{Name: "cn-hongkong", Display: "中国香港"},
	{Name: "ap-southeast-1", Display: "新加坡"},
	{Name: "ap-southeast-2", Display: "澳大利亚（悉尼）"},
	{Name: "ap-southeast-3", Display: "马来西亚（吉隆坡）"},
	{Name: "ap-southeast-5", Display: "印度尼西亚（雅加达）"},
	{Name: "ap-southeast-6", Display: "菲律宾（马尼拉）"},
	{Name: "ap-southeast-7", Display: "泰国（曼谷）"},
	{Name: "ap-south-1", Display: "印度（孟买）"},
	{Name: "ap-northeast-1", Display: "日本（东京）"},
	{Name: "ap-northeast-2", Display: "韩国（首尔）"},
	{Name: "us-west-1", Display: "美国（硅谷）"},
	{Name: "us-east-1", Display: "美国（弗吉尼亚）"},
	{Name: "eu-central-1", Display: "德国（法兰克福）"},
	{Name: "eu-west-1", Display: "英国（伦敦）"},
	{Name: "me-east-1", Display: "阿联酋（迪拜）"},
}

var CloudProviders = []*CloudProvider{
	{Name: "aliyun", Display: "阿里云"},
}

func GetAliyunRegions() []string {
	regions := make([]string, 0)
	for _, r := range AliyunRegions {
		regions = append(regions, r.Display)
	}
	return regions
}

func GetCloudProviers() []string {
	providers := make([]string, 0)
	for _, p := range CloudProviders {
		providers = append(providers, p.Display)
	}
	return providers
}

func GetProviderValue(s string) (string, error) {
	for _, p := range CloudProviders {
		if p.Display == s {
			return p.Name, nil
		}
	}
	return "", fmt.Errorf("provider name not found: %s", s)
}

func GetProvider(s string) (*CloudProvider, error) {
	for _, p := range CloudProviders {
		if p.Display == s {
			return p, nil
		}
	}
	return nil, fmt.Errorf("provider name not found: %s", s)
}

func GetAliyunRegionValue(s string) (string, error) {
	for _, r := range AliyunRegions {
		if r.Display == s {
			return r.Name, nil
		}
	}
	return "", fmt.Errorf("region name not found: %s", s)
}

func GetAliyunRegion(s string) (*Region, error) {
	for _, r := range AliyunRegions {
		if r.Display == s {
			return r, nil
		}
	}
	return nil, fmt.Errorf("region name not found: %s", s)
}

func GetImageByRegion(region string) string {
	return fmt.Sprintf("registry-vpc.%s.aliyuncs.com/hjcrocks/detour2:latest", region)
}
