# altFINS CLI

Terminal-native crypto market intelligence for traders, analysts, and AI agents.

`af` gives you fast access to altFINS market data, analytics, signals, news, and a full-screen TUI without asking you to install a language toolchain or wire up your own API client.

<p align="center">
  <img src="docs/assets/tui-markets.svg" alt="altFINS CLI markets TUI" width="100%" />
</p>

<p align="center">
  <img src="docs/assets/tui-signals.svg" alt="altFINS CLI signals TUI" width="100%" />
</p>

## Features at a Glance

- Interactive TUI for markets, signals, technical analysis, and news
- Candle-first OHLC charts with green/red bodies and braille fallback for tight panes
- Machine-readable `json`, `jsonl`, and `csv` output for scripts and pipelines
- `--dry-run` request previews with redacted auth headers
- `af commands -o json` for agent and LLM self-discovery
- Reference data for symbols, intervals, field types, and signal keys
- Historical analytics and OHLCV endpoints exposed as clean CLI workflows
- Single binary distribution with Homebrew-first installation

## Install

### Homebrew

One-line install:

```bash
brew install altfins-com/tap/af
```

If you prefer an explicit tap step:

```bash
brew tap altfins-com/tap
brew install af
```

### Manual

Download the archive for your platform from [GitHub Releases](https://github.com/altfins-com/altfins-cli/releases), extract it, and place `af` somewhere in your `PATH`.

## 60-Second Quickstart

1. Save your altFINS API key:

```bash
af auth set
```

2. Verify that auth is configured:

```bash
af auth status
af quota all
```

3. Start using the CLI immediately:

```bash
af markets search --symbols BTC,ETH --display-type MARKET_CAP,RSI14
af signals list --direction BULLISH --from 2026-03-01
af tui markets
```

## Real CLI Snippets

### Auth and First Check

```bash
af auth set
af auth status
af quota all
```

If your API key is valid, `af quota all` gives you an immediate sanity check against your current and monthly permit counts.

### Agent-Friendly Dry Run

Preview the exact request shape before you execute a live query:

```bash
ALTFINS_API_KEY=demo-key af markets search \
  --symbols BTC,ETH \
  --display-type MARKET_CAP,RSI14 \
  --dry-run -o json
```

```json
{
  "method": "POST",
  "url": "https://altfins.com/api/v2/public/screener-data/search-requests",
  "body": {
    "displayType": ["MARKET_CAP", "RSI14"],
    "symbols": ["BTC", "ETH"]
  },
  "headers": {
    "Accept": "application/json",
    "Content-Type": "application/json",
    "X-Api-Key": "redacted"
  },
  "authSource": "env"
}
```

### Typical Table Workflow

Use the screener like a terminal-native market scanner:

```bash
af markets search --symbols BTC,ETH --display-type MARKET_CAP,RSI14
```

```text
symbol  name      lastPrice  MARKET_CAP  RSI14
BTC     Bitcoin   70000      1000000     61.2
ETH     Ethereum  4000       500000      57.8
```

### Signal Hunting

```bash
af signals list --direction BULLISH --from 2026-03-01
af signals keys
```

```text
timestamp              symbol  symbolName  direction  signalKey                      signalName
2026-03-18T09:30:00Z   BTC     Bitcoin     BULLISH    FRESH_MOMENTUM_MACD_SIGNAL...  Bullish MACD crossover
2026-03-18T10:15:00Z   SOL     Solana      BULLISH    SUPPORT_RESISTANCE_BREAKOUT     Resistance breakout
```

### Technical Analysis and News

```bash
af ta list --symbol SOL
af news list --from 2026-03-01
af news get --message-id 12345 --source-id 12
```

## TUI Experience

Launch any of the full-screen terminal views:

```bash
af tui markets
af tui signals
af tui ta
af tui news
```

Keyboard controls:

```text
j / k or arrows  Move selection
/                Filter the current list
Enter            Open detail mode
Esc              Return to the list
Tab              Switch focus
f                Toggle filter drawer
z                Toggle chart zoom
c                Toggle candles/braille
i                Cycle chart interval
r                Refresh data
q                Quit
```

The current TUI layout includes:

- A left-side browser for lists and search
- A chart-first detail pane for markets, signals, and technical analysis
- Green/red OHLC candles with a braille fallback when the pane is narrow
- A dedicated chart zoom mode for the selected asset
- Interval cycling between hourly, 4-hour, and daily presets inside the TUI
- Active filter visibility
- Permit counts in the footer

Example candle-style detail:

```text
BTC  DAILY  30 candles
O 70000  H 71450  L 69210  C 70980  @ 2026-03-18 00:00 UTC

71450 ┤          │ █
70980 ┤      █   │██
70500 ┤    ████  ███
70000 ┤  ███████ ███
69600 ┤ ██ │ ████ │
69210 ┤ │  │  ██  │
      └────────────────
        03/01   03/10   03/18
```

## Common Workflows

### Screen the market

```bash
af markets fields
af markets search --symbols BTC,ETH,SOL --display-type MARKET_CAP,RSI14,MACD
```

### Pull historical analytics

```bash
af analytics types
af analytics history --symbol BTC --type RSI14 --interval DAILY --from 2026-03-01 --to 2026-03-18
```

### Retrieve OHLCV data

```bash
af ohlcv snapshot --symbols BTC,ETH --interval DAILY
af ohlcv history --symbol BTC --interval DAILY --from 2026-03-01 --to 2026-03-18
```

### Explore reference data

```bash
af refs symbols
af refs intervals
af signals keys
af analytics types
```

## AI and LLM Friendly by Design

Use `af` as a terminal tool for agents, evals, workflows, and prompt-time retrieval:

```bash
af commands -o json
af markets search --symbols BTC,ETH --display-type MARKET_CAP,RSI14 -o json
af signals list --direction BULLISH --from 2026-03-01 -o json
```

Example command metadata:

```bash
af commands -o json | sed -n '1,20p'
```

```json
{
  "command": "af",
  "use": "af",
  "short": "altFINS CLI for market data, signals, analytics, and TUI workflows",
  "flags": [
    {
      "name": "dry-run",
      "type": "bool",
      "usage": "Show the API request without executing it"
    }
  ]
}
```

Why this matters:

- JSON output stays machine-readable on stdout
- dry runs expose endpoint and request body shape before execution
- `af commands` gives agents a self-documenting command index
- one binary is easier to drop into local tools, shells, and CI jobs

## Project Links

- altFINS API docs: <https://altfins.com/crypto-market-and-analytical-data-api/documentation/>
- altFINS OpenAPI schema: <https://altfins.com/crypto-market-and-analytical-data-api/openApi.json>
- Contributor workflow: [CONTRIBUTING.md](CONTRIBUTING.md)
- Release and distribution notes: [docs/releasing.md](docs/releasing.md)

## Contributing

The main README is intentionally focused on installation and usage. Source builds, tests, and release workflow live in [CONTRIBUTING.md](CONTRIBUTING.md).
