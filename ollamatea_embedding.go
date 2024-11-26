// OllamaTea Copyright (c) 2024 Neomantra Corp

package ollamatea

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	ollama "github.com/ollama/ollama/api"
)

//////////////////////////////////////////////////////////////////////////////
// BubbleTea messages

type StartEmbedMsg struct {
	ID int64 // ID is the session ID to start
}

type StopEmbedMsg struct {
	ID int64 // ID is the session ID to stop
}

// EmbedResponseMsg is the message generated each time there is a reply from Ollama.
// The information contained is only partial.
// To check what has been received so far in the request, check [Session.Response()]
// To focus solely on full responses, listen for GenerateDoneMsg.
type EmbedResponseMsg struct {
	ID        int64                // ID is the generation session ID corresponding to the Response
	CreatedAt time.Time            // CreatedAt is the timestamp of the response.
	Response  ollama.EmbedResponse // Response is the EmbedResponse from Ollama

}

// EmbedErrorMsg is the message generated when the generation is complete.
// It contains the complete response along with [Context], which may be set on
// [Session.Context] to carry on the conversation.
type EmbedErrorMsg struct {
	ID        int64     // ID is the generation session ID corresponding to the Response
	CreatedAt time.Time // CreatedAt is the timestamp of the response.
	Error     error     // Error is the reason the model stopped generating text.
}

///////////////////////////////////////////////////////////////////////////////

// EmbedSession holds the data for an OllamaTea Embed, both its request and response
// See https://github.com/ollama/ollama/blob/main/api/types.go#L248
type EmbedSession struct {
	Host  string // Ollama Host -- really the service's URL
	Model string // Ollama LLM model.  See https://ollama.com/library

	Options map[string]interface{} // Options lists model-specific options.

	Input     any            // Input is the input to embed.
	KeepAlive *time.Duration // KeepAlive controls how long the model will stay loaded in memory following this request.
	Truncate  *bool          // Truncate the end of each input to fit within context length

	// Private
	ctx        context.Context
	cancelFunc context.CancelFunc
	id         int64 // Unique Session ID
	lastError  error // Last error

	isEmbedding bool                  // Currently inferencing? Only one per session
	response    *ollama.EmbedResponse // Ollama embed response
}

