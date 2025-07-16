package cases

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestDuration(t *testing.T) {
	t.Parallel()
	testingSuite := &Duration{Test: new(testUtl.Test)}
	suite.Run(t, testingSuite)
}

type Duration struct {
	*testUtl.Test
}

func (t *Duration) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Duration) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Duration) TestParseDuration() {
	t.Catch(func() {
		var parseDurationTests = []struct {
			in   string
			want time.Duration
		}{
			// simple
			{"0", 0},
			{"5s", 5 * time.Second},
			{"30s", 30 * time.Second},
			{"1478s", 1478 * time.Second},
			// sign
			{"-5s", -5 * time.Second},
			{"+5s", 5 * time.Second},
			{"-0", 0},
			{"+0", 0},
			// decimal
			{"5.0s", 5 * time.Second},
			{"5.6s", 5*time.Second + 600*time.Millisecond},
			{"5.s", 5 * time.Second},
			{".5s", 500 * time.Millisecond},
			{"1.0s", 1 * time.Second},
			{"1.00s", 1 * time.Second},
			{"1.004s", 1*time.Second + 4*time.Millisecond},
			{"1.0040s", 1*time.Second + 4*time.Millisecond},
			{"100.00100s", 100*time.Second + 1*time.Millisecond},
			// different units
			{"10ns", 10 * time.Nanosecond},
			{"11us", 11 * time.Microsecond},
			{"12µs", 12 * time.Microsecond}, // U+00B5
			{"12μs", 12 * time.Microsecond}, // U+03BC
			{"13ms", 13 * time.Millisecond},
			{"14s", 14 * time.Second},
			{"15m", 15 * time.Minute},
			{"16h", 16 * time.Hour},
			// composite durations
			{"3h30m", 3*time.Hour + 30*time.Minute},
			{"10.5s4m", 4*time.Minute + 10*time.Second + 500*time.Millisecond},
			{"-2m3.4s", -(2*time.Minute + 3*time.Second + 400*time.Millisecond)},
			{"1h2m3s4ms5us6ns", 1*time.Hour + 2*time.Minute + 3*time.Second + 4*time.Millisecond + 5*time.Microsecond + 6*time.Nanosecond},
			{"39h9m14.425s", 39*time.Hour + 9*time.Minute + 14*time.Second + 425*time.Millisecond},
			// large value
			{"52763797000ns", 52763797000 * time.Nanosecond},
			// more than 9 digits after decimal point, see https://golang.org/issue/6617
			{"0.3333333333333333333h", 20 * time.Minute},
			// 9007199254740993 = 1<<53+1 cannot be stored precisely in a float64
			{"9007199254740993ns", (1<<53 + 1) * time.Nanosecond},
			// largest duration that can be represented by int64 in nanoseconds
			{"9223372036854775807ns", (1<<63 - 1) * time.Nanosecond},
			{"9223372036854775.807us", (1<<63 - 1) * time.Nanosecond},
			{"9223372036s854ms775us807ns", (1<<63 - 1) * time.Nanosecond},
			{"-9223372036854775808ns", -1 << 63 * time.Nanosecond},
			{"-9223372036854775.808us", -1 << 63 * time.Nanosecond},
			{"-9223372036s854ms775us808ns", -1 << 63 * time.Nanosecond},
			// largest negative value
			{"-9223372036854775808ns", -1 << 63 * time.Nanosecond},
			// largest negative round trip value, see https://golang.org/issue/48629
			{"-2562047h47m16.854775808s", -1 << 63 * time.Nanosecond},
			// huge string; issue 15011.
			{"0.100000000000000000000h", 6 * time.Minute},
			// This value tests the first overflow check in leadingFraction.
			{"0.830103483285477580700h", 49*time.Minute + 48*time.Second + 372539827*time.Nanosecond},
		}

		for _, tc := range parseDurationTests {
			d, err := utils.ParseDuration(tc.in)
			if err != nil || d != tc.want {
				t.Errorf(err, "ParseDuration(%q) = %v, %v, want %v, nil", tc.in, d, err, tc.want)
			}
		}
	})
}

