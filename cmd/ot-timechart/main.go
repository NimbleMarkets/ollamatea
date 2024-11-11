// OllamaTea Copyright (c) 2024 Neomantra Corp
// ot-timechart
//
// Adapted from ntcharts-ohlc:
//   https://github.com/NimbleMarkets/ntcharts/tree/main/cmd/ntcharts-ohlc

package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	tslc "github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/NimbleMarkets/ollamatea"
	"github.com/ollama/ollama/api"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/pflag"
)

/////////////////////////////////////////////////////////////////////////////////////

const defaultOllamaPrompt = "Describe this image for a visually impaired person"

var usageFormatShort string = `usage:  %s [--help] [options] --in <input-csv-filename>`

var usageFormat string = `usage:  %s [--help] [options] --in <input-csv-filename>

A mini-TUI for generating an Ollama response from a simple CSV file.
The CSV file should have a header row with the first column being the time.

The prompt may be specified with  --prompt or the OLLAMATEA_PROMPT envvar.
The default prompt is:
  ` + defaultOllamaPrompt + `'.

See https://github.com/NimbleMarkets/ollamatea/tree/main/cmd/ot-timechart

`

const inputTextPlaceholder = "Prompt about the chart..."

/////////////////////////////////////////////////////////////////////////////////////
// Style

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

var labelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6")) // cyan

/////////////////////////////////////////////////////////////////////////////////////
// timechartModel

// timechartModel is the primary BubbleTea model for the timechart TUI
type timechartModel struct {
	chart     tslc.Model
	chatPanel ollamatea.ChatPanelModel

	Title      string
	UseBraille bool
}

func newTimechartModel(timePoints []tslc.TimePoint) timechartModel {
	otSession := ollamatea.NewSession()
	otSession.Prompt = defaultOllamaPrompt

	m := timechartModel{
		chart: tslc.New(20, 10,
			tslc.WithAxesStyles(axisStyle, labelStyle),
		),
		chatPanel: ollamatea.NewChatPanel(otSession),
	}
	m.chart.Focus()
	minX, maxX := int64(math.MaxInt64), int64(math.MinInt64)
	minY, maxY := math.MaxFloat64, -math.MaxFloat64
	for _, tp := range timePoints {
		sec := tp.Time.Unix()
		if sec < minX {
			minX = sec
		}
		if sec > maxX {
			maxX = sec
		}
		if tp.Value < minY {
			minY = tp.Value
		}
		if tp.Value > maxY {
			maxY = tp.Value
		}
		m.chart.Push(tp)
	}
	m.chart.SetViewTimeAndYRange(time.Unix(minX, 0), time.Unix(maxX, 0), minY, maxY)
	m.chart.UpdateGraphSizes()
	m.chatPanel = m.chatPanel.SetPlaceholder(inputTextPlaceholder)
	return m
}

func (m timechartModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.chart.Init(),
		m.chatPanel.Init(),
	}
	return tea.Sequence(cmds...)
}

func (m timechartModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// TODO: moderate chatPanel width?
		m.chatPanel = m.chatPanel.SetHeight(msg.Height - 1)

		// chat window has a constant width and chart size fills rest
		chartWidth := msg.Width - m.chatPanel.Width - 2 // 2 for padding
		chartHeight := msg.Height - 3
		m.chart.Resize(chartWidth, chartHeight)

		// choose which rune drawing method to use based on user options
		switch {
		case m.UseBraille:
			m.chart.DrawBrailleAll()
		default:
			m.chart.DrawAll()
		}

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case ollamatea.StartGenerateMsg:
		// Before we start generating,  conver the chart to an image
		view := m.Title + m.chart.View()
		pngBytes, err := ollamatea.ConvertTerminalTextToImage(view, nil)
		if err != nil {
			// TODO: how to communicate error to user?
			return m, nil
		}
		m.chatPanel.Session.Images = []api.ImageData{pngBytes}
	case ollamatea.GenerateDoneMsg:
		// When done, maintain the Ollama conversation's Context
		m.chatPanel.Session.Context = msg.Context
		return m, nil
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.chatPanel, cmd = m.chatPanel.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.chart, cmd = m.chart.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m timechartModel) View() string {
	chartView := m.chart.View()
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		defaultStyle.Render(m.Title+chartView),
		m.chatPanel.View())
}

/////////////////////////////////////////////////////////////////////////////////////

// resetTimeRange set displayed time range such that each graph column is a single day
func (m *timechartModel) resetTimeRange() {
	viewMin := time.Unix(int64(m.chart.ViewMinX()), 0)
	viewMax := viewMin.Add(time.Hour * time.Duration(24*m.chart.GraphWidth()))
	if viewMax.Unix() > int64(m.chart.MaxX()) {
		viewMax = time.Unix(int64(m.chart.MaxX()), 0)
	}
	m.chart.SetViewTimeRange(viewMin, viewMax)
}

/////////////////////////////////////////////////////////////////////////////////////

