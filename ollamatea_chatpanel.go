// OllamaTea Copyright (c) 2024 Neomantra Corp
// Portions adapted from:
//    https://github.com/charmbracelet/bubbletea/blob/master/examples/chat/main.go

package ollamatea

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultChatWidth   = 40
	defaultChatHeight  = 20
	defaultInputHeight = 4
	defaultInputOnTop  = false
)

///////////////////////////////////////////////////////////////////////////////
// ollamatea.ChatPanelModel

// ollamatea.ChatPanelModel holds a simple Panel TUI for an Ollama chat
type ChatPanelModel struct {
	Width       int  // Width is the width of the ollamatea.ChatPanelModel
	Height      int  // Height is the height of the ollamatea.ChatPanelModel
	InputHeight int  // Height of the Input Box, other heights derive from this
	InputOnTop  bool // InputOnTop indicates whether the input box is at the top of screen

	Session *Session

	spinner      spinner.Model  // spins while waiting for response
	inputText    textarea.Model // prompt input
	responseView viewport.Model // response view
}

func NewChatPanel(session Session) ChatPanelModel {
	width := defaultChatWidth
	height := defaultChatHeight
	inputHeight := defaultInputHeight
	responseHeight := defaultChatHeight - inputHeight

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	inputText := textarea.New()
	inputText.Placeholder = "Enter your prompt here..."
	inputText.Focus()
	inputText.Prompt = "| "
	inputText.CharLimit = 300
	inputText.SetWidth(width)
	inputText.SetHeight(inputHeight)
	inputText.FocusedStyle.CursorLine = lipgloss.NewStyle()
	inputText.ShowLineNumbers = false
	inputText.KeyMap.InsertNewline.SetEnabled(false)

	responseView := viewport.New(width, responseHeight)
	responseView.SetContent(session.Response())

	return ChatPanelModel{
		Width:        width,
		Height:       height,
		InputHeight:  inputHeight,
		InputOnTop:   defaultInputOnTop,
		Session:      &session,
		spinner:      s,
		inputText:    inputText,
		responseView: responseView,
	}
}

func (m ChatPanelModel) SetWidth(w int) ChatPanelModel {
	m.Width = w
	m.inputText.SetWidth(w)
	m.responseView.Width = w
	return m
}

func (m ChatPanelModel) SetHeight(height int) ChatPanelModel {
	inputHeight := m.InputHeight
	if m.InputHeight >= height {
		m.InputHeight = height - 1
		if m.InputHeight < 0 {
			m.InputHeight = 0
		}
	}

	m.Height = height
	m.InputHeight = inputHeight
	m.inputText.SetHeight(m.InputHeight)
	m.responseView.Height = height - m.InputHeight
	return m
}

// SetInputHeight sets the height of the input window.  This is clamped to [0,Height)
func (m ChatPanelModel) SetInputHeight(inputHeight int) ChatPanelModel {
	if inputHeight >= m.Height {
		inputHeight = m.Height - 1
	}
	if inputHeight < 0 {
		inputHeight = 0
	}
	m.InputHeight = inputHeight

	m.inputText.SetHeight(inputHeight)
	m.responseView.Height = m.Height - inputHeight
	return m
}

func (m ChatPanelModel) GetInputHeight() int {
	return m.InputHeight
}

func (m ChatPanelModel) GetPlaceholder() string {
	return m.inputText.Placeholder
}

func (m ChatPanelModel) SetPlaceholder(s string) ChatPanelModel {
	m.inputText.Placeholder = s
	return m
}

//////////////////////////////////////////////////////////////////////////////
// BubbleTea handling

// Init handles the initialization of an ChatPanelModel
func (m ChatPanelModel) Init() tea.Cmd {
	sessionCmd := m.Session.Init()
	return tea.Batch(textarea.Blink, m.spinner.Tick, sessionCmd)
}

// Update handles BubbleTea messages for the ChatPanelModel
func (m ChatPanelModel) Update(msg tea.Msg) (ChatPanelModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m = m.SetWidth(msg.Width)
		m = m.SetHeight(msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			v := m.inputText.Value()
			if v == "" {
				// Don't send empty messages.
				return m, nil
			} else if m.Session.Prompt == v {
				// Don't repeat an unchanged prompt
				return m, nil
			}

			m.Session.Prompt = v
			m.Session.ClearResponse()
			m.responseView.SetContent("")
			return m, m.Session.StartGenerateMsg
		case "shift+up":
			m = m.SetInputHeight(m.GetInputHeight() + 1)
			return m, nil
		case "shift+down":
			m = m.SetInputHeight(m.GetInputHeight() - 1)
			return m, nil
		default:
			// Send all other keypresses to the textarea.
			m.inputText, cmd = m.inputText.Update(msg)
			return m, cmd
		}

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		m.inputText, cmd = m.inputText.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case GenerateResponseMsg:
		var cmds []tea.Cmd
		_, cmd = m.Session.Update(msg)
		cmds = append(cmds, cmd)
		m.responseView.SetContent(m.Session.Response())
		m.responseView, cmd = m.responseView.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	default:
		var cmds []tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		_, cmd = m.Session.Update(msg)
		cmds = append(cmds, cmd)
		m.responseView, cmd = m.responseView.Update(msg)
		cmds = append(cmds, cmd)
		m.inputText, cmd = m.inputText.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
}

// View renders the ChatPanelModel's view.
func (m ChatPanelModel) View() string {
	var respView string
	if m.Session.IsGenerating() {
		respView = m.spinner.View()
	}
	respView += m.responseView.View()
	if m.InputOnTop {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.inputText.View(),
			respView,
		)
	} else {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			respView,
			m.inputText.View(),
		)
	}
}
