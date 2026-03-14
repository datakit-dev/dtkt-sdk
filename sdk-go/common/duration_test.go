package common_test

import (
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/stretchr/testify/assert"
)

func TestDuration(t *testing.T) {
	var tests = map[string]time.Duration{
		"1s":     time.Second,
		"1m0s":   time.Minute,
		"3h0m0s": 3 * time.Hour,
	}

	for str, dur1 := range tests {
		dur2, err := common.Duration(str).ToDuration()
		assert.NoError(t, err)
		assert.Equal(t, dur1, dur2)
		assert.Equal(t, str, common.NewDuration(dur1).String())
	}
}

func TestDuration_Validator(t *testing.T) {
	var tests = map[string]bool{
		"1s":  true,
		"-5m": false,
		"13h": true,
	}

	for str, valid := range tests {
		err := common.DurationValidator(str)
		if valid {
			assert.NoError(t, err)
		}
	}
}

func TestDuration_RangeValidator(t *testing.T) {
	type rangeTest struct {
		min   time.Duration
		max   time.Duration
		valid bool
	}

	var tests = map[string]rangeTest{
		"5s": {
			min:   time.Second,
			max:   time.Hour,
			valid: true,
		},
		"10s": {
			min:   11 * time.Second,
			max:   time.Hour,
			valid: false,
		},
		"13h": {
			min:   time.Minute,
			max:   12 * time.Hour,
			valid: false,
		},
	}

	for str, test := range tests {
		var valid = common.DurationRangeValidator(test.min, test.max)
		err := valid(str)
		if test.valid {
			assert.NoError(t, err)
		}
	}
}
