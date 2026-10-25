# Path-field inventory method

The v1 and v2 inventories are derived from the pinned ACP schema release assets downloaded by the `Test` workflow, not
from field-name searches alone.

For each schema version, inspect method parameters and results through their `$ref` chains. Include a field only when
the schema defines it as one of:

- a filesystem path, working directory, or workspace directory;
- a file URI in a typed protocol value; or
- an executable path, such as a stdio MCP server command.

Map only the documented field within its known method and tagged-union variant.

## Unknown fields

Embedded schemas validate traffic. For schema-invalid messages, `--unknown-fields` is enabled by default and rewrites
whole-string absolute paths in payload fields absent from the matching schema branch:

- `C:\work` or `C:/work` becomes `/mnt/c/work` toward Linux.
- `/mnt/c/work` becomes `C:\work` toward Windows.

Set `--unknown-fields=false` to disable this pass.

Schema-defined command arguments, environment values, free text, HTTP URLs, and tagged-union values remain unchanged.
Only `C:\`, `C:/`, and `/mnt/c` candidates are inspected. Candidates over 128 KiB are skipped.

When a schema release changes, update the embedded release asset, repeat the inventory review, update the versioned
inventory, and add a schema-validated mapping test for each newly included field.
