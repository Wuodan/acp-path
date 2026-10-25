// Command acp-wsl bridges ACP traffic between Windows clients and Linux agents.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/aicage/acp-wsl/internal/bridge"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "acp-wsl:", err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output, stderr io.Writer) error {
	flags := flag.NewFlagSet("acp-wsl", flag.ContinueOnError)
	flags.SetOutput(stderr)
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
		_, err := fmt.Fprintf(output, "acp-wsl %s\n", version)
		return err
	}
	rest := flags.Args()
	if !hasSeparator || len(rest) == 0 {
		return errors.New("downstream command must follow --")
	}
	return bridge.Run(input, output, stderr, nil, rest, *unknownFields)
}
