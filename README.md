# cmd

Executable entrypoints. Each subdirectory is a `package main` that builds one
binary. Keep these thin: parse flags/config, wire dependencies, and start the
service — push real logic down into `internal/<service>` so it stays testable.

Build a specific binary:

```shell
go build ./cmd/<name>
```

- [`summarizer/`](summarizer/README.md) runs the per-CID batch that analyzes ZTA
  reports, optionally obtains GenAI Hub guidance, validates it, and writes
  Markdown narratives.
- [`analyticsapi/`](analyticsapi/README.md) serves stored Markdown narratives by
  CID over HTTP.
