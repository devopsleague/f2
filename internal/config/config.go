// Package config is responsible for setting the program config from
// command-line arguments
package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	errInvalidArgument = errors.New(
		"Invalid argument: one of `-f`, `-r`, `-csv` or `-u` must be present and set to a non empty string value. Use 'f2 --help' for more information",
	)

	errInvalidSimpleModeArgs = errors.New(
		"At least one argument must be specified in simple mode",
	)
)

var conf *Config

// Config represents the program configuration.
type Config struct {
	Date              time.Time
	Stdin             io.Reader
	Stderr            io.Writer
	Stdout            io.Writer
	SearchRegex       *regexp.Regexp
	CSVFilename       string
	Sort              string
	Replacement       string
	WorkingDir        string
	FindSlice         []string
	ExcludeRegex      *regexp.Regexp
	ReplacementSlice  []string
	FilesAndDirPaths  []string
	NumberOffset      []int
	MaxDepth          int
	StartNumber       int
	ReplaceLimit      int
	Recursive         bool
	IgnoreCase        bool
	ReverseSort       bool
	OnlyDir           bool
	Revert            bool
	IncludeDir        bool
	IgnoreExt         bool
	AllowOverwrites   bool
	Verbose           bool
	IncludeHidden     bool
	Quiet             bool
	AutoFixConflicts  bool
	Exec              bool
	StringLiteralMode bool
	SimpleMode        bool
	JSON              bool
	Interactive       bool
}

// SetFindStringRegex compiles a regular expression for the
// find string of the corresponding replacement index (if any).
// Otherwise, the created regex will match the entire file name.
func (c *Config) SetFindStringRegex(replacementIndex int) error {
	// findPattern is set to match the entire file name by default
	// except if a find string for the corresponding replacement index
	// is found
	findPattern := ".*"
	if len(c.FindSlice) > replacementIndex {
		findPattern = c.FindSlice[replacementIndex]

		// Escape all regular expression metacharacters in string literal mode
		if c.StringLiteralMode {
			findPattern = regexp.QuoteMeta(findPattern)
		}

		if c.IgnoreCase {
			findPattern = "(?i)" + findPattern
		}
	}

	re, err := regexp.Compile(findPattern)
	if err != nil {
		return err
	}

	c.SearchRegex = re

	return nil
}

func (c *Config) setOptions(ctx *cli.Context) error {
	if len(ctx.StringSlice("find")) == 0 &&
		len(ctx.StringSlice("replace")) == 0 &&
		ctx.String("csv") == "" &&
		!ctx.Bool("undo") {
		return errInvalidArgument
	}

	c.FindSlice = ctx.StringSlice("find")
	c.ReplacementSlice = ctx.StringSlice("replace")
	c.CSVFilename = ctx.String("csv")
	c.Revert = ctx.Bool("undo")
	c.FilesAndDirPaths = ctx.Args().Slice()

	// Default to the current working directory if no path arguments are provided
	if len(c.FilesAndDirPaths) == 0 {
		c.FilesAndDirPaths = append(c.FilesAndDirPaths, ".")
	}

	// Ensure that each findString has a corresponding replacement.
	// The replacement defaults to an empty string if unset
	for len(c.FindSlice) > len(c.ReplacementSlice) {
		c.ReplacementSlice = append(c.ReplacementSlice, "")
	}

	return c.SetFindStringRegex(0)
}

// setSimpleModeOptions is used to set the options for the
// renaming operation in simpleMode.
func (c *Config) setSimpleModeOptions(ctx *cli.Context) error {
	args := ctx.Args().Slice()

	if len(args) < 1 {
		return errInvalidSimpleModeArgs
	}

	// If a replacement string is not specified, it shoud be
	// an empty string
	if len(args) == 1 {
		args = append(args, "")
	}

	minArgs := 2

	c.SimpleMode = true
	c.FindSlice = []string{args[0]}
	c.ReplacementSlice = []string{args[1]}

	// override default options
	c.IncludeDir = true
	c.Exec = true
	c.Interactive = true

	if len(args) > minArgs {
		c.FilesAndDirPaths = args[minArgs:]
	}

	return c.SetFindStringRegex(0)
}

// setDefaultOpts applies any options that may be set through
// F2_DEFAULT_OPTS.
func (c *Config) setDefaultOpts(ctx *cli.Context) error {
	c.AutoFixConflicts = ctx.Bool("fix-conflicts")
	c.IncludeDir = ctx.Bool("include-dir")
	c.IncludeHidden = ctx.Bool("hidden")
	c.IgnoreCase = ctx.Bool("ignore-case")
	c.IgnoreExt = ctx.Bool("ignore-ext")
	c.Recursive = ctx.Bool("recursive")
	c.OnlyDir = ctx.Bool("only-dir")
	c.StringLiteralMode = ctx.Bool("string-mode")
	c.MaxDepth = int(ctx.Uint("max-depth"))
	c.Verbose = ctx.Bool("verbose")
	c.AllowOverwrites = ctx.Bool("allow-overwrites")
	c.ReplaceLimit = ctx.Int("replace-limit")
	c.Quiet = ctx.Bool("quiet")
	c.JSON = ctx.Bool("json")
	c.Exec = ctx.Bool("exec")
	c.Interactive = ctx.Bool("interactive")

	excludePattern := ctx.StringSlice("exclude")
	if len(excludePattern) > 0 {
		excludeMatchRegex, err := regexp.Compile(
			strings.Join(excludePattern, "|"),
		)
		if err != nil {
			return err
		}

		c.ExcludeRegex = excludeMatchRegex
	}

	if c.Interactive {
		c.Exec = true
	}

	// Sorting
	if ctx.String("sort") != "" {
		c.Sort = ctx.String("sort")
	} else if ctx.String("sortr") != "" {
		c.Sort = ctx.String("sortr")
		c.ReverseSort = true
	}

	if c.OnlyDir {
		c.IncludeDir = true
	}

	return nil
}

func SetReplacement(replacement string) {
	conf.Replacement = replacement
}

func SetFindStringRegex(replacementIndex int) error {
	return conf.SetFindStringRegex(replacementIndex)
}

func SetReplacementSlice(s []string) {
	conf.ReplacementSlice = s
}

func SetFindSlice(s []string) {
	conf.FindSlice = s
}

func SetNumberOffset(offset []int) {
	conf.NumberOffset = offset
}

// Get retrives an already set config or panics if the configuration
// has not yet been initialized.
func Get() *Config {
	if conf == nil {
		panic("config has not been initialized")
	}

	return conf
}

func Init(ctx *cli.Context) (*Config, error) {
	conf = &Config{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
		Date:   time.Now(),
	}

	v, exists := ctx.App.Metadata["reader"]
	if exists {
		r, ok := v.(io.Reader)
		if ok {
			conf.Stdin = r
		}
	}

	v, exists = ctx.App.Metadata["writer"]
	if exists {
		w, ok := v.(io.Writer)
		if ok {
			conf.Stdout = w
		}
	}

	var err error

	err = conf.setDefaultOpts(ctx)
	if err != nil {
		return nil, err
	}

	if _, ok := ctx.App.Metadata["simple-mode"]; ok {
		err = conf.setSimpleModeOptions(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		err = conf.setOptions(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Get the current working directory
	conf.WorkingDir, err = filepath.Abs(".")
	if err != nil {
		return nil, err
	}

	return conf, nil
}
