// Package bridge forwards ACP traffic between a client and a subprocess.
package bridge

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"

	"github.com/aicage/acp-wsl/internal/transform"
)

// Run starts commandLine and forwards ACP traffic between its standard streams.
// When logger is non-nil, it records complete directional traffic.
func Run(input io.Reader, output, stderr io.Writer, logger *log.Logger, commandLine []string, unknownFields bool) error {
	command := exec.Command(commandLine[0], commandLine[1:]...)
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
	logf(logger, "subprocess started: %q", commandLine)
	mapper := transform.New(unknownFields)
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
	if err := command.Wait(); err != nil {
		logf(logger, "subprocess exited with error: %v", err)
		return err
	}
	logf(logger, "subprocess exited successfully")
	return nil
}

func forward(source io.Reader, destination io.Writer, mapper *transform.Transformer, direction transform.Direction, logger *log.Logger, label string) {
	reader := bufio.NewReader(source)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			mapped, decisions := mapper.Transform(line, direction)
			logf(logger, "%s raw=%q mapped=%q decisions=%+v", label, line, mapped, decisions)
			if _, writeErr := destination.Write(mapped); writeErr != nil {
				logf(logger, "%s write error: %v", label, writeErr)
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logf(logger, "%s read error: %v", label, err)
			}
			return
		}
	}
}

func logf(logger *log.Logger, format string, args ...any) {
	if logger != nil {
		logger.Printf(format, args...)
	}
}
