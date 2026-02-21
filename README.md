# MarkyMark

![eeehm](./assets/akq73x.jpg)

## What is this?

MarkyMark is a markdown viewer with that 90s vibe.

## Installation

### Homebrew (macOS)

```sh
brew install madflow/markymark/markymark
```

### Go install

```sh
go install github.com/madflow/markymark@latest
```

### Build from source

**Prerequisites:**

- [Go 1.23+](https://go.dev/dl/)
- [`templ` CLI](https://templ.guide/): `go install github.com/a-h/templ/cmd/templ@latest`

```sh
git clone https://github.com/madflow/markymark.git
cd markymark
make build
```

This produces a `./markymark` binary in the project root.

### Pre-built binaries

Download archives for Linux, macOS, and Windows from the [GitHub Releases](https://github.com/madflow/markymark/releases) page.

## Usage

### Basic usage

```sh
# Open a specific markdown file
markymark path/to/file.md

# Auto-detect README.md in the current directory
markymark
```

MarkyMark starts a local server at `http://localhost:3000` and opens it in your default browser automatically.

### Watch mode

Use `-w` / `--watch` to live-reload the browser whenever the file changes:

```sh
markymark -w path/to/file.md

# Watch mode with auto-detected README
markymark -w
```

### Flags

| Flag      | Shorthand | Default | Description                                            |
| --------- | --------- | ------- | ------------------------------------------------------ |
| `--watch` | `-w`      | `false` | Watch the file for changes and auto-reload the browser |

