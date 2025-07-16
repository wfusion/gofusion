package cases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test/config"

	fusCfg "github.com/wfusion/gofusion/config"
)

func TestRemote(t *testing.T) {
	testingSuite := &Remote{Test: new(config.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Remote struct {
	*config.Test
}

func (t *Remote) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)

	})
}

func (t *Remote) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Remote) TestApollo() {
	t.Catch(func() {
		// Given
		files := []string{
			"app.required.local.yml",
			"app.required.yml",
		}
		defer t.RawCopy(files, 1)()
		defer t.mockApolloData()()
		appSetting := new(appConf)
		defer fusCfg.New(t.AppName()).Init(&appSetting, fusCfg.Files(t.ConfigFiles()))()

		// When
		allConfigs := fusCfg.Use(t.AppName()).GetAllConfigs()

		// Then
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fusCfg.Use(t.AppName()).AppName())
		log.Info(context.Background(), "get debug: %+v", fusCfg.Use(t.AppName()).Debug())
	})
}

func (t *Remote) mockApolloData() (cl func()) {
	apolloAddr := "http://apollo_portal:8070"
	jsonFilename := fmt.Sprintf("%s.app.json", t.AppName())
	yamlFilename := "app.required.yml"
	switch {
	case env.GetEnv() == env.Dev:
		apolloAddr = "http://localhost:38070"
		yamlFilename = "app.required.local.yml"
	}
	yamlData, err := os.ReadFile(filepath.Join(env.WorkDir, "configs", yamlFilename))
	t.Require().NoError(err)
	jsonData, err := os.ReadFile(filepath.Join(env.WorkDir, "configs", jsonFilename))
	t.Require().NoError(err)

	cli := newApolloAdminClient(apolloAddr)
	cl = func() {
		for k := range apolloProperties {
			_ = cli.DeleteItem("application", k)
		}
		_ = cli.PublishRelease("application")

		for _, ns := range [...]string{apolloYamlNamespace, apolloJsonNamespace, apolloTxtNamespace} {
			_ = cli.DeleteItem(ns, "content")
			_ = cli.PublishRelease(ns)
		}
	}

	for k, v := range apolloProperties {
		t.Require().NoError(cli.UpsertItem("application", k, v))
	}
	t.Require().NoError(cli.PublishRelease("application"))

	t.Require().NoError(cli.CreateNamespace(apolloYamlNamespace, "yaml", false, "Test YAML Namespace"))
	t.Require().NoError(cli.UpsertItem(apolloYamlNamespace, "content", string(yamlData)))
	t.Require().NoError(cli.PublishRelease(apolloYamlNamespace))

	t.Require().NoError(cli.CreateNamespace(apolloJsonNamespace, "json", false, "Test JSON Namespace"))
	t.Require().NoError(cli.UpsertItem(apolloJsonNamespace, "content", string(jsonData)))
	t.Require().NoError(cli.PublishRelease(apolloJsonNamespace))

	t.Require().NoError(cli.CreateNamespace(apolloTxtNamespace, "txt", false, "Test TXT Namespace"))
	t.Require().NoError(cli.UpsertItem(apolloTxtNamespace, "content", apolloTxtData))
	t.Require().NoError(cli.PublishRelease(apolloTxtNamespace))

	return
}
