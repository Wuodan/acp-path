// Package main runs the OpenRouter integration smoke test.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const smokeTimeout = 100 * time.Minute

func main() {
	flag.Parse()

	if err := runSmoke(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "openrouter smoke:", err)
		os.Exit(1)
	}
}

func runSmoke(agentCommand []string) error {
	if len(agentCommand) == 0 {
		return fmt.Errorf("agent command is required after --")
	}
	workspace := os.Getenv("GITHUB_WORKSPACE")
	if workspace == "" {
		return fmt.Errorf("GITHUB_WORKSPACE is not set")
	}
	absoluteMarkerPath := os.Getenv("ACP_PATH_SMOKE_ABSOLUTE_MARKER_PATH")
	if absoluteMarkerPath == "" {
		return fmt.Errorf("ACP_PATH_SMOKE_ABSOLUTE_MARKER_PATH is not set")
	}

	markerTarget := filepath.Join(workspace, marker+".txt")
	absoluteMarkerTarget := filepath.Join(workspace, "testdata", absoluteMarker+".txt")
	if err := removePreviousMarker(markerTarget); err != nil {
		return err
	}
	if err := removePreviousMarker(absoluteMarkerTarget); err != nil {
		return err
	}

	smokeContext, cancel := context.WithTimeout(context.Background(), smokeTimeout)
	defer cancel()

	client, command, err := startAgent(smokeContext, agentCommand)
	if err != nil {
		return err
	}
	defer stopAgent(command, client.input)

	if err := initialize(client); err != nil {
		return err
	}
	sessionID, err := createSession(client, workspace)
	if err != nil {
		return err
	}
	if err := askForMarkerFiles(client, sessionID, absoluteMarkerPath); err != nil {
		return err
	}
	if err := checkMarkerFile(markerTarget, marker); err != nil {
		return err
	}
	return checkMarkerFile(absoluteMarkerTarget, absoluteMarker)
}
