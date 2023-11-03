package constant

import "regexp"

var (
	NumberReg    = regexp.MustCompile(`\d+`)
	NonNumberReg = regexp.MustCompile(`\D+`)

	LetterReg    = regexp.MustCompile(`[a-zA-Z]+`)
	NonLetterReg = regexp.MustCompile(`[^a-zA-Z]+`)

	NumberLetterReg    = regexp.MustCompile(`[a-zA-Z0-9]+`)
	NonNumberLetterReg = regexp.MustCompile(`[^a-zA-Z0-9]+`)

	// FlavorNameReg
	//nolint: revive // reg expression issue
	FlavorNameReg = regexp.MustCompile(`(?P<prefix>([scm]|pi|pak|pck|lite|ir)\d+|[lkhf][scm]\d+|p2v(s)?|g\d+(s)?|p8a)\.(?P<middle>small|medium|large|(\d+)*xlarge)\.(?P<suffix>\d+(\.\d)?)`)

	// FullFlavorNameReg
	//nolint: revive // reg expression issue
	FullFlavorNameReg = regexp.MustCompile(`^(?P<prefix>([scm]|pi|pak|pck|lite|ir)\d+|[lkhf][scm]\d+|p2v(s)?|g\d+(s)?|p8a)\.(?P<middle>small|medium|large|(\d+)*xlarge)\.(?P<suffix>\d+(\.\d)?)$`)
)
