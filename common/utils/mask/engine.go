package mask

import (
	"bufio"
	stdJson "encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
	"gopkg.in/yaml.v3"
)

// const var for default values
const (
	version          = "v1.2.15"
	defMaxInput      = 1024 * 1024                       // 1MB, the max input string length
	defLimitErr      = "<--[mask] Log Limit Exceeded-->" // append to log if limit is exceeded
	defMaxLogItem    = 16                                // max input items for log
	defResultSize    = 4                                 // default results size for array allocation
	defLineBlockSize = 1024                              // default line block
	defCutter        = " /\r\n\\[](){}:=\"',"            // default cutter for finding KV object in string
	defMaxItem       = 1024 * 4                          // max input items for MAP api
	defMaxCallDeep   = 5                                 // max call depth for MaskStruct
)

var (
	defMaxLogInput    int32 = 1024 // default 1KB, the max input lenght for log, change it in conf
	defMaxRegexRuleId int32 = 0    // default 0, no regex rule will be used for log default, change it in conf
)

// Engine Object implements all DLP api functions
type Engine struct {
	Version      string
	callerID     string
	endPoint     string
	accessKey    string
	secretKey    string
	isLegal      bool // true: auth is ok, false: auth failed
	isClosed     bool // true: Close() has been called
	isForLog     bool // true: NewLogProcessor() has been called, will not do other api
	isConfigured bool // true: ApplyConfig* api has been called, false: not been called
	confObj      *conf
	detectorMap  map[int32]detectorAPI
	maskerMap    map[string]api
}

func NewEngine(callerID string) (EngineAPI, error) {
	defer recoveryImplStatic()
	eng := new(Engine)
	eng.Version = version
	eng.callerID = callerID
	eng.detectorMap = make(map[int32]detectorAPI)
	eng.maskerMap = make(map[string]api)

	return eng, nil
}

// Close release inner object, such as detector and masker
func (e *Engine) Close() {
	defer e.recoveryImpl()
	for k, v := range e.detectorMap {
		if v != nil {
			v.Close()
			e.detectorMap[k] = nil
		}
	}
	for k, v := range e.maskerMap {
		if v != nil {
			e.maskerMap[k] = nil
		}
	}
	e.detectorMap = nil
	e.confObj = nil
	e.isClosed = true
}

// ShowResults print results in console
func (e *Engine) ShowResults(results DetectResultList) {
	defer e.recoveryImpl()
	fmt.Println()
	fmt.Printf("\tTotal Results: %d\n", len(results))
	for i, item := range results {
		fmt.Printf("Result[%d]: %+v\n", i, *item)
	}
	fmt.Println()
}

// GetVersion return DLP SDK version
func (e *Engine) GetVersion() string {
	defer e.recoveryImpl()
	return version
}

// NewLogProcessor create a log processer for the package logs
// Once invoked, Engine can only be used for log processing,
// because the rules are specifically optimized for use by other apis
func (e *Engine) NewLogProcessor() Processor {
	defer e.recoveryImpl()

	e.isForLog = true
	e.selectRulesForLog()
	return func(rawLog string, kvs ...interface{}) (string, []interface{}, bool) {
		// do not call log api in this func
		defer e.recoveryImpl()
		// do not call report at here, because this func will call Deidentify()
		//Do not use logs function inside this function
		newLog := rawLog
		logCutted := false
		if int32(len(newLog)) >= defMaxLogInput {
			// cut for long log
			newLog = newLog[:defMaxLogInput]
			logCutted = true
		}
		newLog, _, _ = e.deidentifyImpl(newLog)
		if logCutted {
			newLog += defLimitErr
		}
		//fmt.Printf("LogProcesser rawLog: %s, kvs: %+v\n", rawLog, kvs)
		sz := len(kvs)
		//k1,v1,k2,v2,...
		if sz%2 != 0 {
			sz--
		}
		kvCutted := false
		if sz >= defMaxLogItem {
			// cut for too many items
			sz = defMaxLogItem
			kvCutted = true
		}
		retKvs := make([]interface{}, 0, sz)
		if sz > 0 {
			inMap := make(map[string]string)
			for i := 0; i < sz; i += 2 {
				keyStr := e.interfaceToStr(kvs[i])
				valStr := e.interfaceToStr(kvs[i+1])
				inMap[keyStr] = valStr
			}
			outMap, _, _ := e.deidentifyMapImpl(inMap)
			for k, v := range outMap {
				v, _, _ = e.deidentifyImpl(v)
				retKvs = append(retKvs, k, v)
			}
		}
		if kvCutted {
			retKvs = append(retKvs, "<--[mask error]-->", defLimitErr)
		}
		return newLog, retKvs, true
	}
}

