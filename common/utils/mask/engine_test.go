package mask

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"
)

type RuleTestItem struct {
	RuleID int32  `yaml:"RuleID"`
	In     string `yaml:"In"`
	Out    string `yaml:"Out"`
}
type RuleTest struct {
	Date     string         `yaml:"Date"`
	TestList []RuleTestItem `yaml:"TestList"`
}

// public func
func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}
func TestRule(t *testing.T) {
	testPath := "./rule_test.yml"
	if buf, err := ioutil.ReadFile(testPath); err == nil {
		ruleTestPtr := new(RuleTest)
		if err := yaml.Unmarshal(buf, ruleTestPtr); err == nil {
			t.Logf("%s: Data:%s", testPath, ruleTestPtr.Date)
			if eng, err := NewEngine("replace.your.psm"); err == nil {
				eng.ApplyConfigFile("./conf.yml")
				for _, item := range ruleTestPtr.TestList {
					if out, results, err := eng.Deidentify(item.In); err == nil {
						if len(results) == 0 && item.RuleID == 0 { // no sensitive info found, it's ok
							// check ok
							continue
						}
						if out == item.Out && len(results) >= 1 && results[0].RuleID == item.RuleID { // check ok
							// check ok
							continue
						} else {
							resultId := int32(-1)
							if len(results) >= 1 {
								resultId = results[0].RuleID
							}
							t.Errorf("Error RuleId: %d, in: %s, out: %s, Deidentify: %s, Results RuleId: %d", item.RuleID, item.In, item.Out, out, resultId)
							eng.ShowResults(results)
						}
					} else {
						t.Error(err.Error())
					}

				}
				t.Logf("Total %d rule Test Case pass", len(ruleTestPtr.TestList))
			} else {
				t.Error(err)
			}

		} else {
			t.Error(err)
		}
	} else {
		t.Error(err)
	}
}

func TestDeidentifyJSONByResult(t *testing.T) {
	jsonBody := `
				{
					"name": "abcdefg",
					"uid": "1234567890"
				}
				`
	eng, err := NewEngine("replace.your.psm")
	if err != nil {
		t.Error(err)
	}

	err = eng.ApplyConfigFile("./conf.yml")
	if err != nil {
		t.Error(err)
	}

	// detectRes contains NAME and UID
	detectRes, err := eng.DetectJSON(jsonBody)
	if err != nil {
		t.Error(err)
	}

	// deidentify the original text
	out, err := eng.DeidentifyJSONByResult(jsonBody, detectRes)
	if err != nil {
		t.Error(err)
	}

	if out != "{\"name\":\"abc****\",\"uid\":\"1*********\"}" {
		t.Error("incorrect output")
	}

	// remove the rule NAME from the detectResults
	newDetectRes := make(DetectResultList, 0, len(detectRes))
	for _, r := range detectRes {
		if r.InfoType != "NAME" {
			newDetectRes = append(newDetectRes, r)
		}

		detectRes = newDetectRes
	}

	// apply the new rule on the original text
	out, err = eng.DeidentifyJSONByResult(jsonBody, detectRes)
	if err != nil {
		t.Error(err)
	}

	// the removed rule should be ignored
	if out != "{\"name\":\"abcdefg\",\"uid\":\"1*********\"}" {
		t.Error("incorrect output")
	}

	// apply the rule UID on a JSON text which doesn't have an UID
	jsonBody = "{\"name\":\"abcdefg\"}"
	out, err = eng.DeidentifyJSONByResult(jsonBody, detectRes)
	if err != nil {
		t.Error(err)
	}

	if out != jsonBody {
		t.Error("incorrect output")
	}
}

// private func

func setup() {
	runtime.GOMAXPROCS(1)
}

func shutdown() {

}
