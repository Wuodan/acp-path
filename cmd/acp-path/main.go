// Command acp-path bridges ACP traffic between Windows clients and Linux agents.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/Wuodan/acp-path/internal/transform"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "acp-path:", err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("acp-path", flag.ContinueOnError)
	flags.SetOutput(stderr)
	defaultLog, err := defaultLogFile()
	if err != nil {
		return err
	}
	logFile := flags.String("log-file", defaultLog, "append development traffic log")
	hasSeparator := false
	for _, argument := range args {
		if argument == "--" {
			hasSeparator = true
			break
		}
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	rest := flags.Args()
	if !hasSeparator || len(rest) == 0 {
		return errors.New("downstream command must follow --")
	}
	logWriter, err := openLog(*logFile)
	if err != nil {
		return err
	}
	defer logWriter.Close()
	logger := log.New(logWriter, "", log.LstdFlags|log.Lmicroseconds)
	command := exec.Command(rest[0], rest[1:]...)
	childIn, err := command.StdinPipe()
	if err != nil {
		return fmt.Errorf("open downstream stdin: %w", err)
	}
	childOut, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open downstream stdout: %w", err)
	}
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		return fmt.Errorf("start downstream command: %w", err)
	}
	logger.Printf("subprocess started: %q", rest)
	mapper := transform.New()
	var copyWG sync.WaitGroup
	copyWG.Add(2)
	go func() {
		defer copyWG.Done()
		defer childIn.Close()
		forward(input, childIn, mapper, transform.ToLinux, logger, "client->agent")
	}()
	go func() {
		defer copyWG.Done()
		forward(childOut, output, mapper, transform.ToWindows, logger, "agent->client")
	}()
	copyWG.Wait()
	err = command.Wait()
	if err != nil {
		logger.Printf("subprocess exited with error: %v", err)
		return err
	}
	logger.Printf("subprocess exited successfully")
	return nil
}

func forward(source io.Reader, destination io.Writer, mapper *transform.Transformer, direction transform.Direction, logger *log.Logger, label string) {
	reader := bufio.NewReader(source)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			mapped, decisions := mapper.Transform(line, direction)
			logger.Printf("%s raw=%q mapped=%q decisions=%+v", label, line, mapped, decisions)
			if _, writeErr := destination.Write(mapped); writeErr != nil {
				logger.Printf("%s write error: %v", label, writeErr)
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logger.Printf("%s read error: %v", label, err)
			}
			return
		}
	}
}

func defaultLogFile() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find user cache directory: %w", err)
	}
	return filepath.Join(cache, "acp-path", "development.log"), nil
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
