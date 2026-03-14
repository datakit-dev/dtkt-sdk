package tippecanoe

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
)

var (
	DefaultConfig = NewConfig([]string{
		"-zg",
		// "--force",
		// "--minimum-zoom=10",
		// "--maximum-zoom=16",
		"--drop-densest-as-needed",
		"--extend-zooms-if-still-dropping",
		"--coalesce-densest-as-needed",
		"--detect-shared-borders",
		"--simplify-only-low-zooms",
		// "--preserve-input-order",
		// "--no-tile-size-limit",
		// "--no-feature-limit",

	}, false)
	MobileOptimizedConfig = NewConfig([]string{
		"--minimum-zoom=8",
		"--maximum-zoom=14",
		"--drop-densest-as-needed",
		"--coalesce-densest-as-needed",
		"--simplify-only-low-zooms",
		"--detect-shared-borders",
		"--no-feature-limit",
		"--maximum-tile-bytes=500000", // ~500KB tile cap
	}, false)
	HighDetailConfig = NewConfig([]string{
		"--minimum-zoom=12",
		"--maximum-zoom=18",
		"--no-feature-limit",
		"--no-tile-size-limit",
		"--no-line-simplification",
		"--no-tiny-polygon-reduction",
		"--preserve-input-order",
		"--detect-shared-borders",
	}, false)
	FastPreviewConfig = NewConfig([]string{
		"--minimum-zoom=10",
		"--maximum-zoom=12",
		"--drop-rate=0.75",
		"--coalesce-densest-as-needed",
		"--simplify-only-low-zooms",
		"--preserve-input-order",
	}, true)
)

type (
	// Config defines a reusable base set of options for constructing a tippecanoe Command.
	Config struct {
		Flags     []string
		LogOutput bool
	}
)

func NewConfig(flags []string, logOutput bool) Config {
	return Config{
		Flags:     append([]string{}, flags...), // defensive copy
		LogOutput: logOutput,
	}
}

func WithRawNDJSONReader(r io.Reader) Option {
	return func(c *Command) error {
		c.stdin = r
		c.files = append(c.files, "/dev/stdin")
		return nil
	}
}

func WithInputReader(r io.Reader) Option {
	return func(c *Command) error {
		// Wrap the raw NDJSON input as a valid FeatureCollection stream
		c.stdin = NewFeatureCollectionStream(r)

		// Tippecanoe will read from stdin
		c.files = append(c.files, "/dev/stdin")
		return nil
	}
}

// WithInputFileAutoWrap detects NDJSON and wraps it into a FeatureCollection
func WithInputFileAutoWrap(path string) Option {
	return func(c *Command) error {
		isNDJSON := filepath.Ext(path) == ".ndgeojson" || filepath.Ext(path) == ".ndjson"

		if !isNDJSON {
			c.files = append(c.files, path)
			return nil
		}

		// Create a temp file for the wrapped GeoJSON
		tempFile, err := os.CreateTemp("", "wrapped-*.geojson")
		if err != nil {
			return err
		}
		tempPath := tempFile.Name()

		// Track for deletion after command completes
		c.files = append(c.files, tempPath)
		if c.cleanup == nil {
			c.cleanup = make([]func(), 0)
		}
		c.cleanup = append(c.cleanup, func() {
			err = os.Remove(tempPath)
			if err != nil {
				slog.Error("Error removing temp file", slog.String("path", tempPath), slog.Any("error", err))
			}
		})

		// Wrap the NDJSON input into a FeatureCollection
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			if err := in.Close(); err != nil {
				slog.Error("failed to close input", log.Err(err))
			}
		}()

		err = convertNDJSONToFeatureCollection(in, tempFile)
		_ = tempFile.Close()
		return err
	}
}

//
// Functional option helpers
//

// WithLogOutput enables (or disables) logging of stdout/stderr as the command runs.
func WithLogOutput(enable bool) Option {
	return func(c *Command) error {
		c.logOutput = enable
		return nil
	}
}

//
// Output Tileset Options
//

// WithOutputFile sets the --output flag with the given file name.
func WithOutputFile(file string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--output=%s", filepath.Base(file)))
		c.outputFile = file
		return nil
	}
}

// WithOutputToDirectory sets the --output-to-directory flag.
func WithOutputToDirectory(dir string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--output-to-directory=%s", dir))
		return nil
	}
}

// WithForce adds the --force flag.
func WithForce(force bool) Option {
	return func(c *Command) error {
		if force {
			c.flags = append(c.flags, "--force")
		}
		return nil
	}
}

// WithAllowExisting adds the --allow-existing flag.
func WithAllowExisting(allow bool) Option {
	return func(c *Command) error {
		if allow {
			c.flags = append(c.flags, "--allow-existing")
		}
		return nil
	}
}

//
// Tileset Description and Attribution Options
//

// WithName sets the --name flag.
func WithName(name string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--name=%s", name))
		return nil
	}
}

// WithAttribution sets the --attribution flag.
func WithAttribution(attr string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--attribution=%s", attr))
		return nil
	}
}

