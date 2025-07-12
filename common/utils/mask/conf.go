package mask

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	defAPIVersionPrefix = "v2"
	defModeSet          = []string{"debug", "release"}
	defMaskTypeSet      = []string{"CHAR", "TAG", "REPLACE", "ALGO"}
	defMaskAlgo         = []string{"BASE64", "MD5", "CRC32", "ADDRESS", "NUMBER", "DEIDENTIFY"}
	defIgnoreKind       = []string{"NUMERIC", "ALPHA_UPPER_CASE", "ALPHA_LOWER_CASE", "WHITESPACE", "PUNCTUATION"}
)

type rule struct {
	RuleName      string   `yaml:"RuleName"`
	MaskType      string   `yaml:"MaskType"` // one of [CHAR, TAG, REPLACE, EMPTY, ALGO ]
	Value         string   `yaml:"Value"`
	Offset        int32    `yaml:"Offset"`
	Padding       int32    `yaml:"Padding"`
	Length        int32    `yaml:"Length"`
	Reverse       bool     `yaml:"Reverse"`
	IgnoreCharSet string   `yaml:"IgnoreCharSet"`
	IgnoreKind    []string `yaml:"IgnoreKind"` // one of [NUMERIC, ALPHA_UPPER_CASE, ALPHA_LOWER_CASE, WHITESPACE, PUNCTUATION]
}

type ruleItem struct {
	RuleID      int32  `yaml:"RuleID"`
	InfoType    string `yaml:"InfoType"`
	Description string `yaml:"Description"`
	EnName      string `yaml:"EnName"`
	CnName      string `yaml:"CnName"`
	Level       string `yaml:"Level"` // L1 (least Sensitive) ~ L4 (Most Sensitive)
	// (KReg || KDict) && (VReg || VDict)
	Detect struct {
		KReg  []string `yaml:"KReg"`       // Regex List for Key
		KDict []string `yaml:"KDict,flow"` // Dict for Key
		VReg  []string `yaml:"VReg"`       // Regex List for Value
		VDict []string `yaml:"VDict,flow"` // Dict for Value
	} `yaml:"Detect"`
	// result which is hit by blacklist will not returned to caller
	Filter struct {
		// BReg || BDict
		BReg  []string `yaml:"BReg"`       // Regex List for BlackList
		BDict []string `yaml:"BDict,flow"` // Dict for BlackList
		BAlgo []string `yaml:"BAlgo"`      // Algorithm List for BlackList, one of [ MASKED ]
	} `yaml:"Filter"`
	// result need pass verify process before retured to caller
	Verify struct {
		// CReg || CDict
		CReg  []string `yaml:"CReg"`       // Regex List for Context Verification
		CDict []string `yaml:"CDict,flow"` // Dict for Context Verification
		VAlgo []string `yaml:"VAlgo"`      // Algorithm List for Verification, one of [ IDVerif , CardVefif ]
	} `yaml:"Verify"`
	Mask    string            `yaml:"Mask"` // rule.RuleName for Mask
	ExtInfo map[string]string `yaml:"ExtInfo"`
}

type conf struct {
	Global struct {
		Date           string  `yaml:"Date"`
		ApiVersion     string  `yaml:"ApiVersion"`
		Mode           string  `yaml:"Mode"`
		AllowRPC       bool    `yaml:"AllowRPC"`
		EnableRules    []int32 `yaml:"EnableRules,flow"`
		DisableRules   []int32 `yaml:"DisableRules,flow"`
		MaxLogInput    int32   `yaml:"MaxLogInput"`
		MaxRegexRuleID int32   `yaml:"MaxRegexRuleID"`
	} `yaml:"Global"`
	MaskRules []rule     `yaml:"MaskRules"`
	Rules     []ruleItem `yaml:"Rules"`
}

// public func

// newDlpConf creates conf object by conf content string
func newDlpConf(confString string) (*conf, error) {
	return newDlpConfImpl(confString)
}

// newDlpConfByPath creates conf object by confPath
func newDlpConfByPath(confPath string) (*conf, error) {
	if len(confPath) == 0 {
		return nil, ErrConfPathEmpty
	}
	if fileData, err := ioutil.ReadFile(confPath); err == nil {
		return newDlpConfImpl(string(fileData))
	} else {
		return nil, err
	}
}

func (d *conf) verify() error {
	// ApiVersion
	if !strings.HasPrefix(d.Global.ApiVersion, defAPIVersionPrefix) {
		return fmt.Errorf("%w, Global.APIVersion:%s failed", ErrConfVerifyFailed, d.Global.ApiVersion)
	}
	// Mode
	d.Global.Mode = strings.ToLower(d.Global.Mode)
	if inList(d.Global.Mode, defModeSet) == -1 { // not found
		return fmt.Errorf("%w, Global.Mode:%s failed", ErrConfVerifyFailed, d.Global.Mode)
	}
	// MaskRules
	for _, rule := range d.MaskRules {
		// MaskType
		if inList(rule.MaskType, defMaskTypeSet) == -1 {
			return fmt.Errorf("%w, Mask RuleName:%s, MaskType:%s is not suppored", ErrConfVerifyFailed, rule.RuleName, rule.MaskType)
		}
		if strings.Compare(rule.MaskType, "ALGO") == 0 {
			if inList(rule.Value, defMaskAlgo) == -1 {
				return fmt.Errorf("%w, Mask RuleName:%s, ALGO Value: %s is not supported", ErrConfVerifyFailed, rule.RuleName, rule.Value)
			}
		}
		if !(rule.Offset >= 0) {
			return fmt.Errorf("%w, Mask RuleName:%s, Offset: %d need >=0", ErrConfVerifyFailed, rule.RuleName, rule.Offset)
		}
		if !(rule.Length >= 0) {
			return fmt.Errorf("%w, Mask RuleName:%s, Length: %d need >=0", ErrConfVerifyFailed, rule.RuleName, rule.Length)
		}
		for _, kind := range rule.IgnoreKind {
			if inList(kind, defIgnoreKind) == -1 {
				return fmt.Errorf("%w, Mask RuleName:%s, IgnoreKind: %s is not supported", ErrConfVerifyFailed, rule.RuleName, kind)
			}
		}
	}
	// Rules
	for _, rule := range d.Rules {
		de := rule.Detect
		// at least one detect rule
		if len(de.KReg) == 0 && len(de.KDict) == 0 && len(de.VReg) == 0 && len(de.VDict) == 0 {
			return fmt.Errorf("%w, RuleID:%d, Detect field missing", ErrConfVerifyFailed, rule.RuleID)
		}
	}
	return nil
}

// newDlpConfImpl implements newDlpConf by receving conf content string
func newDlpConfImpl(confString string) (*conf, error) {
	if len(confString) == 0 {
		return nil, ErrConfEmpty
	}
	confObj := new(conf)
	if err := yaml.Unmarshal([]byte(confString), &confObj); err == nil {
		if err := confObj.verify(); err == nil {
			return confObj, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

// inList finds item in list
func inList(item string, list []string) int {
	for i, v := range list {
		if strings.Compare(item, v) == 0 { // found
			return i
		}
	}
	return -1 // not found
}
