package mask

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"reflect"
	"strings"
	"unicode/utf8"
)

const (
	// Char Replacing sensitive information with characters
	// requires more detailed configuration items later.
	Char = "CHAR"
	// Tag Replace the sensitive information with the InfoType
	// in the recognition and processing rules in the form '< InfoType > ' .
	Tag = "TAG"
	// Replace The string defined with Value, replacing sensitive information,
	// can be set to an empty string for direct erasure.
	Replace = "REPLACE"

	// Algo Algorithmic functions defined with Value,
	// dealing with sensitive information,
	// replacing text with algorithmic return values.
	// Currently supported algorithms are [ Base64, MD5, CRC32]
	Algo = "ALGO"

	AlgoBase64 = "BASE64"
	AlgoMd5    = "MD5"
	AlgoCrc32  = "CRC32"

	UnknownTag = "UNKNOWN"
)

type worker struct {
	rule   rule
	parent EngineAPI
}

type api interface {
	// GetRuleName return RuleName of a worker
	GetRuleName() string
	// Mask will return masked string
	Mask(string) (string, error)
	// MaskResult will modify DetectResult.MaskText
	MaskResult(*DetectResult) error
}

// public func

// newMaskWorker create worker based on MaskRule
func newMaskWorker(rule rule, p EngineAPI) (api, error) {
	obj := new(worker)
	//IgnoreKind
	for _, kind := range rule.IgnoreKind {
		switch kind {
		case "NUMERIC":
			rule.IgnoreCharSet += "0123456789"
		case "ALPHA_LOWER_CASE":
			rule.IgnoreCharSet += "abcdefghijklmnopqrstuvwxyz"
		case "ALPHA_UPPER_CASE":
			rule.IgnoreCharSet += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		case "WHITESPACE":
			rule.IgnoreCharSet += " \t\n\x0B\f\r"
		case "PUNCTUATION":
			rule.IgnoreCharSet += "!\"#$%&'()*+,-./:;<=>?@[]^_`{|}~"
		}
	}
	obj.rule = rule
	obj.parent = p
	return obj, nil
}

// GetRuleName return RuleName of a worker
func (w *worker) GetRuleName() string {
	return w.rule.RuleName
}

// MaskResult will modify DetectResult.MaskText
func (w *worker) MaskResult(res *DetectResult) error {
	var err error
	if strings.Compare(w.rule.MaskType, Tag) == 0 {
		res.MaskText, err = w.maskTagImpl(res.Text, res.InfoType)
	} else {
		res.MaskText, err = w.Mask(res.Text)
	}
	return err
}

// Mask will return masked string
func (w *worker) Mask(in string) (string, error) {
	out := in
	err := fmt.Errorf("RuleName: %s, MaskType: %s , %w", w.rule.RuleName, w.rule.MaskType, ErrMaskNotSupport)
	switch w.rule.MaskType {
	case Char:
		out, err = w.maskCharImpl(in)
	case Tag:
		out, err = w.maskStrTagImpl(in)
	case Replace:
		out, err = w.maskReplaceImpl(in)
	case Algo:
		out, err = w.maskAlgoImpl(in)
	}
	return out, err
}

const (
	// base64
	enterListRes = "6KGX6YGTfOi3r3zooZd86YeMfOadkXzplYd85bGvfOe7hAo="
	midListRes   = "56S+5Yy6fOWwj+WMunzlpKfljqZ85bm/5Zy6fOWPt+alvHzljZXlhYN85Y+3fOWxgnzlrqR85oi3Cg=="
)

var (
	enterList = make([]string, 0, 0)
	midList   = make([]string, 0, 0)
)

func init() {
	enterList = loadResList(enterListRes)
	midList = loadResList(midListRes)
}

// loadResList accepts base64 string, then convert them to string list
func loadResList(res string) []string {
	retList := make([]string, 0, 0)
	if decode, err := base64.StdEncoding.DecodeString(res); err == nil {
		trim := strings.TrimSpace(string(decode))
		retList = strings.Split(trim, "|")
	}
	return retList
}