// NewEmptyLogProcessor will new a log processer which will do nothing
func (e *Engine) NewEmptyLogProcessor() Processor {
	return func(rawLog string, kvs ...interface{}) (string, []interface{}, bool) {
		return rawLog, kvs, true
	}
}

// ShowDlpConf print conf on console
func (e *Engine) ShowDlpConf() error {
	// copy obj
	confObj := *e.confObj
	out, err := yaml.Marshal(confObj)
	if err == nil {
		fmt.Println("====ngdlp conf start====")
		fmt.Println(string(out))
		fmt.Println("====ngdlp conf end====")
		return nil
	} else {
		return err
	}
}

// DisableAllRules will use embeded local config, only used for DLP team
func (e *Engine) DisableAllRules() error {
	for i, _ := range e.detectorMap {
		e.detectorMap[i] = nil
	}
	return nil
}

// Mask will return masked text directly based on methodName
func (e *Engine) Mask(inputText string, methodName string) (outputText string, err error) {
	defer e.recoveryImpl()
	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return "", ErrProcessAfterClose
	}
	if len(inputText) > defMaxInput {
		return inputText, fmt.Errorf("defMaxInput: %d , %w", defMaxInput, ErrMaxInputLimit)
	}
	if maskWorker, ok := e.maskerMap[methodName]; ok {
		return maskWorker.Mask(inputText)
	} else {
		return inputText, fmt.Errorf("methodName: %s, error: %w", methodName, ErrMaskWorkerNotfound)
	}
}

// MaskStruct will mask a struct object by tag mask info
func (e *Engine) MaskStruct(in any) (out any, err error) {
	out = in                  // fail back to in
	err = ErrMaskStructOutput // default return err if panic
	defer e.recoveryImpl()
	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return in, ErrProcessAfterClose
	}
	if in == nil {
		return nil, ErrMaskStructInput
	}
	out, err = e.maskStructImpl(in, defMaxCallDeep)
	return
}

// RegisterMasker Register DIY Masker
func (e *Engine) RegisterMasker(maskName string, maskFunc func(string) (string, error)) error {
	defer e.recoveryImpl()
	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return ErrProcessAfterClose
	}
	if _, ok := e.maskerMap[maskName]; ok {
		return ErrMaskNameConflict
	} else {
		if worker, err := e.NewDIYMaskWorker(maskName, maskFunc); err == nil {
			e.maskerMap[maskName] = worker
			return nil
		} else {
			return err
		}
	}
}

// ApplyConfig by configuration content
func (e *Engine) ApplyConfig(confString string) error {
	defer e.recoveryImpl()
	if confObj, err := newDlpConf(confString); err == nil {
		return e.applyConfigImpl(confObj)
	} else {
		return err
	}
}

// ApplyConfigFile by config file path
func (e *Engine) ApplyConfigFile(filePath string) error {
	defer e.recoveryImpl()
	var retErr error
	if confObj, err := newDlpConfByPath(filePath); err == nil {
		retErr = e.applyConfigImpl(confObj)
	} else {
		retErr = err
	}
	return retErr
}

// Detect find sensitive information for input string
func (e *Engine) Detect(inputText string) (retResults DetectResultList, retErr error) {
	defer e.recoveryImpl()

	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return nil, ErrProcessAfterClose
	}
	if len(inputText) > defMaxInput {
		return nil, fmt.Errorf("defMaxInput: %d , %w", defMaxInput, ErrMaxInputLimit)
	}
	retResults, retErr = e.detectImpl(inputText)
	return
}

// DetectMap detects KV map
func (e *Engine) DetectMap(inputMap map[string]string) (retResults DetectResultList, retErr error) {
	defer e.recoveryImpl()

	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return nil, ErrProcessAfterClose
	}
	if len(inputMap) > defMaxItem {
		return nil, fmt.Errorf("defMaxItem: %d , %w", defMaxItem, ErrMaxInputLimit)
	}
	inMap := make(map[string]string)
	for k, v := range inputMap {
		loK := strings.ToLower(k)
		inMap[loK] = v
	}
	retResults, retErr = e.detectMapImpl(inMap)
	return
}

// DetectJSON detects json string
func (e *Engine) DetectJSON(jsonText string) (retResults DetectResultList, retErr error) {
	defer e.recoveryImpl()

	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return nil, ErrProcessAfterClose
	}
	retResults, _, retErr = e.detectJSONImpl(jsonText)
	return
}