func main() {
	var inputCSVFilename string
	var inputIsZstd, useBraille bool
	var ollamaHost, ollamaModel, ollamaPrompt string
	var chartTitle string
	var verbose, showHelp bool

	pflag.StringVarP(&inputCSVFilename, "in", "i", "", "Input CSV filename ('-' is stdin)")
	pflag.StringVarP(&ollamaHost, "host", "h", ollamatea.DefaultHost(), "Host for Ollama (also OLLAMATEA_HOST env)")
	pflag.StringVarP(&ollamaModel, "model", "m", ollamatea.DefaultModel(), "Model for Ollama (also OLLAMATEA_MODEL env)")
	pflag.StringVarP(&ollamaPrompt, "prompt", "p", "", "Prompt for Ollama (see --help for default)")
	pflag.StringVarP(&chartTitle, "title", "t", "", "Title for the chart")
	pflag.BoolVarP(&inputIsZstd, "zstd", "z", false, "Input is ZSTD compressed (otherwise uses filename ending in .zst or zstd)")
	pflag.BoolVar(&useBraille, "braille", false, "use braille lines (default: arc lines)")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	pflag.BoolVarP(&showHelp, "help", "", false, "show help")
	pflag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stdout, usageFormat, os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}
	if len(inputCSVFilename) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: missing required argument: --in\n")
		fmt.Fprintf(os.Stderr, usageFormatShort, os.Args[0])
		os.Exit(1)
	}
	if len(ollamaPrompt) == 0 {
		ollamaPrompt = defaultOllamaPrompt
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "INFO: ohost=%s omodel=%s oprompt=\"%s\"\n", ollamaHost, ollamaModel, ollamaPrompt)
	}

	// Read the CSV file and build the dataset
	fileReader, fileCloser, err := makeCompressedReader(inputCSVFilename, inputIsZstd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	defer fileCloser.Close()

	records, err := recordsFromCSV(fileReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}

	// Create timechartModel and run the BubbleTea Program
	m := newTimechartModel(records)
	m.Title = chartTitle + "\n"
	m.UseBraille = useBraille

	_, err = tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

/////////////////////////////////////////////////////////////////////////////////////

// recordsFromCSV reads from a io.Reader and returns
// a slice of timechartRecord objects
func recordsFromCSV(r io.Reader) ([]tslc.TimePoint, error) {
	var records []tslc.TimePoint
	firstRow := true
	csvReader := csv.NewReader(r)
	for {
		cols, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return records, err
		}
		if len(cols) < 2 {
			return records, errors.New("not enough columns in CSV record")
		}
		if firstRow {
			firstRow = false
			if cols[0][0] <= 0 || cols[0][0] >= '9' {
				// skip header row
				continue
			}
		}
		newRecord, err := newRecord(cols)
		if err == nil {
			records = append(records, newRecord)
		}
	}
	return records, nil
}

// NewRecord returns a record from a given []string
// containing OHLC record information
// Function expects the following columns:
// [Date, Value]   with Date as "2006-01-02" or a epoch value
func newRecord(cols []string) (tslc.TimePoint, error) {
	rec := tslc.TimePoint{}
	if len(cols) < 2 {
		return rec, errors.New("not enough columns in CSV record")
	}
	var err error
	rec.Time, err = strToDate(cols[0])
	if err != nil {
		return rec, err
	}
	rec.Value, err = strconv.ParseFloat(cols[1], 64)
	if err != nil {
		return rec, fmt.Errorf("bad float: '%s' %v", cols[1], err)
	}
	return rec, nil
}

func strToDate(str string) (time.Time, error) {
	// First try to extract as YYYY-MM-DD
	d, err := time.Parse("2006-01-02", str)
	if err == nil {
		return d, nil
	}
	// Otherwise, grab as an epoch value and detect unit
	epoch, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("bad date: '%s' %v", str, err)
	}
	const biggestSeconds = 5000000000
	if epoch < biggestSeconds { // assume seconds
		return time.Unix(epoch, 0), nil
	} else if epoch < biggestSeconds*1000 { // assume milliseconds
		return time.UnixMilli(epoch), nil
	} else if epoch < biggestSeconds*1000*1000 { // assume microseconds
		return time.UnixMicro(epoch), nil
	} else { // assume nanoseconds
		return time.Unix(0, epoch), nil
	}
}

type nullCloser struct{}

func (nullCloser) Close() error { return nil }

// makeCompressedReader returns a io.Reader for the given filename, or os.Stdout if filename is "-".
// If isGZ is true or the filename ends in ".gz", the writer will gzip the output.
//
// https://gist.github.com/neomantra/691a6028cdf2ac3fc6ec97d00e8ea802
func makeCompressedReader(filename string, isZstd bool) (io.Reader, io.Closer, error) {
	var reader io.Reader
	var closer io.Closer

	if filename != "-" {
		if file, err := os.Open(filename); err == nil {
			reader, closer = file, file
		} else {
			return nil, nil, err
		}
	} else {
		reader, closer = os.Stdin, nullCloser{}
	}

	var err error
	if isZstd || strings.HasSuffix(filename, ".zst") || strings.HasSuffix(filename, ".zstd") {
		reader, err = zstd.NewReader(reader)
	}

	if err != nil {
		// clean up file
		if closer != nil {
			closer.Close()
		}
		return nil, nil, err
	}
	return reader, closer, nil
}
