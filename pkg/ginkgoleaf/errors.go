package ginkgoleaf

import (
	"errors"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

// Sentinel errors. Match with errors.Is.
var (
	// ErrUnknownFormat is returned/panicked when an unsupported Format is given.
	// It is the same sentinel render.ValidateFormat returns.
	ErrUnknownFormat = render.ErrUnknownFormat

	// ErrAlreadyRegistered is panicked on a second Register call in the same suite.
	ErrAlreadyRegistered = errors.New("ginkgoleaf: already registered")
)