// maskCharImpl mask in string with char
func (w *worker) maskCharImpl(in string) (string, error) {
	ch := byte('*') // default
	if len(w.rule.Value) > 0 {
		ch = w.rule.Value[0]
	}
	sz := len(in)
	out := []byte(in)
	if !w.rule.Reverse {
		cnt := 0
		st := 0
		if w.rule.Offset >= 0 {
			st = int(w.rule.Offset)
		}
		ed := sz
		if w.rule.Padding >= 0 {
			ed = sz - int(w.rule.Padding)
		}
		for i := st; i < ed; i++ {
			// if Length == 0 , do not check
			if w.rule.Length > 0 && cnt >= int(w.rule.Length) {
				break
			}
			if strings.IndexByte(w.rule.IgnoreCharSet, out[i]) == -1 { // ignore check
				out[i] = ch
			}
			cnt++
		}
	} else {
		cnt := 0
		ed := sz
		if w.rule.Offset >= 0 {
			ed = sz - 1 - int(w.rule.Offset)
		}
		st := 0
		if w.rule.Padding >= 0 {
			st = int(w.rule.Padding)
		}
		for i := ed; i >= st; i-- {
			if w.rule.Length > 0 && cnt >= int(w.rule.Length) {
				break
			}
			if strings.IndexByte(w.rule.IgnoreCharSet, out[i]) == -1 { // ignore check
				out[i] = ch
			}
			cnt++
		}
	}
	return string(out), nil
}

// maskTagImpl mask with the tag of in string
func (w *worker) maskTagImpl(in string, infoType string) (string, error) {
	return fmt.Sprintf("<%s>", infoType), nil
}

// maskReplaceImpl replace with rule.Value
func (w *worker) maskReplaceImpl(in string) (string, error) {
	return w.rule.Value, nil
}

// maskStrTagImpl first Deidentify to get infotype, then mask with infotype
func (w *worker) maskStrTagImpl(in string) (string, error) {
	if results, err := w.parent.Detect(in); err == nil {
		if len(results) > 0 {
			res := results[0]
			return w.maskTagImpl(in, res.InfoType)
		}
	}
	return w.maskTagImpl(in, UnknownTag)
}

// maskAlgoImpl replace with algo(in)
func (w *worker) maskAlgoImpl(in string) (string, error) {
	inBytes := []byte(in)
	switch w.rule.Value {
	case "BASE64":
		return base64.StdEncoding.EncodeToString(inBytes), nil
	case "MD5":
		return fmt.Sprintf("%x", md5.Sum(inBytes)), nil
	case "CRC32":
		return fmt.Sprintf("%08x", crc32.ChecksumIEEE(inBytes)), nil
	case "ADDRESS":
		return w.maskAddressImpl(in)
	case "NUMBER":
		return w.maskNumberImpl(in)
	case "DEIDENTIFY":
		return w.maskDeidentifyImpl(in)
	default:
		return in, fmt.Errorf("RuleName: %s, MaskType: %s , Value:%s, %w",
			w.rule.RuleName, w.rule.MaskType, w.rule.Value, ErrMaskNotSupport)
	}
}

// maskAddressImpl masks Address
func (w *worker) maskAddressImpl(in string) (string, error) {
	st := 0

	if pos, id := w.indexSubList(in, st, enterList, true); pos != -1 { // found
		st = pos + len(enterList[id])
	}
	out := in[:st]
	sz := len(in)
	for pos, id := w.indexSubList(in, st, midList, false); pos != -1 && st < sz; pos, id = w.indexSubList(in, st, midList, false) {
		out += strings.Repeat("*", pos-st)
		out += midList[id]
		st = pos + len(midList[id])
	}
	out += in[st:]
	out, _ = w.maskNumberImpl(out)
	if strings.Compare(in, out) == 0 { // mask Last 3 rune
		lastByteSz := 0
		for totalRune := 3; totalRune > 0 && len(out) > 0; totalRune-- {
			_, width := utf8.DecodeLastRuneInString(out)
			lastByteSz += width
			out = out[0 : len(out)-width]
		}
		out += strings.Repeat("*", lastByteSz)
	}
	return out, nil
}

// IndexSubList find index of a list of sub strings from a string
func (w *worker) indexSubList(in string, st int, list []string, isLast bool) (int, int) {
	tmp := in[st:]
	retPos := -1
	retId := -1
	for i, word := range list {
		if pos := strings.Index(tmp, word); pos != -1 { // found
			loc := st + pos
			if retPos == -1 { // first
				retPos = loc
				retId = i
				if !isLast { // not last return directly
					return retPos, retId
				}
			} else {
				if isLast {
					if loc >= retPos {
						retPos = loc
						retId = i
					}
				}
			}

		}
	}
	return retPos, retId
}

// maskNumberImpl will mask all number in the string
func (w *worker) maskNumberImpl(in string) (string, error) {
	outBytes := []byte(in)
	for i, ch := range outBytes {
		if ch >= '0' && ch <= '9' {
			outBytes[i] = '*'
		}
	}
	return string(outBytes), nil
}

func (w *worker) maskDeidentifyImpl(in string) (string, error) {
	out, _, err := w.parent.Deidentify(in)
	return out, err
}

// private func

