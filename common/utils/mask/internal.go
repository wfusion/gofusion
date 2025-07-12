package mask

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

type httpResponseBase struct {
	RetCode int    `json:"ret_code"`
	RetMsg  string `json:"ret_msg"`
}

type describeRulesResponse struct {
	httpResponseBase
	Rule []byte `json:"rule,omitempty"`
	Crc  uint32 `json:"crc,omitempty"` // rule crc
}

// recoveryImplStatic implements recover if panic which is used for NewEngine api
func recoveryImplStatic() {
	if r := recover(); r != nil {
		if isCriticalPanic(r.(error)) {
			panic(r)
		} else {
			fmt.Fprintf(os.Stderr, "%s, msg: %+v\n", ErrPanic.Error(), r)
			debug.PrintStack()
		}
	}
}

// recoveryImpl implements recover if panic
func (e *Engine) recoveryImpl() {
	if r := recover(); r != nil {
		if isCriticalPanic(r.(error)) {
			panic(r)
		} else {
			fmt.Fprintf(os.Stderr, "%s, msg: %+v\n", ErrPanic.Error(), r)
			debug.PrintStack()
		}
	}
}

// isCriticalPanic checks whether error is critical error
func isCriticalPanic(r error) bool {
	isCritical := false
	switch r {
	case ErrHasNotConfigured:
		isCritical = true
	default:
		isCritical = false
	}
	return isCritical
}

// hasClosed check whether the engine has been closed
func (e *Engine) hasClosed() bool {
	return e.isClosed
}

func (e *Engine) isOnlyForLog() bool {
	return e.isForLog
}

// hasConfigured check whether the engine has been configed
func (e *Engine) hasConfigured() bool {
	return e.isConfigured
}

// postLoadConfig will load config object
func (e *Engine) postLoadConfig() error {
	if e.confObj.Global.MaxLogInput > 0 {
		defMaxLogInput = e.confObj.Global.MaxLogInput
	}
	if e.confObj.Global.MaxRegexRuleID > 0 {
		defMaxRegexRuleId = e.confObj.Global.MaxRegexRuleID
	}
	if err := e.loadDetector(); err != nil {
		return err
	}
	if err := e.loadMaskWorker(); err != nil {
		return err
	}
	e.isConfigured = true
	return nil
}

// loadDetector loads detectors from config
func (e *Engine) loadDetector() (err error) {
	// fill detectorMap
	if err = e.fillDetectorMap(); err != nil {
		return
	}
	// disable rules
	return e.disableRulesImpl(e.confObj.Global.DisableRules)
}

// loadMaskWorker loads maskworker from config
func (e *Engine) loadMaskWorker() error {
	maskRuleList := e.confObj.MaskRules
	if e.maskerMap == nil {
		e.maskerMap = make(map[string]api)
	}
	for _, rule := range maskRuleList {
		if obj, err := newMaskWorker(rule, e); err == nil {
			ruleName := obj.GetRuleName()
			if _, ok := e.maskerMap[ruleName]; !ok {
				e.maskerMap[ruleName] = obj
			}
		}
	}
	return nil
}

// dfsJSON walk a json object, used for DetectJSON and DeidentifyJSON
// in DetectJSON(), isDeidentify is false, kvMap is write only, will store json object path and value
// in DeidentifyJSON(), isDeidentify is true, kvMap is read only, will store path and MaskText of sensitive information
func (e *Engine) dfsJSON(path string, ptr *interface{}, kvMap map[string]string, isDeidentify bool) interface{} {
	path = strings.ToLower(path)
	switch (*ptr).(type) {
	case map[string]interface{}:
		for k, v := range (*ptr).(map[string]interface{}) {
			subpath := path + "/" + k
			(*ptr).(map[string]interface{})[k] = e.dfsJSON(subpath, &v, kvMap, isDeidentify)
		}
	case []interface{}:
		for i, v := range (*ptr).([]interface{}) {
			subpath := ""
			if len(path) == 0 {
				subpath = fmt.Sprintf("/[%d]", i)
			} else {
				subpath = fmt.Sprintf("%s[%d]", path, i)
			}
			(*ptr).([]interface{})[i] = e.dfsJSON(subpath, &v, kvMap, isDeidentify)
		}
	case string:
		var subObj interface{}
		if val, ok := (*ptr).(string); ok {
			// try nested json Unmarshal
			if e.maybeJSON(val) {
				if err := json.Unmarshal([]byte(val), &subObj); err == nil {
					obj := e.dfsJSON(path, &subObj, kvMap, isDeidentify)
					if ret, err := json.Marshal(obj); err == nil {
						retStr := string(ret)
						return retStr
					} else {
						return obj
					}
				}
			} else { // plain text
				if isDeidentify {
					if mask, ok := kvMap[path]; ok {
						return mask
					} else {
						return val
					}
				} else {
					kvMap[path] = val
					return val
				}
			}
		}
	}
	return *ptr
}

// maybeJSON check whether input string is a JSON object or array
func (e *Engine) maybeJSON(in string) bool {
	maybeObj := strings.IndexByte(in, '{') != -1 && strings.LastIndexByte(in, '}') != -1
	maybeArray := strings.IndexByte(in, '[') != -1 && strings.LastIndexByte(in, ']') != -1
	return maybeObj || maybeArray
}

// selectRulesForLog will select rules for log
func (e *Engine) selectRulesForLog() error {
	return nil
}

func (e *Engine) fillDetectorMap() error {
	ruleList := e.confObj.Rules
	if e.detectorMap == nil {
		e.detectorMap = make(map[int32]detectorAPI)
	}
	enableRules := e.confObj.Global.EnableRules
	fullSet := map[int32]bool{}
	for _, rule := range ruleList {
		if obj, err := newDetector(rule); err == nil {
			ruleID := obj.GetRuleID()
			e.detectorMap[ruleID] = obj
			fullSet[ruleID] = false
		} else {
			return err
		}
	}
	// if EnableRules is empty, all rules are loaded
	// else only some rules are enabled.
	if len(enableRules) > 0 {
		for _, ruleID := range enableRules {
			if _, ok := e.detectorMap[ruleID]; ok {
				fullSet[ruleID] = true
			}
		}
		for k, v := range fullSet {
			if !v {
				e.detectorMap[k] = nil
			}
		}
	}
	return nil
}

// disableRules will disable rules based on ruleList, pass them all
func (e *Engine) applyDisableRules(ruleList []int32) {
	e.confObj.Global.DisableRules = ruleList
	e.loadDetector()
}

func (e *Engine) disableRulesImpl(ruleList []int32) error {
	for _, ruleID := range ruleList {
		if _, ok := e.detectorMap[ruleID]; ok {
			e.detectorMap[ruleID] = nil
		}
	}
	total := 0
	for k, rule := range e.detectorMap {
		if rule != nil {
			total++
		} else {
			delete(e.detectorMap, k)
		}
	}
	return nil
}
