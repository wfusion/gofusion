package mask

import (
	"strings"
)

const (
	ExampleCHAR    = "ExampleCHAR"
	ExampleTAG     = "ExampleTAG"
	ExampleREPLACE = "ExampleREPLACE"
	ExampleEMPTY   = "ExampleEMPTY"
	ExampleBASE64  = "ExampleBASE64"
	NULL           = "NULL"
	CHINAPHONE     = "CHINAPHONE"
	PHONE          = "PHONE"
	CHINAID        = "CHINAID"
	IDCARD         = "IDCARD"
	Email          = "Email"
	UID            = "UID"
	BANK           = "BANK"
	PASSPORT       = "PASSPORT"
	ADDRESS        = "ADDRESS"
	NAME           = "NAME"
	NUMBER         = "NUMBER"
	MACADDR        = "MACADDR"
	ABA            = "ABA"
	BITCOIN        = "BITCOIN"
	CAR            = "CAR"
	DID            = "DID"
	BIRTH          = "BIRTH"
	AGE            = "AGE"
	EDU            = "EDU"
)

// EngineAPI is a collection of DLP APIs
type EngineAPI interface {
	// ApplyConfig by configuration content
	ApplyConfig(conf string) error

	// ApplyConfigFile by config file path
	ApplyConfigFile(filePath string) error

	// Detect string
	Detect(inputText string) (DetectResultList, error)

	// DetectMap detects KV map
	DetectMap(inputMap map[string]string) (DetectResultList, error)

	// DetectJSON detects json string
	DetectJSON(jsonText string) (DetectResultList, error)

	// DeidentifyJSONByResult  returns masked json object in string format from the passed-in DetectResultList.
	// You may want to call DetectJSON first to obtain the DetectResultList.
	DeidentifyJSONByResult(jsonText string, detectResults DetectResultList) (outStr string, retErr error)

	// Deidentify detects string firstly, then return masked string and results
	Deidentify(inputText string) (string, DetectResultList, error)

	// DeidentifyMap detects KV map firstly,then return masked map
	DeidentifyMap(inputMap map[string]string) (map[string]string, DetectResultList, error)

	// DeidentifyJSON detects JSON firstly, then return masked json object in string formate and results
	DeidentifyJSON(jsonText string) (string, DetectResultList, error)

	// ShowResults print results in console
	ShowResults(resultArray DetectResultList)

	// Mask inputText following predefined method of MaskRules in config
	Mask(inputText string, methodName string) (string, error)

	// MaskStruct will mask a strcut object by tag mask info
	MaskStruct(inObj any) (any, error)

	// NewLogProcessor create a log processer for the package logs
	// After calling this function, eng can only be used for log processing,
	// because the rules will be specially optimized and not suitable for other API use.
	// The maximum input is 1KB, 16 items, and the expected highest QPS is 200. If it exceeds,
	// the log will be truncated, and the CPU will also rise accordingly.
	// Business needs to pay special attention.
	NewLogProcessor() Processor

	// Close engine object, release memory of inner object
	Close()

	// GetVersion Get Dlp SDK version string
	GetVersion() string

	// RegisterMasker Register DIY Masker
	RegisterMasker(maskName string, maskFunc func(string) (string, error)) error

	// DisableAllRules will disable all rules, only used for benchmark baseline
	DisableAllRules() error

	// NewEmptyLogProcessor NewEmptyLogProcesser will new a log processer which will do nothing
	NewEmptyLogProcessor() Processor

	// ShowDlpConf will print config file
	ShowDlpConf() error
}

// DetectResult DataStrcuture. Two kinds of result
// ResultType: VALUE, returned from Detect() and Deidentify()
// ResultType: KV, returned from DetectMap(), DetectJSON() and DeidentifyMap()
type DetectResult struct {
	RuleID     int32  `json:"rule_id"`     // RuleID of rules in conf file
	Text       string `json:"text"`        // substring which is detected by rule
	MaskText   string `json:"mask_text"`   // maskstring which is deidentify by rule
	ResultType string `json:"result_type"` // VALUE or KV, based on rule
	Key        string `json:"key"`         // In ResultType: KV, Key is key of map for path of json object
	// In ResultType: VALUE mode, DetectResult.Text will be inputText[ByteStart:ByteEnd]
	// In ResultType: KV, DetectResult.Text will be inputMap[DetectResult.Key][ByteStart:ByteEnd]
	ByteStart int `json:"byte_start"`
	ByteEnd   int `json:"byte_end"`
	// fields are defined in conf file
	InfoType  string            `json:"info_type"`
	EnName    string            `json:"en_name"`
	CnName    string            `json:"cn_name"`
	GroupName string            `json:"group_name"`
	Level     string            `json:"level"`
	ExtInfo   map[string]string `json:"ext_info,omitempty"`
}

// DetectResultList Result type define is uesd for sort in mergeResults
type DetectResultList []*DetectResult

// Len function is used for sort in mergeResults
func (a DetectResultList) Len() int {
	return len(a)
}

// Swap function is used for sort in mergeResults
func (a DetectResultList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Less function is used for sort in mergeResults
func (a DetectResultList) Less(i, j int) bool {
	if a[i].ByteStart < a[j].ByteStart {
		return true
	} else if a[i].ByteStart == a[j].ByteStart {
		if a[i].ByteEnd < a[j].ByteEnd {
			return true
		} else if a[i].ByteEnd == a[j].ByteEnd { // same
			return a[i].RuleID < a[j].RuleID
		} else {
			return false
		}
	} else {
		return false
	}
}

// Contain checks whether a[i] contains a[j]
func (a DetectResultList) Contain(i, j int) bool {
	return a[i].Key == a[j].Key && a[i].ByteStart <= a[j].ByteStart && a[j].ByteEnd <= a[i].ByteEnd
}

// Equal checks whether positions are equal
func (a DetectResultList) Equal(i, j int) bool {
	return a[i].ByteStart == a[j].ByteStart && a[j].ByteEnd == a[i].ByteEnd && a[i].Key == a[j].Key
}

type Processor func(rawLog string, kvs ...any) (string, []any, bool)

// IsValue checks whether the ResultType is VALUE
func (d *DetectResult) IsValue() bool {
	return strings.Compare(d.ResultType, "VALUE") == 0
}

// IsKV checks whether the ResultType is KV
func (d *DetectResult) IsKV() bool {
	return strings.Compare(d.ResultType, "KV") == 0
}