// applyConfigImpl sets confObj into Engine, then postLoadConfig(), such as load detector and worker
func (e *Engine) applyConfigImpl(confObj *conf) error {
	e.confObj = confObj
	return e.postLoadConfig()
}

// interfaceToStr converts interface to string
func (e *Engine) interfaceToStr(in interface{}) string {
	out := ""
	switch in.(type) {
	case []byte:
		out = string(in.([]byte))
	case string:
		out = in.(string)
	default:
		out = fmt.Sprint(in)
	}
	return out
}

// formatEndPoint formats endpoint
func (e *Engine) formatEndPoint(endpoint string) string {
	out := endpoint
	if !strings.HasPrefix(endpoint, "http") { //not( http or https)
		out = "http://" + endpoint // defualt use http
		out = strings.TrimSuffix(out, "/")
	}
	return out
}

// detectImpl works for the Detect api
func (e *Engine) detectImpl(inputText string) (DetectResultList, error) {
	rd := bufio.NewReaderSize(strings.NewReader(inputText), defLineBlockSize)
	currPos := 0
	results := make(DetectResultList, 0, defResultSize)
	for {
		line, err := rd.ReadBytes('\n')
		if len(line) > 0 {
			line := e.detectPre(line)
			lineResults := e.detectProcess(line)
			postResutls := e.detectPost(lineResults, currPos)
			results = append(results, postResutls...)
			currPos += len(line)
		}
		if err != nil {
			if err != io.EOF {
				//show err
			}
			break
		}
	}
	return results, nil
}

// detectPre calls prepare func before detect
func (e *Engine) detectPre(line []byte) []byte {
	line = e.unquoteEscapeChar(line)
	line = e.replaceWideChar(line)
	return line
}

// detectProcess detects sensitive info for a line
func (e *Engine) detectProcess(line []byte) DetectResultList {
	// detect from a byte array
	bytesResults, _ := e.detectBytes(line)
	// detect from a kvList which is extracted from the byte array
	// kvList is used for the two item with same key
	kvList := e.extractKVList(line)
	kvResults, _ := e.detectKVList(kvList)
	results := e.mergeResults(bytesResults, kvResults)
	return results
}

// detectBytes detects for a line
func (e *Engine) detectBytes(line []byte) (DetectResultList, error) {
	results := make(DetectResultList, 0, defResultSize)
	var retErr error
	//start := time.Now()
	for _, obj := range e.detectorMap {
		if obj != nil && obj.IsValue() {
			if e.isOnlyForLog() { // used in log processor mod, need very efficient
				if obj.GetRuleID() > defMaxRegexRuleId && obj.UseRegex() { // if ID>MAX and rule uses regex
					continue // will not use this rule in log processor mod
				}
			}
			res, err := obj.DetectBytes(line)
			if err != nil {
				retErr = err
			}
			results = append(results, res...)
		}
	}
	//fmt.Printf("check rule:%d, len:%d, cast:%v\n", len(e.detectorMap), len(line), time.Since(start))

	// the last error will be returned
	return results, retErr
}

// extractKVList extracts KV item into a returned list
func (e *Engine) extractKVList(line []byte) []*kvItem {
	kvList := make([]*kvItem, 0, defResultSize)

	sz := len(line)
	for i := 0; i < sz; {
		// k:v k=v k:=v k==v, chinese big "："
		ch, width := utf8.DecodeRune(line[i:])
		if width == 0 { // error
			break
		}
		if i+1 < sz && isEqualChar(ch) {
			left := ""
			right := ""
			vPos := []int{-1, -1}
			kPos := []int{-1, -1}
			isFound := false
			if i+2 < sz {
				nx, nxWidth := utf8.DecodeRune(line[i+width:])
				if nx == '=' {
					left, kPos = lastToken(line, i)
					right, vPos = firstToken(line, i+width+nxWidth)
					isFound = true
				}
			}
			if !isFound {
				left, kPos = lastToken(line, i)
				right, vPos = firstToken(line, i+width)
				isFound = true
			}
			//log.Debugf("%s [%d,%d) = %s [%d,%d)", left, kPos[0], kPos[1], right, vPos[0], vPos[1])
			_ = kPos
			if len(left) != 0 && len(right) != 0 {
				loLeft := strings.ToLower(left)
				kvList = append(kvList, &kvItem{
					Key:   loLeft,
					Value: right,
					Start: vPos[0],
					End:   vPos[1],
				})
			}
		}
		i += width
	}
	return kvList
}

