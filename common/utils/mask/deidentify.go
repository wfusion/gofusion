package mask

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

// Deidentify detects string firstly, then return masked string and results
func (e *Engine) Deidentify(inputText string) (outputText string, retResults DetectResultList, retErr error) {
	defer e.recoveryImpl()
	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return "", nil, ErrProcessAfterClose
	}
	if e.isOnlyForLog() {
		return inputText, nil, ErrOnlyForLog
	}
	if len(inputText) > defMaxInput {
		return inputText, nil, fmt.Errorf("defMaxInput: %d , %w", defMaxInput, ErrMaxInputLimit)
	}
	outputText, retResults, retErr = e.deidentifyImpl(inputText)
	return
}

// DeidentifyMap detects KV map firstly,then return masked map
func (e *Engine) DeidentifyMap(inputMap map[string]string) (outMap map[string]string, retResults DetectResultList, retErr error) {
	defer e.recoveryImpl()

	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return nil, nil, ErrProcessAfterClose
	}
	if len(inputMap) > defMaxItem {
		return inputMap, nil, fmt.Errorf("defMaxItem: %d , %w", defMaxItem, ErrMaxInputLimit)
	}
	outMap, retResults, retErr = e.deidentifyMapImpl(inputMap)
	return
}

// DeidentifyJSON detects JSON firstly, then return masked json object in string format and results
func (e *Engine) DeidentifyJSON(jsonText string) (outStr string, retResults DetectResultList, retErr error) {
	defer e.recoveryImpl()

	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return jsonText, nil, ErrProcessAfterClose
	}
	outStr = jsonText
	if results, kvMap, err := e.detectJSONImpl(jsonText); err == nil {
		retResults = results
		var jsonObj interface{}
		if err := json.Unmarshal([]byte(jsonText), &jsonObj); err == nil {
			//kvMap := e.resultsToMap(results)
			outObj := e.dfsJSON("", &jsonObj, kvMap, true)
			if outJSON, err := json.Marshal(outObj); err == nil {
				outStr = string(outJSON)
			} else {
				retErr = err
			}
		} else {
			retErr = err
		}
	} else {
		retErr = err
	}
	return
}

// DeidentifyJSONByResult  returns masked json object in string format from the passed-in DetectResultList.
// You may want to call DetectJSON first to obtain the DetectResultList.
func (e *Engine) DeidentifyJSONByResult(jsonText string, detectResults DetectResultList) (outStr string, retErr error) {
	defer e.recoveryImpl()
	// have to use closure to pass retResults parameters
	if !e.hasConfigured() { // not configed
		panic(ErrHasNotConfigured)
	}
	if e.hasClosed() {
		return jsonText, ErrProcessAfterClose
	}
	outStr = jsonText
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(jsonText), &jsonObj); err == nil {
		kvMap := e.resultsToMap(detectResults)
		outObj := e.dfsJSON("", &jsonObj, kvMap, true)
		if outJSON, err := json.Marshal(outObj); err == nil {
			outStr = string(outJSON)
		} else {
			retErr = err
		}
	} else {
		retErr = err
	}

	return
}

// deidentifyImpl implements Deidentify string
func (e *Engine) deidentifyImpl(inputText string) (outputText string, retResults DetectResultList, retErr error) {
	outputText = inputText // default same text
	if arr, err := e.detectImpl(inputText); err == nil {
		retResults = arr
		if out, err := e.deidentifyByResult(inputText, retResults); err == nil {
			outputText = out
		} else {
			retErr = err
		}
	} else {
		retErr = err
	}
	return
}

// deidentifyMapImpl implements DeidentifyMap
func (e *Engine) deidentifyMapImpl(inputMap map[string]string) (outMap map[string]string, retResults DetectResultList, retErr error) {
	outMap = make(map[string]string)
	if results, err := e.detectMapImpl(inputMap); err == nil {
		if len(results) == 0 { // detect nothing
			return inputMap, results, nil
		} else {
			outMap = inputMap
			for _, item := range results {
				if orig, ok := outMap[item.Key]; ok {
					if out, err := e.deidentifyByResult(orig, DetectResultList{item}); err == nil {
						outMap[item.Key] = out
					}
				}
			}
			retResults = results
		}
	} else {
		outMap = inputMap
		retErr = err
	}
	return
}

// deidentifyByResult concatenate MaskText
func (e *Engine) deidentifyByResult(in string, arr DetectResultList) (string, error) {
	out := make([]byte, 0, len(in)+8)
	pos := 0
	inArr := s2b(in)
	for _, res := range arr {
		if pos < res.ByteStart {
			out = append(out, []byte(inArr[pos:res.ByteStart])...)
		}
		out = append(out, []byte(res.MaskText)...)
		pos = res.ByteEnd
	}
	if pos < len(in) {
		out = append(out, []byte(inArr[pos:])...)
	}
	outStr := b2s(out)
	return outStr, nil
}

// resultsToMap convert results array into Map[Key]=MaskText
func (e *Engine) resultsToMap(results DetectResultList) map[string]string {
	kvMap := make(map[string]string)
	for _, item := range results {
		kvMap[item.Key] = item.MaskText
	}
	return kvMap
}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func s2b(s string) (b []byte) {
	/* #nosec G103 */
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	/* #nosec G103 */
	sh := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Len = sh.Len
	bh.Cap = sh.Len
	return b
}