// DIYMaskWorker stores maskFuc and maskName
type DIYMaskWorker struct {
	maskFunc func(string) (string, error)
	maskName string
}

// GetRuleName is required by api
func (d *DIYMaskWorker) GetRuleName() string {
	return d.maskName
}

// Mask is required by api
func (d *DIYMaskWorker) Mask(in string) (string, error) {
	return d.maskFunc(in)
}

// MaskResult is required by api
func (d *DIYMaskWorker) MaskResult(res *DetectResult) error {
	if out, err := d.Mask(res.Text); err == nil {
		res.MaskText = out
		return nil
	} else {
		return err
	}
}

// NewDIYMaskWorker creates api object
func (e *Engine) NewDIYMaskWorker(maskName string, maskFunc func(string) (string, error)) (api, error) {
	worker := new(DIYMaskWorker)
	worker.maskName = maskName
	worker.maskFunc = maskFunc
	return worker, nil
}

// maskStructImpl will mask a strcut object by tag mask info
// 根据tag mask里定义的脱敏规则对struct object直接脱敏, 会修改obj本身，传入指针，返回指针
func (e *Engine) maskStructImpl(inPtr any, level int) (any, error) {
	//log.Errorf("[mask] level:%d, maskStructImpl: %+v", level, inPtr)
	if level <= 0 { // call deep check
		//log.Errorf("[mask] !call deep loop detected!")
		//log.Errorf("obj: %+v", inPtr)
		return inPtr, nil
	}
	valPtr := reflect.ValueOf(inPtr)
	if valPtr.Kind() != reflect.Ptr || valPtr.IsNil() || !valPtr.IsValid() || valPtr.IsZero() {
		return inPtr, ErrMaskStructInput
	}
	val := reflect.Indirect(valPtr)
	var retErr error
	if val.CanSet() {
		if val.Kind() == reflect.Struct {
			sz := val.NumField()
			if sz > defMaxInput {
				return inPtr, fmt.Errorf("defMaxInput: %d , %w", defMaxInput, ErrMaxInputLimit)
			}
			for i := 0; i < sz; i++ {
				valField := val.Field(i)
				typeField := val.Type().Field(i)
				inStr := valField.String()
				outStr := inStr // default is orignal str
				methodName, ok := typeField.Tag.Lookup("mask")
				if !ok { // mask tag not found
					continue
				}
				if valField.CanSet() {
					switch valField.Kind() {
					case reflect.String:
						if len(methodName) > 0 {
							if maskWorker, ok := e.maskerMap[methodName]; ok {
								if masked, err := maskWorker.Mask(inStr); err == nil {
									outStr = masked
									valField.SetString(outStr)
								}
							}
						}
					case reflect.Struct:
						if valField.CanAddr() {
							//log.Errorf("[mask] Struct, %s", typeField.Name)
							_, retErr = e.maskStructImpl(valField.Addr().Interface(), level-1)
						}
					case reflect.Ptr:
						if !valField.IsNil() {
							//log.Errorf("[mask] Ptr, %s", typeField.Name)
							_, retErr = e.maskStructImpl(valField.Interface(), level-1)
						}
					case reflect.Interface:
						if valField.CanInterface() {
							valInterFace := valField.Interface()
							if inStr, ok := valInterFace.(string); ok {
								outStr := inStr
								if len(methodName) > 0 {
									if maskWorker, ok := e.maskerMap[methodName]; ok {
										if masked, err := maskWorker.Mask(inStr); err == nil {
											outStr = masked
											if valField.CanSet() {
												valField.Set(reflect.ValueOf(outStr))
											}
										}
									}
								}
							}
						}
					case reflect.Slice, reflect.Array:
						length := valField.Len()
						for i := 0; i < length; i++ {
							item := valField.Index(i)
							if item.Kind() == reflect.String {
								inStr := item.String()
								outStr := inStr
								// use parent mask info
								if len(methodName) > 0 {
									if maskWorker, ok := e.maskerMap[methodName]; ok {
										if masked, err := maskWorker.Mask(inStr); err == nil {
											outStr = masked
											if item.CanSet() {
												item.SetString(outStr)
											}
										}
									}
								}
							} else if item.Kind() == reflect.Ptr {
								if !item.IsNil() {
									//log.Errorf("[mask] Ptr, %s", item.Type().Name())
									_, retErr = e.maskStructImpl(item.Interface(), level-1)
								}
							} else if item.Kind() == reflect.Struct {
								if item.CanAddr() {
									//log.Errorf("[mask] Struct, %s", item.Type().Name())
									_, retErr = e.maskStructImpl(item.Addr().Interface(), level-1)
								}
							}
						}
					}
				}
			}
		}
	}
	return inPtr, retErr
}
