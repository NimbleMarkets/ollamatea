// OllamaTea Copyright (c) 2024 Neomantra Corp
// ot-embed

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/NimbleMarkets/ollamatea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"
)

/////////////////////////////////////////////////////////////////////////////////////

var usageFormatShort string = `usage:  %s [--help] [options] --in <input-filename>`

var usageFormat string = `usage:  %s [--help] [options] --in <input-filename>

Creates an embedding for the input data.
Outputs as JSON to output, or per --out.

Example:  $ ot-embed --in hello.txt -m llava

`

/////////////////////////////////////////////////////////////////////////////////////
// Simple BubbleTea model that does the embedding and exits

type model struct {
	EmbedSession ollamatea.EmbedSession
}

func (m model) Init() tea.Cmd {
	m.EmbedSession.Init()
	return m.EmbedSession.StartEmbedCmd() // Kick off an embed
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.EmbedSession.Update(msg)

	switch msg := msg.(type) {
	case ollamatea.EmbedResponseMsg:
		if msg.ID != m.EmbedSession.ID() {
			return m, nil // Ignore messages for other sessions
		}
		return m, tea.Quit
	case ollamatea.EmbedErrorMsg:
		// Quit after the first message
		return m, tea.Quit
	}
	_, cmd := m.EmbedSession.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return ""
}

/////////////////////////////////////////////////////////////////////////////////////

func main() {
	var inputFilename, outputFilename string
	var ollamaHost, ollamaModel string
	var verbose, showHelp bool

	pflag.StringVarP(&inputFilename, "in", "i", "", "Input filename ('-' is stdin)")
	pflag.StringVarP(&outputFilename, "out", "o", "", "Output filename ('-' is stdout)")
	pflag.StringVarP(&ollamaHost, "host", "h", ollamatea.DefaultHost(), "Host for Ollama (also OLLAMATEA_HOST env)")
	pflag.StringVarP(&ollamaModel, "model", "m", ollamatea.DefaultModel(), "Model for Ollama (also OLLAMATEA_MODEL env)")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	pflag.BoolVarP(&showHelp, "help", "", false, "show help")
	pflag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stdout, usageFormat, os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}
	if len(inputFilename) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: missing required argument: --in\n")
		fmt.Fprintf(os.Stderr, usageFormatShort, os.Args[0])
		os.Exit(1)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "INFO: ohost=%s omodel=%s\n", ollamaHost, ollamaModel)
	}

	// Open input file for reading, or use Stdin
	var err error
	infile := os.Stdin
	if len(inputFilename) != 0 && inputFilename != "-" {
		infile, err = os.OpenFile(inputFilename, os.O_RDONLY, 0777)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to open input file %s\n", err.Error())
			os.Exit(1)
		}
		defer infile.Close()
	}

	// Open output file now, or use Stdout.  Error now rather than after an whole embed request
	outfile := os.Stdout
	if outputFilename != "" && outputFilename != "-" {
		outfile, err = os.OpenFile(outputFilename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to open output file %s\n", err.Error())
			os.Exit(1)
		}
		defer outfile.Close()
	}

	// Capture input until EOF
	inputData, err := io.ReadAll(infile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read file %s\n", err.Error())
		os.Exit(1)
	}
	infile.Close() // we don't need it anymore

	// Use ollamatea.EmbedSession's machinery to embed input
	s := ollamatea.NewEmbedSession(
		ollamatea.WithHost(ollamaHost),
		ollamatea.WithModel(ollamaModel),
		ollamatea.WithInput(inputData))
	m := model{EmbedSession: s}

	mret, err := tea.NewProgram(m, tea.WithInput(nil)).Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
	m = mret.(model)

	// Check response
	resp := m.EmbedSession.Response()
	if resp == nil {
		if err := m.EmbedSession.Error(); err != nil {
			fmt.Fprintf(os.Stderr, "Embedding failed: %s\n", err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "no embedding response\n")
		}
		os.Exit(1)
	}
	jstr, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to JSON marshal response %s\n", err.Error())
		os.Exit(1)
	}

	// Write JSON
	_, err = outfile.Write(jstr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write response %s\n", err.Error())
		os.Exit(1)
	}
	outfile.WriteString("\n")
}
