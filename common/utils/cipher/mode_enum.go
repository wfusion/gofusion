package cipher

import (
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
)

//go:generate stringer -type=Mode -trimprefix=Mode
type Mode uint8

const (
	ModeUnknown Mode = iota
	ModeECB
	ModeCBC
	ModeCFB
	ModeCTR
	ModeOFB
	ModeGCM
	// modeStream may be ECB but not, it is unnecessary for padding like ECB or CBC
	modeStream
)

var (
	modeEnum = utils.NewEnumString[Mode, []Mode](
		map[Mode]string{
			ModeECB:    "ecb",
			ModeCBC:    "cbc",
			ModeCFB:    "cfb",
			ModeCTR:    "ctr",
			ModeOFB:    "ofb",
			ModeGCM:    "gcm",
			modeStream: "stream",
		},
	)
	ivMode      = utils.NewSet(ModeCBC, ModeCFB, ModeCTR, ModeOFB)
	streamMode  = utils.NewSet(ModeCFB, ModeCTR, ModeOFB, ModeGCM, modeStream)
	paddingMode = utils.NewSet(ModeECB, ModeCBC)
)

func (m Mode) Value() uint8 {
	return uint8(m)
}

func (m Mode) String() string {
	return modeEnum.String(m)
}

func (m Mode) IsValid() bool {
	return modeEnum.IsValid(m)
}

func (m Mode) ShouldPadding() bool {
	return paddingMode.Contains(m)
}

func (m Mode) NeedIV() bool {
	return ivMode.Contains(m)
}

func (m Mode) SupportStream() bool {
	return streamMode.Contains(m)
}

func ParseMode(s any) Mode {
	switch v := s.(type) {
	case string:
		if enumList := modeEnum.Enum(v); len(enumList) > 0 {
			return enumList[0]
		}
	case Mode:
		return v
	default:
		return Mode(cast.ToInt(s))
	}
	return ModeUnknown
}
