# stock-tui

Real-time stock and cryptocurrency tracker for the terminal.

<p align="center">
  <img src="https://img.shields.io/badge/go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img src="https://img.shields.io/github/license/ni5arga/stock-tui?style=flat-square">
  <img src="https://img.shields.io/github/actions/workflow/status/ni5arga/stock-tui/build.yml?branch=main&style=flat-square">
</p>

![screenshot](screenshots/stock-tui.png)

## Features

- Real-time price tracking for stocks and cryptocurrencies
- Multiple data providers (CoinGecko, Yahoo Finance, or combined)
- Historical price charts with multiple time ranges
- Sparkline visualization
- Keyboard-driven interface with Vim-style navigation

## Installation

### Method 1: Build from Source (Recommended)

This method ensures you have the configuration file and all assets immediately available.

```bash
git clone https://github.com/ni5arga/stock-tui.git
cd stock-tui
go build ./cmd/stock-tui
./stock-tui
```

### Method 2: Go Install (Quick Start)

Good for trying out the app with default settings.

```bash
go install github.com/ni5arga/stock-tui/cmd/stock-tui@latest
```

## Configuration

The app looks for configuration in the following order:

1. **CLI Flag**: `--config` / `-c` (e.g., `stock-tui -c /path/to/conf.toml`)
2. **Environment Variable**: `STOCK_TUI_CONFIG`
3. **User Config Directory** (XDG supported):
   - Linux/Mac: `~/.config/stock-tui/config.toml`
   - Windows: `%APPDATA%\stock-tui\config.toml`
4. **Current Directory**: `./config.toml`

A sample `config.toml` is included in the repo. To use it system-wide:

```bash
mkdir -p ~/.config/stock-tui
curl -sL https://raw.githubusercontent.com/ni5arga/stock-tui/main/config.toml > ~/.config/stock-tui/config.toml
```

**Example config.toml:**

```toml
# Data provider: "simulator", "coingecko", "yahoo", or "multi" (default)
provider = "multi"

# Refresh interval
refresh_interval = "5s"

# Default chart range: "1H", "24H", "7D", "30D"
default_range = "24H"

# Watchlist symbols
# Crypto: use -USD suffix (BTC-USD, ETH-USD)
# Stocks: use ticker (AAPL, GOOGL)
symbols = [
    "BTC-USD",
    "ETH-USD",
    "SOL-USD",
    "AAPL",
    "GOOGL",
    "TSLA",
    "MSFT",
    "NVDA"
]
```

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down in watchlist |
| `k` / `↑` | Move up in watchlist |
| `/` | Search/filter symbols |
| `Esc` | Exit search mode |
| `s` | Cycle sort mode (Name/Price/Change%) |
| `S` | Toggle sort direction (Asc/Desc) |
| `Tab` | Cycle time range |
| `1` | 1 hour range |
| `2` | 24 hour range |
| `3` | 7 day range |
| `4` | 30 day range |
| `Tab` | Cycle chart type (Line/Area/Candle) |
| `r` | Refresh data |
| `?` | Toggle help |
| `q` | Quit |

## Data Providers

| Provider | Assets | API Key |
|----------|--------|---------|
| `simulator` | Fake data | None |
| `coingecko` | Crypto | None (free tier) |
| `yahoo` | Stocks | None (unofficial) |
| `multi` | Both | None |

> **Note**: Yahoo Finance API is unofficial and may have rate limits.
> CoinGecko free tier allows ~10-30 requests/minute.

## Supported Platforms

- Linux
- macOS
- Windows

## Architecture

```
cmd/stock-tui/       Entry point
internal/
├── app/             Bubble Tea model
├── config/          Viper configuration
├── data/            Provider implementations
├── models/          Domain types
└── ui/
    ├── chart/       Price chart component
    ├── footer/      Status bar
    ├── help/        Help overlay
    ├── modal/       Generic modal
    ├── styles/      Lip Gloss styles
    └── watchlist/   Symbol list
```

## Development

```bash
# Run
go run ./cmd/stock-tui

# Build
go build ./cmd/stock-tui

# Test
go test ./...

# Lint
go vet ./...
```

## License

[MIT](https://github.com/ni5arga/stock-tui/blob/main/LICENSE)
