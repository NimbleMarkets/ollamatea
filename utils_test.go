// Ollama Tea Copyright (c) 2024 Neomantra Corp
//
// ./bin/ot-ansi-to-png --in tests/hello.txt --out tests/hello.png
//

package ollamatea

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConverter tests Converter type.
func TestConvertTerminalTextToImage(t *testing.T) {
	assert := require.New(t)

	terminalText, err := os.ReadFile(path.Join("tests", "hello.txt"))
	assert.NoError(err, "ReadFile TXT should return no error")

	pngBytes, err := os.ReadFile(path.Join("tests", "hello.png"))
	assert.NoError(err, "ReadFile PNG should return no error")

	convertedBytes, err := ConvertTerminalTextToImage(string(terminalText), nil)
	assert.NoError(err, "ConvertTerminalTextToImage should return no error")

	assert.Equal(pngBytes, convertedBytes)
}
