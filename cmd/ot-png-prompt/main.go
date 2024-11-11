// OllamaTea Copyright (c) 2024 Neomantra Corp
// ot-png-prompt

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/NimbleMarkets/ollamatea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"
)

/////////////////////////////////////////////////////////////////////////////////////

const defaultOllamaPrompt = "Describe this image for a visually impaired person"

var usageFormatShort string = `usage:  %s [--help] [options] --in <input-png-filename>`

var usageFormat string = `usage:  %s [--help] [options] --in <input-png-filename>

Generates an Ollama response from a given PNG image.

The prompt may be specified with  --prompt or the OLLAMATEA_PROMPT envvar.
The default prompt is:
  ` + defaultOllamaPrompt + `'.

Example:  $ ot-png-prompt --in hello.png -m llava

`

/////////////////////////////////////////////////////////////////////////////////////
// Simple BubbleTea model that does the inference and exits

type model struct {
	Session ollamatea.Session
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.Session.Init(),           // Session Init is required to be chained
		m.Session.StartGenerateMsg, // Kick off a generate
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ollamatea.GenerateResponseMsg:
		if msg.ID != m.Session.ID() {
			return m, nil // Ignore messages for other sessions
		}
		fmt.Fprintf(os.Stdout, msg.Response)
		return m, nil
	case ollamatea.GenerateDoneMsg:
		// Quit after the first message
		return m, tea.Quit
	}
	_, cmd := m.Session.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return ""
}

/////////////////////////////////////////////////////////////////////////////////////

func main() {
	var inputPNGFilename, outputTXTFilename string
	var ollamaHost, ollamaModel, ollamaPrompt string
	var verbose, showHelp bool

	pflag.StringVarP(&inputPNGFilename, "in", "i", "", "Input PNG filename ('-' is stdin)")
	pflag.StringVarP(&outputTXTFilename, "out", "o", "", "Output PNG filename")
	pflag.StringVarP(&ollamaHost, "host", "h", ollamatea.DefaultHost(), "Host for Ollama (also OLLAMATEA_HOST env)")
	pflag.StringVarP(&ollamaModel, "model", "m", ollamatea.DefaultModel(), "Model for Ollama (also OLLAMATEA_MODEL env)")
	pflag.StringVarP(&ollamaPrompt, "prompt", "p", "", "Prompt for Ollama (see --help for default)")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	pflag.BoolVarP(&showHelp, "help", "", false, "show help")
	pflag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stdout, usageFormat, os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}
	if len(inputPNGFilename) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: missing required argument: --out\n")
		fmt.Fprintf(os.Stderr, usageFormatShort, os.Args[0])
		os.Exit(1)
	}
	if len(ollamaPrompt) == 0 {
		ollamaPrompt = defaultOllamaPrompt
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "INFO: ohost=%s omodel=%s oprompt=\"%s\"\n", ollamaHost, ollamaModel, ollamaPrompt)
	}

	// Open input PNG file for reading, or use Stdin
	var err error
	infile := os.Stdin
	if len(inputPNGFilename) != 0 && inputPNGFilename != "-" {
		infile, err = os.OpenFile(inputPNGFilename, os.O_RDONLY, 0777)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to open input file %s\n", err.Error())
			os.Exit(1)
		}
		defer infile.Close()
	}

	// Capture file until EOF
	imageData, err := io.ReadAll(infile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read file %s\n", err.Error())
		os.Exit(1)
	}
	infile.Close() // we don't need it anymore

	// Use ollamatea.Session's machinery to convert to image
	s := ollamatea.NewSession()
	s.Host = ollamaHost
	s.Model = ollamaModel
	s.Prompt = ollamaPrompt
	s.Images = []ollamatea.ImageData{imageData}
	m := model{Session: s}

	_, err = tea.NewProgram(m, tea.WithInput(nil)).Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	// Write response
	outfile := os.Stdout
	if outputTXTFilename != "" && outputTXTFilename != "-" {
		outfile, err = os.OpenFile(outputTXTFilename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to open output file %s\n", err.Error())
			os.Exit(1)
		}
		defer outfile.Close()
	}

	_, err = outfile.Write([]byte(m.Session.Response()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write prompt %s\n", err.Error())
		os.Exit(1)
	}
	outfile.WriteString("\n")
}
