// OllamaTea Copyright (c) 2024 Neomantra Corp
// ot-model-chooser
//
// Simple exerciser of ollamatea.ModelChooser
//

package main

import (
	"fmt"
	"os"

	"github.com/NimbleMarkets/ollamatea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"
)

/////////////////////////////////////////////////////////////////////////////////////

var usageFormat string = `usage:  %s [--help] [options]
Simple exercise of ollamatea.ModelChooser
`

/////////////////////////////////////////////////////////////////////////////////////
// simpleModelChooserModel
// The ridiculous name is intentional.

type simpleModelChooserModel struct {
	modelChooser   ollamatea.ModelChooser
	finalSelection *ollamatea.ListModelResponse
	lastError      error
}

func newSimpleModelChooserModel(ollamaHost string) simpleModelChooserModel {
	return simpleModelChooserModel{
		modelChooser: ollamatea.NewModelChooser(ollamaHost),
	}
}

func (m simpleModelChooserModel) Init() tea.Cmd {
	return m.modelChooser.Init()
}

func (m simpleModelChooserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c": // quit
			return m, tea.Quit
		}
	case ollamatea.ModelChooserSelectedMsg:
		m.finalSelection = &msg.Selection
		return m, tea.Quit
	case ollamatea.ModelChooserAbortedMsg:
		m.lastError = msg.Error
		return m, tea.Quit
	}

	m.modelChooser, cmd = m.modelChooser.Update(msg)
	return m, cmd
}

func (m simpleModelChooserModel) View() string {
	return m.modelChooser.View()
}

/////////////////////////////////////////////////////////////////////////////////////

func main() {
	var ollamaHost string
	var showHelp bool

	pflag.StringVarP(&ollamaHost, "host", "h", ollamatea.DefaultHost(), "Host for Ollama (also OLLAMATEA_HOST env)")
	pflag.BoolVarP(&showHelp, "help", "", false, "show help")
	pflag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stdout, usageFormat, os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}

	// Create simpleChooserModel and run the BubbleTea Program
	m := newSimpleModelChooserModel(ollamaHost)
	model, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	m = model.(simpleModelChooserModel)

	if m.lastError != nil {
		fmt.Fprintf(os.Stderr, "Chooser Aborted: %s\n", m.lastError.Error())
		os.Exit(1)
	}
	if m.finalSelection == nil {
		fmt.Fprintf(os.Stderr, "No selection\n")
	} else {
		fmt.Fprintf(os.Stdout, "Selected:   %s  %s\n", m.finalSelection.Name, m.finalSelection.Digest)
	}
}
