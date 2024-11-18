// OllamaTea Copyright (c) 2024 Neomantra Corp

package ollamatea

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	ollama "github.com/ollama/ollama/api"
)

//////////////////////////////////////////////////////////////////////////////

// Internal FetchListModel ID management. Ensures that messages are received
// only by components that sent them.
var lastFetchModelListID int64

// GetNextFetchModelListID atomically returns the next FetchModelList ID.
// Call this to get a unique ID for a [FetchModelList] request.
func GetNextModelChooserID() int64 {
	return atomic.AddInt64(&lastFetchModelListID, 1)
}

// Type alias in this package for convenience
type ListModelResponse = ollama.ListModelResponse

// FetchModelListResponseMsg is sent when a FetchModelList succeeds.
type FetchModelListResponseMsg struct {
	ID         int64               // ID of the original request
	OllamaHost string              // Ollama Host generating the response
	Models     []ListModelResponse // Models delivered
}

// FetchModelListErrorMsg is sent when a FetchModelList fails.
type FetchModelListErrorMsg struct {
	ID         int64  // ID of the original request
	OllamaHost string // Ollama Host generating the error
	Error      error  // Error returned
}

// FetchModelList fetches a list of models from the Ollama server and returns a [FetchListResponseMsg].
// If there is an error, a [FetchListErrorMsg] is returned.
//
// It is independent of any Model, so can be used as an independent [tea.Msg] generator
// to implement one's own model selection interfaces.
func FetchModelList(ollamaHost string, id int64) tea.Msg {
	ollamaURL, err := url.Parse(ollamaHost)
	if err != nil {
		return FetchModelListErrorMsg{ID: id, OllamaHost: ollamaHost, Error: err}
	}

	ollamaClient := ollama.NewClient(ollamaURL, http.DefaultClient)
	ctx := context.Background()
	listResponse, err := ollamaClient.List(ctx)
	if err != nil {
		return FetchModelListErrorMsg{ID: id, OllamaHost: ollamaHost, Error: err}
	}

	return FetchModelListResponseMsg{ID: id, OllamaHost: ollamaHost, Models: listResponse.Models}
}

//////////////////////////////////////////////////////////////////////////////

const (
	defaultModelChooserWaiting    = "Loading models..."
	defaultModelChooserMenuPrompt = "Select Ollama model"
)

var modelChooserExtraKeyBindings = []key.Binding{
	key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select")),
	key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "exit"),
	),
}

///////////////////////////////////////////////////////////////////////////////
// ollamatea.ModelChooser
//
// TODO: retry logic?? cancellation?

// ModelChooser is a Terminal UX for selecting a local LLM model from Ollama.
type ModelChooser struct {
	Waiting     string // Waiting to load message (default is "Loading models..")
	MenuPrompt  string // Menu prompt (default is "Select Ollama model")
	FetchOnInit bool   // FetchOnInit indicates whether to fetch the model list in Init (default: true)
	//Filter     string // Filter for model selection (default: none)

	modelList list.Model
	spinner   spinner.Model

	listedModels  []ListModelResponse
	selectedModel *ListModelResponse
	selectedName  string // Name of the selected model, for before we have a fetched list

	id         int64
	ollamaHost string // Ollama Host -- really the service's URL (default: OllamaTea default)
	isFetching bool
	lastError  error
}

// NewModelChooser returns a new ModelChooser for the given Ollama Host.
func NewModelChooser(ollamaHost string) ModelChooser {
	s := spinner.New()
	s.Spinner = spinner.Dot

	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = defaultModelChooserMenuPrompt
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return modelChooserExtraKeyBindings
	}
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return modelChooserExtraKeyBindings
	}

	return ModelChooser{
		id:           GetNextModelChooserID(),
		Waiting:      defaultModelChooserWaiting,
		MenuPrompt:   defaultModelChooserMenuPrompt,
		FetchOnInit:  true,
		selectedName: "",
		modelList:    l,
		spinner:      s,
		ollamaHost:   ollamaHost,
	}
}

// ID returns the ModelChooser unique ID.
func (m ModelChooser) ID() int64 {
	return m.id
}

// Host returns the Ollama Host URL for the ModelChooser.
func (m ModelChooser) Host() string {
	return m.ollamaHost
}

// LastError returns the last error encountered from fetching the model list.
// Returns nil if there is no error.
func (m ModelChooser) LastError() error {
	return m.lastError
}

// IsFetching returns true if the ModelChooser is fetching the model list.
func (m ModelChooser) IsFetching() bool {
	return m.isFetching
}

// SelectedModel returns the selected model from the ModelChooser.
// Returns nil if there is no selected model.
func (m ModelChooser) SelectedModel() *ollama.ListModelResponse {
	return m.selectedModel
}

// SetSelectionByName sets the selection by name.
// Returns true if nothing has been fetched yet.
// Otherwise, returns true if the model was found and set.
func (m *ModelChooser) SetSelectionByName(name string) bool {
	if len(m.listedModels) == 0 {
		m.selectedName = name
		return true
	}
	for i, listedModel := range m.listedModels {
		if listedModel.Name == name {
			m.selectedModel = &m.listedModels[i]
			m.modelList.Select(i)
			m.selectedName = name
			return true
		}
	}
	return false
}

// Styles returns the list.Styles for the ModelChooser.
func (m ModelChooser) Styles() list.Styles {
	return m.modelList.Styles
}