// WithDescription sets the --description flag.
func WithDescription(desc string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--description=%s", desc))
		return nil
	}
}

//
// Input Files and Layers
//

// WithLayer adds a --layer flag. Can be used multiple times.
func WithLayer(layer string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--layer=%s", layer))
		return nil
	}
}

// WithNamedLayer adds a --named-layer flag.
func WithNamedLayer(layer string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--named-layer=%s", layer))
		return nil
	}
}

// WithInputFiles adds one or more positional input filenames.
func WithInputFiles(files ...string) Option {
	return func(c *Command) error {
		c.files = append(c.files, files...)
		return nil
	}
}

//
// Parallel Processing & Projection
//

// WithReadParallel adds the --read-parallel flag.
func WithReadParallel(enable bool) Option {
	return func(c *Command) error {
		if enable {
			c.flags = append(c.flags, "--read-parallel")
		}
		return nil
	}
}

// WithProjection sets the --projection flag.
func WithProjection(proj string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--projection=%s", proj))
		return nil
	}
}

//
// Zoom Options
//

// WithMaximumZoom sets the --maximum-zoom flag.
func WithMaximumZoom(zoom int) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--maximum-zoom=%d", zoom))
		return nil
	}
}

// WithMinimumZoom sets the --minimum-zoom flag.
func WithMinimumZoom(zoom int) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--minimum-zoom=%d", zoom))
		return nil
	}
}

// WithSmallestMaximumZoomGuess sets the --smallest-maximum-zoom-guess flag.
func WithSmallestMaximumZoomGuess(guess int) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--smallest-maximum-zoom-guess=%d", guess))
		return nil
	}
}

// WithExtendZoomsIfStillDropping adds the --extend-zooms-if-still-dropping flag.
func WithExtendZoomsIfStillDropping(enable bool) Option {
	return func(c *Command) error {
		if enable {
			c.flags = append(c.flags, "--extend-zooms-if-still-dropping")
		}
		return nil
	}
}

// WithExtendZoomsIfStillDroppingMaximum sets the --extend-zooms-if-still-dropping-maximum flag.
func WithExtendZoomsIfStillDroppingMaximum(zoom int) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--extend-zooms-if-still-dropping-maximum=%d", zoom))
		return nil
	}
}

// WithGenerateVariableDepthTilePyramid adds the --generate-variable-depth-tile-pyramid flag.
func WithGenerateVariableDepthTilePyramid(enable bool) Option {
	return func(c *Command) error {
		if enable {
			c.flags = append(c.flags, "--generate-variable-depth-tile-pyramid")
		}
		return nil
	}
}

// WithOneTile sets the --one-tile flag.
func WithOneTile(tile string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--one-tile=%s", tile))
		return nil
	}
}

//
// Tile Resolution Options
//

// WithFullDetail sets the --full-detail flag.
func WithFullDetail(detail int) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--full-detail=%d", detail))
		return nil
	}
}

// WithLowDetail sets the --low-detail flag.
func WithLowDetail(detail int) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--low-detail=%d", detail))
		return nil
	}
}

// WithMinimumDetail sets the --minimum-detail flag.
func WithMinimumDetail(detail int) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--minimum-detail=%d", detail))
		return nil
	}
}

// WithExtraDetail sets the --extra-detail flag.
func WithExtraDetail(detail int) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--extra-detail=%d", detail))
		return nil
	}
}

//
// Temporary Storage and Progress Indicator Options
//

// WithTemporaryDirectory sets the --temporary-directory flag.
func WithTemporaryDirectory(dir string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--temporary-directory=%s", dir))
		return nil
	}
}

// WithQuiet adds the --quiet flag.
func WithQuiet(quiet bool) Option {
	return func(c *Command) error {
		if quiet {
			c.flags = append(c.flags, "--quiet")
		}
		return nil
	}
}

//
// Generic options for any flag not yet covered
//

// WithRawFlag adds a flag as-is. For boolean flags, pass the flag exactly as tippecanoe expects (e.g., "--drop-lines").
func WithRawFlag(flag string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, flag)
		return nil
	}
}

// WithCustomOption adds an option using the format "--flag=value".
func WithCustomOption(flag, value string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--%s=%s", flag, value))
		return nil
	}
}

func WithCompression(algo string) Option {
	return func(c *Command) error {
		c.flags = append(c.flags, fmt.Sprintf("--compression=%s", algo))
		return nil
	}
}

func WithOutputTempFile(dir, format string) Option {
	return func(c *Command) error {
		tmp, err := os.CreateTemp(dir, "*."+format)
		if err != nil {
			return err
		}
		err = tmp.Close()
		if err != nil {
			slog.Error("failed to close temp file", log.Err(err))
		}

		c.flags = append(c.flags, "-o "+tmp.Name())
		c.outputFile = tmp.Name()

		// Clean up later
		c.cleanup = append(c.cleanup, func() {
			err = os.Remove(tmp.Name())
			if err != nil {
				slog.Error("failed to remove temp file", slog.String("path", tmp.Name()), log.Err(err))
			}
		})
		return nil
	}
}
