package transform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestV2PermissionFixtureMatchesProtocolSchema(t *testing.T) {
	valid := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/request_permission","params":{"sessionId":"session","title":"Run command","options":[{"optionId":"allow","name":"Allow once","kind":"allow_once"}],"subject":{"type":"command","command":"pwd","cwd":"C:\\work"}}}`)
	validateACPFixture(t, 2, valid)
}

func validateACPFixture(t *testing.T, version int, fixture []byte) {
	t.Helper()
	if err := validateACP(version, fixture); err != nil {
		t.Fatal(err)
	}
}

func validateACP(version int, fixture []byte) error {
	schemaDir := os.Getenv("ACP_PATH_SCHEMA_DIR")
	if schemaDir == "" {
		return nil
	}
	var value any
	if err := json.Unmarshal(fixture, &value); err != nil {
		return err
	}
	compiler := jsonschema.NewCompiler()
	name := filepath.Join(schemaDir, "v"+strconv.Itoa(version), "schema.json")
	schema, err := compiler.Compile(name)
	if err != nil {
		return err
	}
	return schema.Validate(value)
}
