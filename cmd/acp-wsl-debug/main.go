// Command acp-wsl-debug bridges ACP traffic and records complete raw traffic.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/aicage/acp-wsl/internal/bridge"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "acp-wsl-debug:", err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output, stderr io.Writer) error {
	flags := flag.NewFlagSet("acp-wsl-debug", flag.ContinueOnError)
	flags.SetOutput(stderr)
	logFile := flags.String("log-file", "", "write complete raw traffic to this file (required)")
	unknownFields := flags.Bool("unknown-fields", true, "translate paths in fields outside the ACP schema")
	showVersion := flags.Bool("version", false, "print version and exit")
	hasSeparator := false
	for _, argument := range args {
		if argument == "--" {
			hasSeparator = true
			break
		}
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *showVersion {
		_, err := fmt.Fprintf(output, "acp-wsl-debug %s\n", version)
		return err
	}
	rest := flags.Args()
	if !hasSeparator || len(rest) == 0 {
		return errors.New("downstream command must follow --")
	}
	var logger *log.Logger
	if *logFile != "" {
		logWriter, err := openLog(*logFile)
		if err != nil {
			return err
		}
		defer logWriter.Close()
		logger = log.New(logWriter, "", log.LstdFlags|log.Lmicroseconds)
	}
	return bridge.Run(input, output, stderr, logger, rest, *unknownFields)
}

func openLog(name string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	file, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return file, nil
}