// detectKVList accepts kvList to do detection
func (e *Engine) detectKVList(kvList []*kvItem) (DetectResultList, error) {
	results := make(DetectResultList, 0, defResultSize)

	for _, obj := range e.detectorMap {
		if obj != nil && obj.IsKV() {
			if e.isOnlyForLog() { // used in log processor mod, need very efficient
				if obj.GetRuleID() > defMaxRegexRuleId && obj.UseRegex() { // if ID>MAX and rule uses regex
					continue // will not use this rule in log processor mod
				}
			}
			// can not call e.DetectMap, because it will call mask, but position info has not been provided
			mapResults, _ := obj.DetectList(kvList)
			for i, _ := range mapResults {
				// detectKVList is called from detect(), so result type will be VALUE
				mapResults[i].ResultType = resultTypeValue
			}
			results = append(results, mapResults...)
		}
	}
	return results, nil
}

// detectPost calls post func after detect
func (e *Engine) detectPost(results DetectResultList, currPos int) DetectResultList {
	ret := e.ajustResultPos(results, currPos)
	ret = e.maskResults(ret)
	return ret
}

// merge and sort two detect results
func (e *Engine) mergeResults(a DetectResultList, b DetectResultList) DetectResultList {
	var total DetectResultList
	if len(a) == 0 {
		total = b
	} else {
		if len(b) == 0 {
			total = a
		} else { // len(a)!=0 && len(b)!=0
			total = make(DetectResultList, 0, len(a)+len(b))
			total = append(total, a...)
			total = append(total, b...)
		}
	}
	if len(total) == 0 { // nothing
		return total
	}
	// sort
	sort.Sort(DetectResultList(total))
	sz := len(total)
	mark := make([]bool, sz)
	// firstly, all elements will be left
	for i := 0; i < sz; i++ {
		mark[i] = true
	}
	for i := 0; i < sz; i++ {
		if mark[i] {
			for j := i + 1; j < sz; j++ {
				if mark[j] {
					// inner element will be ignored
					if DetectResultList(total).Equal(i, j) {
						mark[i] = false
						break
					} else {
						if DetectResultList(total).Contain(i, j) {
							mark[j] = false
						}
						if DetectResultList(total).Contain(j, i) {
							mark[i] = false
						}
					}
				}
			}
		}
	}
	ret := make(DetectResultList, 0, sz)
	for i := 0; i < sz; i++ {
		if mark[i] {
			ret = append(ret, total[i])
		}
	}
	return ret
}

// ajustResultPos ajust position offset
func (e *Engine) ajustResultPos(results DetectResultList, currPos int) DetectResultList {
	if currPos > 0 {
		for i := range results {
			results[i].ByteStart += currPos
			results[i].ByteEnd += currPos
		}
	}
	return results
}

// maskResults fill result.MaskText by calling mask.MaskResult()
func (e *Engine) maskResults(results DetectResultList) DetectResultList {
	for _, res := range results {
		if detector, ok := e.detectorMap[res.RuleID]; ok {
			maskRuleName := detector.GetMaskRuleName()
			if maskWorker, ok := e.maskerMap[maskRuleName]; ok {
				maskWorker.MaskResult(res)
			} else { // Not Found
				//log.Errorf(fmt.Errorf("MaskRuleName: %s, Error: %w", maskRuleName, ErrMaskRuleNotfound).Error())
				res.MaskText = res.Text
			}
		}
	}
	return results
}

// detectMapImpl detect sensitive info for inputMap
func (e *Engine) detectMapImpl(inputMap map[string]string) (DetectResultList, error) {
	results := make(DetectResultList, 0, defResultSize)
	for _, obj := range e.detectorMap {
		if obj != nil {
			res, err := obj.DetectMap(inputMap)
			if err != nil {
				//log.Errorf(err.Error())
			}
			results = append(results, res...)
		}
	}
	// merge result to reduce combined item
	results = e.mergeResults(results, nil)
	results = e.maskResults(results)

	return results, nil
}

