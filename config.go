// OllamaTea Copyright (c) 2024 Neomantra Corp

package ollamatea

import (
	"os"
	"strings"
)

///////////////////////////////////////////////////////////////////////////////
// Default Configuration for OllamaTea, extracted from environment

var (
	defaultOllamaHost   = "http://localhost:11434" // OLLAMATEA_HOST overrides
	defaultOllamaModel  = "llama3.2-vision:11b"    // OLLAMATEA_MODEL overrides
	defaultOllamaPrompt = ""                       // OLLAMATEA_PROMPT overrides
	defaultOllamaSystem = ""                       // OLLAMATEA_SYSTEM overrides
)

func init() {
	if ollamaNoEnv := os.Getenv("OLLAMATEA_NOENV"); ollamaNoEnv != "" {
		ollamaNoEnv = strings.ToLower(ollamaNoEnv)
		if ollamaNoEnv == "true" || ollamaNoEnv == "yes" || ollamaNoEnv == "1" {
			return
		}
	}
	if ollamaHost := os.Getenv("OLLAMATEA_HOST"); ollamaHost != "" {
		defaultOllamaHost = ollamaHost
	}
	if ollamaModel := os.Getenv("OLLAMATEA_MODEL"); ollamaModel != "" {
		defaultOllamaModel = ollamaModel
	}
	if ollamaPrompt := os.Getenv("OLLAMATEA_PROMPT"); ollamaPrompt != "" {
		defaultOllamaPrompt = ollamaPrompt
	}
	if ollamaSystem := os.Getenv("OLLAMATEA_SYSTEM"); ollamaSystem != "" {
		defaultOllamaSystem = ollamaSystem
	}
}

func DefaultHost() string {
	return defaultOllamaHost
}

func DefaultModel() string {
	return defaultOllamaModel
}

func DefaultPrompt() string {
	return defaultOllamaPrompt
}

func DefaultSystemPrompt() string {
	return defaultOllamaSystem
}