// NewEmbedSession returns a new Session with the default values.
func NewEmbedSession(opts ...EmbedOption) EmbedSession {
	s := EmbedSession{
		Host:        DefaultHost(),
		Model:       DefaultModel(),
		Input:       nil,
		id:          nextSessionID(),
		isEmbedding: false,
	}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// EmbedOption is a functional option for configuring a EmbedSession.
// Various With* functions, such as WitHost, are available to set specific fields.
type EmbedOption func(*EmbedSession)

// WithHost is an EmbedOption to set the Host field.
func WithHost(host string) EmbedOption {
	return func(s *EmbedSession) {
		s.Host = host
	}
}

// WithModel is an EmbedOption to set the Model field.
func WithModel(model string) EmbedOption {
	return func(s *EmbedSession) {
		s.Model = model
	}
}

// WithInput is an EmbedOption to set the Input field.
func WithInput(input any) EmbedOption {
	return func(s *EmbedSession) {
		s.Input = input
	}
}

// WithKeepAlive is an EmbedOption to set the Duration of the KeepAlive field.
func WithKeepAlive(d time.Duration) EmbedOption {
	return func(s *EmbedSession) {
		if d == 0 {
			s.KeepAlive = nil
		} else {
			s.KeepAlive = &d
		}
	}
}

// WithTruncate is an EmbedOption to indicate truncation.
func WithTruncate(trunc bool) EmbedOption {
	return func(s *EmbedSession) {
		s.Truncate = &trunc
	}
}

// ID returns the ID of the EmbedSession
func (s *EmbedSession) ID() int64 {
	return s.id
}

// IsEmbedding returns whether the EmbedSession is currently embedding
func (s *EmbedSession) IsEmbedding() bool {
	return s.isEmbedding
}

// Response returns the last EmbedResponse, if any.
func (s *EmbedSession) Response() *ollama.EmbedResponse {
	return s.response
}

// Error returns the last error, if any
func (s *EmbedSession) Error() error {
	return s.lastError
}

// ClearReponse clears the current response
func (s *EmbedSession) ClearResponse() {
	s.response = nil
}

// ClearError clears the current error
func (s *EmbedSession) ClearError() {
	s.lastError = nil
}

// StartEmbedMsg returns a StartEmbedMsg for the EmbedSession
func (s *EmbedSession) StartEmbedMsg() tea.Msg {
	return StartEmbedMsg{ID: s.id}
}

// StartEmbedCmd returns a command to start emebedding  for the EmbedSession
func (s *EmbedSession) StartEmbedCmd() tea.Cmd {
	return func() tea.Msg {
		return StartEmbedMsg{ID: s.id}
	}
}

//////////////////////////////////////////////////////////////////////////////
// BubbleTea interface

// Init handles the initialization of an EmbedSession
// Currently does nothing
func (m *EmbedSession) Init() tea.Cmd {
	return nil
}

// Update handles BubbleTea messages for the EmbedSession
// This is for starting/stopping/updating generation.
func (m *EmbedSession) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StartEmbedMsg:
		if msg.ID != m.id {
			return m, nil
		}
		if m.isEmbedding {
			// Cancel current inference
			if m.cancelFunc != nil {
				m.cancelFunc()
				m.cancelFunc = nil
			}
			m.ctx = nil
			m.isEmbedding = false
		}
		return m, m.startEmbeddingCmd()

	case StopEmbedMsg:
		if msg.ID != m.id {
			return m, nil
		}
		if m.cancelFunc != nil {
			m.cancelFunc()
			m.cancelFunc = nil
		}
		m.ctx = nil
		m.isEmbedding = false
		return m, nil

	case EmbedResponseMsg:
		m.response = &msg.Response
		m.lastError = nil
		return m, nil

	case EmbedErrorMsg:
		m.response = nil
		m.lastError = msg.Error
		return m, nil
	}
	return m, nil
}

// View renders the Sessions's view.
// This is will either be an error message, a "..." waiting string, or the Ollama response.
// We often set up other components for the TUI chrome and ignore this View.
func (m *EmbedSession) View() string {
	if m.lastError != nil {
		return fmt.Sprintf("ERROR: %s", m.lastError.Error())
	}
	return "" // m.Response()
}

//////////////////////////////////////////////////////////////////////////////

// startEmbeddingCmd is a tea.Msg wrapper for startEmbedding
func (s *EmbedSession) startEmbeddingCmd() tea.Cmd {
	return func() tea.Msg {
		return s.startEmbedding()
	}
}

// startEmbedding starts embedding for a Session
// Performs the actual Ollama /embed call
func (s *EmbedSession) startEmbedding() tea.Msg {
	if s.isEmbedding {
		return nil
	}
	s.isEmbedding = true
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())

	ollamaURL, err := url.Parse(s.Host)
	if err != nil {
		s.lastError = err
		s.isEmbedding = false
		return makeEmbedErrorMsg(s.id, err)
	}

	ollamaClient := ollama.NewClient(ollamaURL, http.DefaultClient)
	req := &ollama.EmbedRequest{
		Model: s.Model,
		Input: s.Input,
		// TODO KeepAlive: &ollama.Duration{},
		// TODO Truncate:  new(bool),
		Options: s.Options,
	}

	resp, err := ollamaClient.Embed(s.ctx, req)
	if err != nil {
		s.lastError = err
		return makeEmbedErrorMsg(s.id, err)
	}

	return makeEmbedResponseMsg(s.id, resp)
}

//////////////////////////////////////////////////////////////////////////////

func makeEmbedResponseMsg(id int64, resp *ollama.EmbedResponse) tea.Msg {
	return EmbedResponseMsg{
		ID:        id,
		CreatedAt: time.Now(),
		Response:  *resp,
	}
}

func makeEmbedErrorMsg(id int64, err error) tea.Msg {
	return EmbedErrorMsg{
		ID:        id,
		CreatedAt: time.Now(),
		Error:     err,
	}
}
