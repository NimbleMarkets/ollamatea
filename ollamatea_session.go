// OllamaTea Copyright (c) 2024 Neomantra Corp

package ollamatea

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	ollama "github.com/ollama/ollama/api"
)

//////////////////////////////////////////////////////////////////////////////
// BubbleTea messages

type StartGenerateMsg struct {
	ID int64 // ID is the session ID to start
}

type StopGenerateMsg struct {
	ID int64 // ID is the session ID to stop
}

// generateResponseMsg is the private message dispatched repeatedly by waitForResponse
// Its handler dispatches the public GenerateResponseMsg and GenerateDoneMsg messages
type generateResponseMsg struct {
	ID        int64     // ID is the generation session ID corresponding to the Response
	CreatedAt time.Time // CreatedAt is the timestamp of the response.
	Response  string    // Response is the textual response itself.

	Done       bool   // Done is true if this is the last response for the generation
	DoneReason string // DoneReason is the reason the model stopped generating text.
	// Context is an encoding of the conversation used in this response; this
	// can be sent in the next request to keep a conversational memory.
	Context []int
}

// GenerateResponseMsg is the message generated each time there is a reply from Ollama.
// The information contained is only partial.
// To check what has been received so far in the request, check [Session.Response()]
// To focus solely on full responses, listen for GenerateDoneMsg.
type GenerateResponseMsg struct {
	ID        int64     // ID is the generation session ID corresponding to the Response
	CreatedAt time.Time // CreatedAt is the timestamp of the response.

	// Response is the textual response in this specific call.
	// Use [GenerateDoneMsg] or [Session.Response()] for fuller responses.
	Response string
}

// GenerateDoneMsg is the message generated when the generation is complete.
// It contains the complete response along with [Context], which may be set on
// [Session.Context] to carry on the conversation.
type GenerateDoneMsg struct {
	ID         int64     // ID is the generation session ID corresponding to the Response
	Response   string    // Full resposne from the Ollama generation
	CreatedAt  time.Time // CreatedAt is the timestamp of the response.
	DoneReason string    // DoneReason is the reason the model stopped generating text.
	// Context is an encoding of the conversation used in this response; this
	// can be sent in the next request to keep a conversational memory.
	Context []int
}

//////////////////////////////////////////////////////////////////////////////

// Internal Session ID management. Ensure that messages are received
// only by components that sent them.
var lastSessionID int64

func nextSessionID() int64 {
	return atomic.AddInt64(&lastSessionID, 1)
}

// Type alias in this package for convenience
type ImageData = ollama.ImageData

//////////////////////////////////////////////////////////////////////////////

// Session holds the data for an OllamaTea Generate, both its request and built response
// See https://github.com/ollama/ollama/blob/main/api/types.go#L42
type Session struct {
	Host     string // Ollama Host -- really the service's URL
	Model    string // Ollama LLM model.  See https://ollama.com/library
	System   string // Ollama System prompt
	Template string // Ollama System prompt
	Context  []int  // Ollama Context

	Prompt  string                 // Ollama Prompt
	Suffix  string                 // Ollama Prompt Suffix
	Images  []ImageData            // List of base64-encoded images
	Options map[string]interface{} // Options lists model-specific options

	// Private
	ctx        context.Context
	cancelFunc context.CancelFunc
	id         int64 // Unique Session ID
	lastError  error // Last error

	isGenerating bool                     // Currently inferencing? Only one per session
	respCh       chan generateResponseMsg // Channel for responses message dispatch
	response     string                   // Ollama response
}

// NewSession returns a new Session with the default values.
func NewSession() Session {
	return Session{
		Host:         DefaultHost(),
		Model:        DefaultModel(),
		Prompt:       DefaultPrompt(),
		System:       DefaultSystemPrompt(),
		id:           nextSessionID(),
		isGenerating: false,
		respCh:       make(chan generateResponseMsg, 100),
	}
}

func (s *Session) ID() int64 {
	return s.id
}

func (s *Session) IsGenerating() bool {
	return s.isGenerating
}

func (s *Session) Response() string {
	return s.response
}

func (s *Session) Error() error {
	return s.lastError
}

func (s *Session) ClearResponse() {
	s.response = ""
}

func (s *Session) ClearError() {
	s.lastError = nil
}

func (s *Session) StartGenerateMsg() tea.Msg {
	return StartGenerateMsg{ID: s.id}
}