// SetStyles sets a list.Styles for the TUI.
// The Spinner is set to the list.Styles.Spinner
// Returns nil if there is no selected model.
func (m *ModelChooser) SetStyles(styles list.Styles) {
	m.spinner.Style = styles.Spinner
	m.modelList.Styles = styles
}

// Width returns the width of the model chooser
func (m ModelChooser) Width() int {
	return m.modelList.Width()
}

// SetWidth sets the width of the model chooser
func (m *ModelChooser) SetWidth(w int) {
	m.modelList.SetWidth(w)
}

// Height returns the height of the ModelChooser
func (m ModelChooser) Height() int {
	return m.modelList.Height()
}

func (m *ModelChooser) SetHeight(h int) {
	m.modelList.SetHeight(h)
}

//////////////////////////////////////////////////////////////////////////////

type ModelChooserSelectedMsg struct {
	ID         int64  // ID of the original request
	OllamaHost string // Ollama Host generating the list
	Selection  ollama.ListModelResponse
}

type ModelChooserAbortedMsg struct {
	ID    int64 // ID of the original request
	Error error // Error that caused the exit, if any
}

// fetchListMsg is sent to fetch the list of models from the Ollama server.
type fetchListMsg struct {
	ID         int64  // ID of the original request
	OllamaHost string // Ollama Host generating the response
}

// FetchListMsg is the message to send the ModelChooser to make
// it to fetch the list of models from the Ollama server.
func (m ModelChooser) FetchListMsg() fetchListMsg {
	return fetchListMsg{ID: m.id, OllamaHost: m.ollamaHost}
}

// startFetchingCmd returns a command to start fetching the model list.
func (m ModelChooser) startFetchingCmd() tea.Cmd {
	return func() tea.Msg {
		return FetchModelList(m.ollamaHost, m.id)
	}
}

//////////////////////////////////////////////////////////////////////////////

type modelChooserListItem struct {
	index int // index in selectedModels
	title string
	desc  string
}

func (i modelChooserListItem) Title() string       { return i.title }
func (i modelChooserListItem) Description() string { return i.desc }
func (i modelChooserListItem) FilterValue() string { return i.title }

func makeModelChooserListItem(index int, model ollama.ListModelResponse) modelChooserListItem {
	return modelChooserListItem{
		index: index,
		title: model.Name,
		desc: fmt.Sprintf("(%s) %s %s %s",
			humanize.Bytes(uint64(model.Size)),
			model.Details.Family,
			model.Details.ParameterSize,
			model.Details.QuantizationLevel,
		)}
}

//////////////////////////////////////////////////////////////////////////////
// BubbleTea interface

// Init handles the initialization of an Session
func (m ModelChooser) Init() tea.Cmd {
	// Fetch the list of models on the next Update
	if !m.FetchOnInit {
		return nil
	}
	return Cmdize(m.FetchListMsg())
}

// Update handles BubbleTea messages for the Session
// This is for starting/stopping/updating generation.
func (m ModelChooser) Update(msg tea.Msg) (ModelChooser, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchListMsg:
		if msg.ID != m.id {
			return m, nil
		}
		// TODO: cancel current
		m.isFetching = true
		return m, tea.Batch(m.startFetchingCmd(), m.spinner.Tick)

	case FetchModelListResponseMsg:
		if msg.ID != m.id {
			return m, nil
		}
		m.isFetching = false
		m.listedModels = msg.Models
		m.lastError = nil

		var items []list.Item
		selectedIndex := -1
		for i, model := range m.listedModels {
			items = append(items, makeModelChooserListItem(i, model))
			if (m.selectedModel != nil && model.Name == m.selectedModel.Name) ||
				(m.selectedName != "" && model.Name == m.selectedName) {
				selectedIndex = i
			}
		}
		if selectedIndex < 0 {
			m.selectedModel = nil
		} else {
			m.modelList.Select(selectedIndex)
			m.selectedName = m.listedModels[selectedIndex].Name
		}
		cmd := m.modelList.SetItems(items)
		return m, cmd

	case FetchModelListErrorMsg:
		if msg.ID != m.id {
			return m, nil
		}
		m.isFetching = false
		m.lastError = msg.Error
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "esc":
			return m, Cmdize(ModelChooserAbortedMsg{ID: m.id, Error: m.lastError})
		case "enter":
			item, ok := m.modelList.SelectedItem().(modelChooserListItem)
			if !ok {
				m.lastError = fmt.Errorf("bad cast -- report bug?")
				return m, nil
			}
			if item.index >= len(m.listedModels) {
				m.lastError = fmt.Errorf("bad index -- report bug?")
				return m, nil
			}
			m.selectedModel = &m.listedModels[item.index]
			return m, Cmdize(ModelChooserSelectedMsg{
				ID: m.id, OllamaHost: m.ollamaHost, Selection: *m.selectedModel})
		}
		var cmd tea.Cmd
		m.modelList, cmd = m.modelList.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.modelList.SetSize(msg.Width, msg.Height)
		return m, nil

	case spinner.TickMsg:
		if m.isFetching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.modelList, cmd = m.modelList.Update(msg)
	cmds = append(cmds, cmd)
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// View renders the ModelChooser's view.
func (m ModelChooser) View() string {
	if m.lastError != nil {
		return fmt.Sprintf("ERROR: %s", m.lastError.Error())
	} else if m.isFetching {
		return m.spinner.View() + " " + m.Waiting
	}
	if len(m.listedModels) == 0 {
		return "<empty>"
	}
	return m.modelList.View()
}
