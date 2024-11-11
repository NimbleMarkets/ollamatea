// OllamaTea Copyright (c) 2024 Neomantra Corp
// ot-ansi-to-png

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/NimbleMarkets/ollamatea"
	"github.com/spf13/pflag"
)

/////////////////////////////////////////////////////////////////////////////////////

var usageFormatShort string = `usage:  %s [--help] [--in <ansitext-filename>] --out <png-filename>`

var usageFormat string = `usage:  %s [--help] [--in <ansitext-filename>] --out <png-filename>

Converts input ANSI terminal text from stdin (or a file with --in)
and renders it visually as a PNG image file saved to --out.

If --in is '-' then stdin is used. If --out is '-' then stdout is used.

Example:  $ echo -e "\033[31mHello\033[0m World" | ot-ansi-to-png --out hello.png

`

/////////////////////////////////////////////////////////////////////////////////////

func main() {
	var inputTXTFilename, outputPNGFilename string
	var showHelp bool
	var err error

	pflag.StringVarP(&inputTXTFilename, "in", "i", "", "Input text filename (default: stdin)")
	pflag.StringVarP(&outputPNGFilename, "out", "o", "", "Output PNG filename ('-' is stdout)")
	pflag.BoolVarP(&showHelp, "help", "", false, "show help")
	pflag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stdout, usageFormat, os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}
	if len(outputPNGFilename) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: missing required argument: --out\n")
		fmt.Fprintf(os.Stderr, usageFormatShort, os.Args[0])
		os.Exit(1)
	}

	// Open input TXT file for reading, or use Stdin
	infile := os.Stdin
	if len(inputTXTFilename) != 0 && inputTXTFilename != "-" {
		infile, err = os.OpenFile(inputTXTFilename, os.O_RDONLY, 0777)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to open input file %s\n", err.Error())
			os.Exit(1)
		}
		defer infile.Close()
	}

	// Capture file until EOF
	ansitextData, err := io.ReadAll(infile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read file %s\n", err.Error())
		os.Exit(1)
	}
	infile.Close() // we don't need it anymore

	// Use OllamaTeas's machinery to convert to image
	pngBytes, err := ollamatea.ConvertTerminalTextToImage(string(ansitextData), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to convert to PNG %s\n", err.Error())
		os.Exit(1)
	}

	// Write file
	outfile := os.Stdout
	if outputPNGFilename != "" && outputPNGFilename != "-" {
		outfile, err = os.OpenFile(outputPNGFilename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to open output file %s\n", err.Error())
			os.Exit(1)
		}
		defer outfile.Close()
	}

	_, err = outfile.Write(pngBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write PNG %s\n", err.Error())
		os.Exit(1)
	}
}
