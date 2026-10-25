# Contributing

## Development

Install the Go toolchain declared in `go.mod`, then run the full test suite:

```text
go test ./...
```

Format changed Go files before submitting them:

```text
gofmt -w path/to/file.go
```

## Embedded ACP schemas

`acp-wsl` embeds the ACP v1 and v2 schemas used to identify documented path fields. Their pinned releases are in
`.github/versions/acp-schema.yml`; the embedded copies are in `internal/transform/acpschema/`.

When updating either pinned version, download the corresponding release asset into the matching embedded file and
commit the version-pin and schema changes together. CI downloads the pinned schemas and checks that they exactly match
the embedded copies.
