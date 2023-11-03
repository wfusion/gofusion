package cases

import (
	"context"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

type appConf struct {
	DataService  DataService `yaml:"DataService"`
	AlgoService  AlgoService `yaml:"AlgoService"`
	InstanceSync cronStruct  `yaml:"InstanceSync"`
	Forecast     forecast    `yaml:"Forecast"`
}

func (*appConf) BeforeLoad(conf any) {
	log.Info(context.Background(), "get business configs before load: %+v", conf)
	log.Info(context.Background(), "get business configs json before load: %s", utils.MustJsonMarshal(conf))
	conf.(*appConf).AlgoService.Url = "before_load"
}

func (*appConf) AfterLoad(conf any) {
	log.Info(context.Background(), "get business configs after load: %+v", conf)
	log.Info(context.Background(), "get business configs json after load: %s", utils.MustJsonMarshal(conf))

	conf.(*appConf).DataService.Url = "after_load"
}

type azInfo struct {
	Url     string `yaml:"Url"`
	Timeout int    `yaml:"Timeout"`
}

type cronStruct struct {
	Enable  bool   `yaml:"Enable"`
	Crontab string `yaml:"Crontab"`
}

type adminRC struct {
	OSProjectDomainName string `yaml:"OSProjectDomainName"`
	OSUserDomainName    string `yaml:"OSUserDomainName"`
	OSProjectName       string `yaml:"OSProjectName"`
	OSUserName          string `yaml:"OSUserName"`
	OSPassword          string `yaml:"OSPassword"`
	OSAuthUrl           string `yaml:"OSAuthUrl"`
}

type GoStack struct {
	AZs     map[string]azInfo `yaml:"AZs"`
	AdminRC adminRC           `yaml:"AdminRC"`
}

type forecast struct {
	Enable  bool   `yaml:"Enable"`
	Crontab string `yaml:"Crontab"`
	History int    `yaml:"History"`
	Future  int    `yaml:"Future"`
}

type migrateTime struct {
	VirTool bool `yaml:"VirTool"`
}
type DataService struct {
	Url     string `yaml:"Url"`
	Timeout int    `yaml:"Timeout"`
}

type AlgoService struct {
	Url     string `yaml:"Url"`
	Timeout int    `yaml:"Timeout"`
}
