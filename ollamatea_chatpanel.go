// OllamaTea Copyright (c) 2024 Neomantra Corp
// Portions adapted from:
//    https://github.com/charmbracelet/bubbletea/blob/master/examples/chat/main.go

package ollamatea

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
// ollamatea.ChatPanelKeyMap

// ChatPanelKeyMap is the all the [key.Binding] for the ChatPanelModel
type ChatPanelKeyMap struct {
	// Viewbox
	// CursorUp key.Binding
	// CursorDown key.Binding

	// InputBox resizing
	InputBoxUp   key.Binding
	InputBoxDown key.Binding

	ChooseModel key.Binding
	SendPrompt  key.Binding
}

// DefaultChatPanelKeyMap returns a default set of keybindings for ChatPanelModel
func DefaultChatPanelKeyMap() ChatPanelKeyMap {
	return ChatPanelKeyMap{
		InputBoxUp: key.NewBinding(
			key.WithKeys("shift+up"),
			key.WithHelp("shift+↑", "input up"),
		),
		InputBoxDown: key.NewBinding(
			key.WithKeys("shift+down"),
			key.WithHelp("shift+↓", "input down"),
		),
		SendPrompt: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send"),
		),
		ChooseModel: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "models"),
		),
	}
}

// FullHelp returns bindings to show the full help view.
// Implements bubble's [help.KeyMap] interface.
func (m *ChatPanelKeyMap) FullHelp() [][]key.Binding {
	kb := [][]key.Binding{{
		m.SendPrompt,
		m.ChooseModel,
		m.InputBoxUp,
		m.InputBoxDown,
	}}
	return kb
}

// ShortHelp returns bindings to show in the abbreviated help view. It's part
// of the help.KeyMap interface.
func (m ChatPanelKeyMap) ShortHelp() []key.Binding {
	kb := []key.Binding{
		m.SendPrompt,
		m.ChooseModel,
		m.InputBoxUp,
		m.InputBoxDown,
	}
	return kb
}

///////////////////////////////////////////////////////////////////////////////
// ollamatea.ChatPanelModel

// ollamatea.ChatPanelModel holds a simple Panel TUI for an Ollama chat
type ChatPanelModel struct {
	Title      string // Title of the ChatPanelModel, if any
	InputOnTop bool   // InputOnTop indicates whether the input box is at the top of screen

	Session *Session

	choosingModel bool

	showHelp bool
	help     help.Model
	KeyMap   ChatPanelKeyMap

	width       int // width of the ollamatea.ChatPanelModel
	height      int // height of the ollamatea.ChatPanelModel
	inputHeight int // inputheight of the Input Box, other heights derive from this

	spinner      spinner.Model  // spins while waiting for response
	inputText    textarea.Model // prompt input
	responseView viewport.Model // response view
	modelChooser ModelChooser
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
	inputText.Prompt = "│ "
	inputText.CharLimit = 300
	inputText.SetWidth(width)
	inputText.SetHeight(inputHeight)
	inputText.FocusedStyle.CursorLine = lipgloss.NewStyle()
	inputText.ShowLineNumbers = false
	inputText.KeyMap.InsertNewline.SetEnabled(false)

	responseView := viewport.New(width, responseHeight)
	responseView.SetContent(session.Response())

	chooser := NewModelChooser(session.Host)
	chooser.FetchOnInit = false

	m := ChatPanelModel{
		InputOnTop:    defaultInputOnTop,
		Session:       &session,
		choosingModel: false,
		KeyMap:        DefaultChatPanelKeyMap(),
		showHelp:      true,
		help:          help.New(),
		width:         width,
		height:        height,
		inputHeight:   inputHeight,
		spinner:       s,
		inputText:     inputText,
		responseView:  responseView,
		modelChooser:  chooser,
	}
	m.SetWidth(width)
	m.SetHeight(height)
	m.SetInputHeight(inputHeight)
	return m
}

// SetWidth sets the width of the ChatPanelModel
func (m *ChatPanelModel) SetWidth(w int) {
	m.width = w
	m.inputText.SetWidth(w)
	m.responseView.Width = w
	m.help.Width = w
	m.modelChooser.SetWidth(w)
}

// Width returns the width of the ChatPanelModel
func (m ChatPanelModel) Width() int {
	return m.width
}

// SetHeight sets the height of the ChatPanelModel
func (m *ChatPanelModel) SetHeight(height int) {
	m.height = height
	m.updateHeights()
}

// Height returns the height of the ChatPanelModel
func (m ChatPanelModel) Height() int {
	return m.height
}

// SetInputHeight sets the height of the input window.
// If inputHeight is less than 0, it is set to 0.
func (m *ChatPanelModel) SetInputHeight(inputHeight int) {
	if inputHeight < 0 {
		inputHeight = 0
	}
	if inputHeight != m.inputHeight {
		m.inputHeight = inputHeight
		m.updateHeights()
	}
}

// InputHeight returns the height of the input window
func (m ChatPanelModel) InputHeight() int {
	return m.inputHeight
}

