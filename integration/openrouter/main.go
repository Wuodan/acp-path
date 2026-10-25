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
	protocolVersion := flag.Int("protocol-version", 1, "ACP protocol version")
	flag.Parse()

	if err := runSmoke(flag.Args(), *protocolVersion); err != nil {
		fmt.Fprintln(os.Stderr, "openrouter smoke:", err)
		os.Exit(1)
	}
}

func runSmoke(agentCommand []string, protocolVersion int) error {
	if len(agentCommand) == 0 {
		return fmt.Errorf("agent command is required after --")
	}
	if protocolVersion != 1 && protocolVersion != 2 {
		return fmt.Errorf("unsupported ACP protocol version %d", protocolVersion)
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

	if err := initialize(client, protocolVersion); err != nil {
		return err
	}
	sessionID, err := createSession(client, workspace)
	if err != nil {
		return err
	}
	if err := askForMarkerFiles(client, sessionID, absoluteMarkerPath, protocolVersion); err != nil {
		return err
	}
	if err := checkMarkerFile(markerTarget, marker); err != nil {
		return err
	}
	return checkMarkerFile(absoluteMarkerTarget, absoluteMarker)
}
