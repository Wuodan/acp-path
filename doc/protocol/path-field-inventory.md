# Path-field inventory method

The v1 and v2 inventories are derived from the pinned ACP schema release assets downloaded by the `Test` workflow, not
from field-name searches alone.

For each schema version, inspect method parameters and results through their `$ref` chains. Include a field only when
the schema defines it as one of:

- a filesystem path, working directory, or workspace directory;
- a file URI in a typed protocol value; or
- an executable path, such as a stdio MCP server command.

Map only the documented field within its known method and tagged-union variant.

Command arguments, environment values, free text, HTTP URLs, extension payloads, and unknown union variants remain
unchanged: the protocol does not establish them as paths or give the bridge enough meaning to rewrite them safely.

When a schema release changes, repeat the inventory review, update the versioned inventory, and add a schema-validated
mapping test for each newly included field.
