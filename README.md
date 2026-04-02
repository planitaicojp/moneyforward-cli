# Money Forward CLI

Money Forward Cloud APIs (Invoice, Expense) CLI tool.

## Install

### macOS (Homebrew)

```bash
brew install planitaicojp/tap/mf
```

### Windows (go install)

```bash
go install github.com/planitaicojp/moneyforward-cli@latest
```

The binary name is `mf`. Make sure `$GOPATH/bin` (or `$GOBIN`) is in your `PATH`.

### Windows (manual download)

Download the latest release from [GitHub Releases](https://github.com/planitaicojp/moneyforward-cli/releases), extract, and place `mf.exe` in a directory on your `PATH`.

### Linux / From source

```bash
go install github.com/planitaicojp/moneyforward-cli@latest
```

## Quick Start

```bash
# Authenticate
mf auth login --service invoice

# List items
mf invoice items list

# Expense transactions
mf auth login --service expense
mf expense transactions list
```

## Documentation

https://planitaicojp.github.io/moneyforward-cli-pages/

## License

Apache License 2.0