func (t *Duration) TestDayWeek() {
	t.Catch(func() {
		var parseDurationDayWeekTests = []struct {
			in   string
			want time.Duration
		}{
			// Day unit tests
			{"1d", 24 * time.Hour},
			{"2d", 48 * time.Hour},
			{"7d", 7 * 24 * time.Hour},
			{"0.5d", 12 * time.Hour},
			{"1.5d", 36 * time.Hour},
			{"0.25d", 6 * time.Hour},
			{"2.75d", 66 * time.Hour},
			{"-1d", -24 * time.Hour},
			{"-2.5d", -60 * time.Hour},
			{"+3d", 72 * time.Hour},

			// Week unit tests
			{"1w", 7 * 24 * time.Hour},
			{"2w", 14 * 24 * time.Hour},
			{"0.5w", 84 * time.Hour},  // 3.5 days
			{"1.5w", 252 * time.Hour}, // 10.5 days
			{"-1w", -7 * 24 * time.Hour},
			{"-2w", -14 * 24 * time.Hour},
			{"+1w", 7 * 24 * time.Hour},

			// Composite durations with days and weeks
			{"1w1d", 8 * 24 * time.Hour},
			{"2w3d", 17 * 24 * time.Hour},
			{"1w2d3h", 7*24*time.Hour + 2*24*time.Hour + 3*time.Hour},
			{"1w1d1h1m1s", 7*24*time.Hour + 24*time.Hour + time.Hour + time.Minute + time.Second},
			{"2w1d12h30m45s", 14*24*time.Hour + 24*time.Hour + 12*time.Hour + 30*time.Minute + 45*time.Second},

			// Mixed with standard units
			{"1d12h", 36 * time.Hour},
			{"1d30m", 24*time.Hour + 30*time.Minute},
			{"1d1h1m1s1ms1us1ns", 24*time.Hour + time.Hour + time.Minute + time.Second + time.Millisecond + time.Microsecond + time.Nanosecond},
			{"1w1d1h1m1s1ms1us1ns", 7*24*time.Hour + 24*time.Hour + time.Hour + time.Minute + time.Second + time.Millisecond + time.Microsecond + time.Nanosecond},

			// Decimal combinations
			{"1.5d12h", 48 * time.Hour},
			{"0.5w2d", (3.5 + 2) * 24 * time.Hour},      // 0.5w = 3.5d, plus 2d = 5.5d
			{"2.5w1.5d", (17.5 + 1.5) * 24 * time.Hour}, // 2.5w = 17.5d, plus 1.5d = 19d
		}

		for _, tc := range parseDurationDayWeekTests {
			d, err := utils.ParseDuration(tc.in)
			if err != nil || d != tc.want {
				t.Errorf(err, "ParseDuration(%q) = %v, %v, want %v, nil", tc.in, d, err, tc.want)
			}
		}
	})
}

func (t *Duration) TestErrors() {
	t.Catch(func() {
		var parseDurationErrorTests = []struct {
			in     string
			expect string
		}{
			// invalid
			{"", `""`},
			{"3", `"3"`},
			{"-", `"-"`},
			{"s", `"s"`},
			{".", `"."`},
			{"-.", `"-."`},
			{".s", `".s"`},
			{"+.s", `"+.s"`},
			{"\\x85\\x85", `"\\x85\\x85"`},
			{"\\xffff", `"\\xffff"`},
			{"hello \\xffff world", `"hello \\xffff world"`},
			{"\\uFFFD", `"\\uFFFD"`},                                         // utf8.RuneError
			{"\\uFFFD hello \\uFFFD world", `"\\uFFFD hello \\uFFFD world"`}, // utf8.RuneError
			// overflow
			{"9223372036854775810ns", `"9223372036854775810ns"`},
			{"9223372036854775808ns", `"9223372036854775808ns"`},
			{"-9223372036854775809ns", `"-9223372036854775809ns"`},
			{"9223372036854776us", `"9223372036854776us"`},
			{"3000000h", `"3000000h"`},
			{"9223372036854775.808us", `"9223372036854775.808us"`},
			{"9223372036854ms775us808ns", `"9223372036854ms775us808ns"`},
		}

		for _, tc := range parseDurationErrorTests {
			_, err := utils.ParseDuration(tc.in)
			if err == nil {
				t.Error(fmt.Errorf("ParseDuration(%q) = _, nil, want _, non-nil", tc.in))
			} else if !strings.Contains(err.Error(), tc.expect) {
				t.Errorf(err, "ParseDuration(%q) = _, %q, error does not contain %q", tc.in, err, tc.expect)
			}
		}
	})
}