// detectJSONImpl implements detectJSON
func (e *Engine) detectJSONImpl(jsonText string) (retResults DetectResultList, kvMap map[string]string, retErr error) {
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(jsonText), &jsonObj); err == nil {
		//fmt.Printf("%+v\n", jsonObj)
		kvMap = make(map[string]string, 0)
		e.dfsJSON("", &jsonObj, kvMap, false)
		retResults, retErr = e.detectMapImpl(kvMap)
		for _, item := range retResults {
			if orig, ok := kvMap[item.Key]; ok {
				if out, err := e.deidentifyByResult(orig, DetectResultList{item}); err == nil {
					kvMap[item.Key] = out
				}
			}
		}
		return
	} else {
		if e, ok := err.(*stdJson.SyntaxError); ok {
			return nil, nil, fmt.Errorf("%s: offset[%d], str[%s]", err.Error(), e.Offset,
				jsonText[utils.Max(int(e.Offset)-4, 0):utils.Min(int(e.Offset+10), len(jsonText))])
		}
		return nil, nil, err
	}
}

// replaceWideChar replace wide char with one byte char
func (e *Engine) replaceWideChar(lineArray []byte) []byte {
	sz := len(lineArray)
	for i := 0; i < sz; {
		if (lineArray[i] & 0x80) != 0x80 { //ascii char
			i++
			continue
		}
		r, width := utf8.DecodeRune(lineArray[i:])
		if width == 0 { //error
			break
		}
		switch r {
		case '【':
			copy(lineArray[i:i+width], "  [")
		case '】':
			copy(lineArray[i:i+width], "]  ")
		case '：':
			copy(lineArray[i:i+width], "  :") // must use [space,space,:], for :=
		case '「':
			copy(lineArray[i:i+width], "  {")
		case '」':
			copy(lineArray[i:i+width], "}  ")
		case '（':
			copy(lineArray[i:i+width], "  (")
		case '）':
			copy(lineArray[i:i+width], ")  ")
		case '《':
			copy(lineArray[i:i+width], "  <")
		case '》':
			copy(lineArray[i:i+width], ">  ")
		case '。':
			copy(lineArray[i:i+width], ".  ")
		case '？':
			copy(lineArray[i:i+width], "?  ")
		case '！':
			copy(lineArray[i:i+width], "!  ")
		case '，':
			copy(lineArray[i:i+width], ",  ")
		case '、':
			copy(lineArray[i:i+width], ",  ")
		case '；':
			copy(lineArray[i:i+width], ";  ")

		}
		i += width
	}
	return lineArray
}

// unquoteEscapeChar replace escaped char with orignal char
func (e *Engine) unquoteEscapeChar(lineArray []byte) []byte {
	sz := len(lineArray)
	for i := 0; i < sz; {
		r := lineArray[i]
		if r == '\\' {
			// last 2 char
			if i+1 < sz {
				c := lineArray[i+1]
				value := byte(' ')
				switch c {
				case 'a':
					value = '\a'
				case 'b':
					value = '\b'
				case 'f':
					value = '\f'
				case 'n':
					value = '\n'
				case 'r':
					value = '\r'
				case 't':
					value = '\t'
				case 'v':
					value = '\v'
				case '\\':
					value = '\\'
				case '"':
					value = '"'
				case '\'':
					value = '\''
				}
				lineArray[i] = byte(' ') // space ch
				lineArray[i+1] = value
				i += 2
			} else {
				i++
			}
		} else {
			i++
		}
	}
	return lineArray
}

// isEqualChar checks whether the r is = or : or :=
func isEqualChar(r rune) bool {
	return r == ':' || r == '=' || r == '：'
}

// firstToken extract the first token from bytes, returns token and position info
func firstToken(line []byte, offset int) (string, []int) {
	sz := len(line)
	if offset >= 0 && offset < sz {
		st := offset
		ed := sz
		// find first non cutter
		for i := offset; i < sz; i++ {
			if strings.IndexByte(defCutter, line[i]) == -1 {
				st = i
				break
			}
		}
		// find first cutter
		for i := st + 1; i < sz; i++ {
			if strings.IndexByte(defCutter, line[i]) != -1 {
				ed = i
				break
			}
		}
		return string(line[st:ed]), []int{st, ed}
	} else { // out of bound
		return "", nil
	}
}

// lastToken extract the last token from bytes, returns token and position info
func lastToken(line []byte, offset int) (string, []int) {
	sz := len(line)
	if offset >= 0 && offset < sz {
		st := 0
		ed := offset
		// find first non cutter
		for i := offset - 1; i >= 0; i-- {
			if strings.IndexByte(defCutter, line[i]) == -1 {
				ed = i + 1
				break
			}
		}
		// find first cutter
		for i := ed - 1; i >= 0; i-- {
			if strings.IndexByte(defCutter, line[i]) != -1 {
				st = i + 1
				break
			}
		}
		return string(line[st:ed]), []int{st, ed}
	} else {
		return "", nil
	}
}
