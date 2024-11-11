// OllamaTea Copyright (c) 2024 Neomantra Corp
// ot-simplegen
//
// Simple Generate TUI using OllamaTeaGen
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

const defaultOllamaPrompt = "Describe this image for a visually impaired person"

var usageFormatShort string = `usage:  %s [--help] [options] --in <input-csv-filename>`

var usageFormat string = `usage:  %s [--help] [options] --in <input-csv-filename>
`

/////////////////////////////////////////////////////////////////////////////////////
// Style

// var defaultStyle = lipgloss.NewStyle().
// 	BorderStyle(lipgloss.NormalBorder()).
// 	BorderForeground(lipgloss.Color("63")) // purple

/////////////////////////////////////////////////////////////////////////////////////
// simpleGenModel

type simpleGenModel struct {
	Title     string // TODO: connect to chatPanel Viewport Title or something?
	chatPanel ollamatea.ChatPanelModel
}

func newSimpleGenModel() simpleGenModel {
	return simpleGenModel{
		Title:     "SimpleGen",
		chatPanel: ollamatea.NewChatPanel(ollamatea.NewSession()),
	}
}

func (m simpleGenModel) Init() tea.Cmd {
	return m.chatPanel.Init()
}

func (m simpleGenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c": // quit
			return m, tea.Quit
		}
	}

	m.chatPanel, cmd = m.chatPanel.Update(msg)
	return m, cmd
}

func (m simpleGenModel) View() string {
	return m.chatPanel.View()
}

/////////////////////////////////////////////////////////////////////////////////////

func main() {
	var ollamaHost, ollamaModel string
	var verbose, showHelp bool

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
	if verbose {
		fmt.Fprintf(os.Stderr, "INFO: ohost=%s omodel=%s\n", ollamaHost, ollamaModel)
	}

	// Create simpleGenModel and run the BubbleTea Program
	m := newSimpleGenModel()
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}
}
