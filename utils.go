// OllamaTea Copyright (c) 2024 Neomantra Corp

package ollamatea

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	ansitoimage "github.com/pavelpatrin/go-ansi-to-image"
)

// ConvertTerminalTextToImage converts the [terminalText] to a PNG image returned as a []byte.
// Returns nil with an error, if any.
// Uses the passed [go-ansi-to-image Config](https://github.com/pavelpatrin/go-ansi-to-image/blob/main/config.go#L4)
// or otherwise the [DefaultConfig](https://github.com/pavelpatrin/go-ansi-to-image/blob/main/config.go#L28).
func ConvertTerminalTextToImage(terminalText string, convertConfig *ansitoimage.Config) ([]byte, error) {
	if convertConfig == nil {
		convertConfig = &ansitoimage.DefaultConfig
	}
	ansiConverter, err := ansitoimage.NewConverter(*convertConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create image converter %w", err)
	}

	err = ansiConverter.Parse(terminalText)
	if err != nil {
		return nil, fmt.Errorf("failed to render text %w", err)
	}

	pngBytes, err := ansiConverter.ToPNG()
	if err != nil {
		return nil, fmt.Errorf("failed to convert terminal text to PNG %w", err)
	}

	return pngBytes, nil
}

///////////////////////////////////////////////////////////////////////////////

// Cmdize is a utility function to convert a given value into a `tea.Cmd`
// https://github.com/KevM/bubbleo/blob/main/utils/utils.go
func Cmdize[T any](t T) tea.Cmd {
	return func() tea.Msg {
		return t
	}
}
