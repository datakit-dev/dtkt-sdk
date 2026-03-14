package common

import (
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	"github.com/invopop/jsonschema"
)

var validDuration = regexp.MustCompile(`[-+]?([0-9]*(\.[0-9]*)?[a-z]+)+`)

type Duration string

func NewDuration(d time.Duration) Duration {
	return Duration(d.String())
}

func DurationValidator(s string) error {
	if d, err := Duration(s).ToDuration(); err != nil {
		return err
	} else if d <= 0 {
		return fmt.Errorf("duration must be greater than zero")
	}
	return nil
}

func DurationRangeValidator(min, max time.Duration) func(string) error {
	return func(s string) error {
		if d, err := Duration(s).ToDuration(); err != nil {
			return err
		} else if d < min {
			return fmt.Errorf("duration must be greater than: %s", min.String())
		} else if d > max {
			return fmt.Errorf("duration must be less than: %s", max.String())
		}
		return nil

	}
}

func (d Duration) ToDuration() (time.Duration, error) {
	return time.ParseDuration(d.String())
}

func (d Duration) AddToTime(t time.Time) (time.Time, error) {
	dur, err := d.ToDuration()
	if err != nil {
		return time.Time{}, err
	}

	return t.Add(dur), nil
}

func (d Duration) String() string {
	if d == "" {
		return "0"
	}
	return string(d)
}

func (d *Duration) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("duration must be a string")
	}

	*d = Duration(str)

	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (d Duration) MarshalGQL(w io.Writer) {
	_, err := fmt.Fprintf(w, "%q", d)
	if err != nil {
		slog.Error("error writing to io.Writer", log.Err(err))
	}
}

func (Duration) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Title:       "Duration",
		Description: `Time duration expression, e.g.: "1s", "2.3h" or "4h35m"`,
		Pattern:     validDuration.String(),
	}
}
