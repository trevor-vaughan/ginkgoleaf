package ginkgoleaf

import (
	"io"
	"os"
)

// ColorMode controls ANSI emission in formats that support color.
type ColorMode int

// Color modes.
const (
	ColorAuto   ColorMode = iota // honour TTY + NO_COLOR
	ColorAlways                  // emit ANSI regardless
	ColorNever                   // never emit ANSI
)

// Option mutates a Config built by NewConfig.
type Option func(*Config) error

// Config holds the resolved options for a registration.
type Config struct {
	format Format
	writer io.Writer
	color  ColorMode
}

// Format returns the selected format.
func (c *Config) Format() Format { return c.format }

// Writer returns the output writer (os.Stdout by default).
func (c *Config) Writer() io.Writer { return c.writer }

// Color returns the resolved color mode.
func (c *Config) Color() ColorMode { return c.color }

// WithWriter sets the output destination. Default: os.Stdout.
func WithWriter(w io.Writer) Option {
	return func(c *Config) error {
		c.writer = w
		return nil
	}
}

// WithColor selects the color mode. Default: ColorAuto.
func WithColor(m ColorMode) Option {
	return func(c *Config) error {
		c.color = m
		return nil
	}
}

// NewConfig builds a Config from a format and option list. Defaults:
// writer=os.Stdout, color=ColorAuto.
func NewConfig(f Format, opts ...Option) (*Config, error) {
	if err := ValidateFormat(f); err != nil {
		return nil, err
	}
	c := &Config{
		format: f,
		writer: os.Stdout,
		color:  ColorAuto,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}