func (t *Duration) TestBoundary() {
	t.Catch(func() {
		var parseDurationTests = []struct {
			in   string
			want time.Duration
		}{
			{"0", 0},
			{"0d", 0},
			{"0w", 0},
			{"1ns", time.Nanosecond},
			{"365d", 365 * 24 * time.Hour},
			{"52w", 52 * 7 * 24 * time.Hour},
			{"366d", 366 * 24 * time.Hour},
		}

		for _, tc := range parseDurationTests {
			d, err := utils.ParseDuration(tc.in)
			if err != nil || d != tc.want {
				t.Errorf(err, "ParseDuration(%q) = %v, %v, want %v, nil", tc.in, d, err, tc.want)
			}
		}

		// for very small values, a certain precision error is permitted
		parseDurationTests = []struct {
			in   string
			want time.Duration
		}{
			{"0.000000000011574d", 1 * time.Nanosecond},
		}
		for _, tc := range parseDurationTests {
			got, err := utils.ParseDuration(tc.in)
			if err != nil {
				t.Errorf(err, "ParseDuration(%q) = %v, %v, want %v, nil", tc.in, got, err, tc.want)
			}
			if got <= 0 || got > 1000*time.Nanosecond {
				t.Error(errors.Errorf("ParseDuration(%q) = %v, want approximately %v", tc.in, got, tc.want))
			}
		}
	})
}

func (t *Duration) TestCompatibility() {
	t.Catch(func() {
		standardTests := []string{
			"0",
			"1ns", "1us", "1µs", "1ms", "1s", "1m", "1h",
			"1.5s", "2m30s", "1h30m", "1h30m45s",
			"-1s", "+1s",
			"72h3m0.5s",
		}

		for _, test := range standardTests {
			stdResult, stdErr := time.ParseDuration(test)
			ourResult, ourErr := utils.ParseDuration(test)

			if stdErr != nil && ourErr == nil {
				t.Errorf(stdErr, "std failed but we success: %q, std failed: %v", test, stdErr)
			} else if stdErr == nil && ourErr != nil {
				t.Errorf(ourErr, "std success but we failed: %q, we failed: %v", test, ourErr)
			} else if stdErr == nil && ourErr == nil && stdResult != ourResult {
				t.Error(errors.Errorf("std and we both success but result not equal: %q, std: %v, our's: %v",
					test, stdResult, ourResult))
			}
		}
	})
}

func (t *Duration) TestOverflow() {
	t.Catch(func() {
		overflowTests := []string{
			"9223372036854775808ns", // exceeds the maximum int64
			"153722867280912w",      // week overflow
			"1000000000000000d",     // day overflow
			"999999999999999w999999999999999d999999999999999h", // overflow
		}

		for _, test := range overflowTests {
			_, err := utils.ParseDuration(test)
			if err == nil {
				t.Error(errors.Errorf("ParseDuration(%q) should overflow but return nil", test))
			} else if !strings.Contains(err.Error(), "invalid duration") {
				t.Errorf(err, "ParseDuration(%q) unexpected error: %v", test, err)
			}
		}
	})
}