// Placeholder gets the placeholder text for the input box
func (m ChatPanelModel) Placeholder() string {
	return m.inputText.Placeholder
}

// SetPlaceholder sets the placeholder text for the input box
func (m ChatPanelModel) SetPlaceholder(s string) {
	m.inputText.Placeholder = s
}

// GetShowHelp gets the ShowHelp setting value.
func (m ChatPanelModel) GetShowHelp() bool {
	return m.showHelp
}

// SetShowHelp sets whether to show help or not.
func (m *ChatPanelModel) SetShowHelp(showHelp bool) {
	m.showHelp = showHelp
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
		m.SetWidth(msg.Width)
		m.SetHeight(msg.Height)
		return m, nil
	case tea.KeyMsg:
		if m.choosingModel {
			m.modelChooser, cmd = m.modelChooser.Update(msg)
			return m, cmd
		}
		return m, m.handleChattingKeyMsg(msg)

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		m.inputText, cmd = m.inputText.Update(msg)
		return m, cmd

	case GenerateResponseMsg:
		var cmds []tea.Cmd
		_, cmd = m.Session.Update(msg)
		cmds = append(cmds, cmd)
		m.responseView.SetContent(m.Session.Response())
		m.responseView, cmd = m.responseView.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case ModelChooserAbortedMsg:
		if msg.ID == m.modelChooser.ID() {
			m.choosingModel = false
		}
		return m, nil

	case ModelChooserSelectedMsg:
		if msg.ID == m.modelChooser.ID() {
			m.choosingModel = false
			m.Session.Model = m.modelChooser.SelectedModel().Model
		}
		return m, nil

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
		m.modelChooser, cmd = m.modelChooser.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}
}

// View renders the ChatPanelModel's view.
func (m ChatPanelModel) View() string {
	if m.choosingModel {
		return m.modelChooser.View()
	}
	var respView string
	if m.Session.IsGenerating() {
		respView = m.spinner.View()
	}
	respView += m.responseView.View()
	var helpView string
	if m.showHelp {
		helpView = m.help.View(&m.KeyMap)
	}
	if m.InputOnTop {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.headerView(),
			m.inputText.View(),
			m.seperatorView(),
			respView,
			helpView,
		)
	} else {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.headerView(),
			respView,
			m.seperatorView(),
			m.inputText.View(),
			helpView,
		)
	}
}

func (m *ChatPanelModel) headerView() string {
	return "─ " + m.Title + " " + strings.Repeat("─", m.width-len(m.Title)-3) + "\n"
}

func (m *ChatPanelModel) seperatorView() string {
	modelLen := len(m.Session.Model)
	return "┌" + strings.Repeat("─", m.width-modelLen-1) + m.Session.Model + "\n"
}

// handleChatting for when a user is in chat mode
func (m *ChatPanelModel) handleChattingKeyMsg(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.InputBoxUp):
			if m.InputHeight() < m.height-2 { // TODO: chromeHeight := helpHeight+seperatorHegith+headerHegith
				m.SetInputHeight(m.InputHeight() + 1)
			}
			return nil

		case key.Matches(msg, m.KeyMap.InputBoxDown):
			if m.InputHeight() > 0 {
				m.SetInputHeight(m.InputHeight() - 1)
			}
			return nil

		case key.Matches(msg, m.KeyMap.SendPrompt):
			v := m.inputText.Value()
			if v == "" {
				// Don't send empty messages.
				return nil
			} else if m.Session.Prompt == v {
				// Don't repeat an unchanged prompt
				return nil
			}

			m.Session.Prompt = v
			m.Session.ClearResponse()
			m.responseView.SetContent("")
			return m.Session.StartGenerateMsg

		case key.Matches(msg, m.KeyMap.ChooseModel):
			m.choosingModel = true
			m.modelChooser.SetSelectionByName(m.Session.Model)
			return Cmdize(m.modelChooser.FetchListMsg())

		default:
			// Send all other keypresses to the textarea.
			var cmd tea.Cmd
			m.inputText, cmd = m.inputText.Update(msg)
			return cmd
		}
	}
	return tea.Batch(cmds...)
}

// updateHeights update the heights of objects
func (m *ChatPanelModel) updateHeights() {
	availHeight := m.height

	headerView := m.headerView()
	if headerView != "" {
		availHeight -= lipgloss.Height(headerView)
	}

	seperatorView := m.seperatorView()
	availHeight -= lipgloss.Height(seperatorView)

	if m.showHelp {
		helpView := m.help.View(&m.KeyMap)
		availHeight -= lipgloss.Height(helpView)
	}

	inputHeight := m.inputHeight
	if inputHeight >= availHeight {
		inputHeight = availHeight - 1
		if inputHeight < 0 {
			inputHeight = 0
		}
	}
	m.inputHeight = inputHeight
	m.inputText.SetHeight(inputHeight)

	responseHeight := availHeight - inputHeight
	if responseHeight < 0 {
		responseHeight = 0
	}
	m.responseView.Height = responseHeight

	m.modelChooser.SetHeight(m.height)
}