//////////////////////////////////////////////////////////////////////////////
// BubbleTea interface

// Init handles the initialization of an Session
func (m *Session) Init() tea.Cmd {
	return waitForResponse(m.respCh) // start the response listener
}

// Update handles BubbleTea messages for the Session
// This is for starting/stopping/updating generation.
func (m *Session) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StartGenerateMsg:
		if msg.ID != m.id {
			return m, nil
		}
		if m.isGenerating {
			// Cancel current inference
			if m.cancelFunc != nil {
				m.cancelFunc()
				m.cancelFunc = nil
			}
			m.ctx = nil
			m.isGenerating = false
			// TODO: done message send?
		}
		return m, m.startGeneratingCmd()

	case StopGenerateMsg:
		if msg.ID != m.id {
			return m, nil
		}
		if m.cancelFunc != nil {
			m.cancelFunc()
			m.cancelFunc = nil
		}
		m.ctx = nil
		m.isGenerating = false
		// TODO: done message send?
		return m, nil

	case generateResponseMsg:
		if msg.ID != m.id {
			return m, nil
		}
		// TODO: string builder
		m.response = m.response + msg.Response

		respMsg := GenerateResponseMsg{
			ID:        m.id,
			CreatedAt: msg.CreatedAt,
			Response:  msg.Response,
		}

		if !msg.Done {
			return m, tea.Batch(Cmdize(respMsg), waitForResponse(m.respCh))
		}

		// We are done generating
		m.isGenerating = false
		doneMsg := GenerateDoneMsg{
			ID:         m.id,
			CreatedAt:  msg.CreatedAt,
			DoneReason: msg.DoneReason,
			Response:   m.response,
			Context:    msg.Context,
		}

		return m, tea.Sequence(
			Cmdize(respMsg),
			Cmdize(doneMsg),
			waitForResponse(m.respCh),
		)
	}
	return m, nil
}

// View renders the Sessions's view.
// This is will either be an error message, a "..." waiting string, or the Ollama response.
// We often set up other components for the TUI chrome and ignore this View.
func (m *Session) View() string {
	if m.lastError != nil {
		return fmt.Sprintf("ERROR: %s", m.lastError.Error())
	}
	return m.Response()
}

//////////////////////////////////////////////////////////////////////////////

// startGenerating
func (m *Session) startGeneratingCmd() tea.Cmd {
	return func() tea.Msg {
		return m.startGenerating()
	}
}

func (m *Session) startGenerating() tea.Msg {
	if m.isGenerating {
		return nil
	}
	m.isGenerating = true
	m.ctx, m.cancelFunc = context.WithCancel(context.Background())

	ollamaURL, err := url.Parse(m.Host)
	if err != nil {
		m.lastError = err
		m.isGenerating = false
		return Cmdize(makeGenerateDoneErrorMsg(m.id, err))
	}

	ollamaClient := ollama.NewClient(ollamaURL, http.DefaultClient)
	req := &ollama.GenerateRequest{
		Model:    m.Model,
		Prompt:   m.Prompt,
		Suffix:   m.Suffix,
		System:   m.System,
		Template: m.Template,
		Context:  m.Context,
		Options:  m.Options,
		Images:   m.Images,
	}

	respFunc := func(resp ollama.GenerateResponse) error {
		m.respCh <- generateResponseMsg{
			ID:         m.id,
			CreatedAt:  resp.CreatedAt,
			Response:   resp.Response,
			Done:       resp.Done,
			DoneReason: resp.DoneReason,
			Context:    resp.Context,
		}
		return nil
	}

	err = ollamaClient.Generate(m.ctx, req, respFunc)
	if err != nil {
		m.lastError = err
		return Cmdize(makeGenerateDoneErrorMsg(m.id, err))
	}
	return nil
}

func makeGenerateDoneErrorMsg(id int64, err error) tea.Msg {
	return GenerateDoneMsg{
		ID:         id,
		Response:   "",
		CreatedAt:  time.Now(),
		DoneReason: err.Error(),
		Context:    nil,
	}
}

//////////////////////////////////////////////////////////////////////////////

// A command that waits for the responses on the channel
func waitForResponse(sub chan generateResponseMsg) tea.Cmd {
	return func() tea.Msg {
		return generateResponseMsg(<-sub)
	}
}
