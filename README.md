# Prime FIX Market Data Client

A Go-based FIX protocol client for receiving real-time and snapshot market data from Coinbase Prime.

## Features

- **Real-time Market Data**: Subscribe to live trades, order book updates, and OHLCV data
- **Snapshot Data**: Get point-in-time snapshots of trades, order books, and OHLCV data  
- **SQLite Storage**: All market data is stored in a local SQLite database for analysis
- **CLI Interface**: Interactive command-line interface with tab completion
- **Multiple Data Types**: Supports trades, bids/offers, and OHLCV (Open, High, Low, Close, Volume)

## Prerequisites

- Go 1.23.2 or higher
- Coinbase Prime account with FIX API access
- Valid API credentials (Access Key, Signing Key, Passphrase, Service Account ID, Portfolio ID)

## Installation

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the application:
   ```bash
   go build -o fix-md-client ./cmd
   ```

## Configuration

Create a `fix.cfg` file in the project root with your FIX session configuration. Example:

```ini
[DEFAULT]
ConnectionType=initiator
ReconnectInterval=60
SenderCompID=your-service-account-id
TargetCompID=COIN
SocketConnectHost=fix-md.prime.coinbase.com
SocketConnectPort=4199
HeartBtInt=30
LogoutTimeout=5
LogonTimeout=5
```

## Environment Variables

Set the following environment variables with your Coinbase Prime credentials:

```bash
export ACCESS_KEY="your-access-key"
export SIGNING_KEY="your-signing-key" 
export PASSPHRASE="your-passphrase"
export SVC_ACCOUNT_ID="your-service-account-id"
export TARGET_COMP_ID="COIN"
export PORTFOLIO_ID="your-portfolio-id"
```

## Usage

Run the application:
```bash
# Option 1: Run directly (no build needed)
go run cmd/main.go

# Option 2: Run compiled binary
./fix-md-client
```

### Available Commands

#### Market Data Request
```bash
md <symbol> [flags...]
```

**Subscription Types:**
- `--snapshot` - Get a one-time snapshot
- `--subscribe` - Subscribe to real-time updates
- `--unsubscribe` - Stop real-time updates

**Depth Control (for order books):**
- `--depth 0` - Full order book (all available price levels)
- `--depth 1` - Top of book L1 (best bid + best offer only)  
- `--depth N` - LN book (best N bids + best N offers, e.g., L5, L10, L25)
                Automatically includes both bids and offers

**Data Types:**
- `--trades` - Trade executions
- `--o` - Opening price
- `--c` - Closing price
- `--h` - High price
- `--l` - Low price
- `--v` - Trading volume

#### Unsubscribe Commands
```bash
unsubscribe <symbol|reqID>
```

**Unsubscribe Options:**
- `unsubscribe BTC-USD` - Cancel ALL active subscriptions for BTC-USD
- `unsubscribe md_1234567890` - Cancel specific subscription by reqID  
- `unsubscribe --reqid md_123` - Cancel specific subscription (explicit flag)

**Auto-detection**: Inputs starting with "md_" are treated as reqIDs, otherwise as symbols.

#### Other Commands
- `status` - Show active subscriptions with reqIDs (live streams only)
- `help` - Display help information
- `version` - Show version
- `exit` - Quit application

### Example Commands

```bash
# Trade snapshot (100 most recent trades)
md BTC-USD --snapshot --trades

# Full order book snapshot (all available levels)
md BTC-USD --snapshot --depth 0

# First level order book snapshot (best bid + best offer)
md BTC-USD --snapshot --depth 1

# First 10 levels order book snapshot (best 10 bids + 10 offers)
md BTC-USD --snapshot --depth 10

# Subscribe to live trades
md BTC-USD --subscribe --trades

# Subscribe to live Level 5 order book (5 bids + 5 offers)
md BTC-USD --subscribe --depth 5

# OHLCV snapshot
md ETH-USD --snapshot --o --c --h --l --v

# Subscribe to live candle updates (allow 30s for connection to establish)
md BTC-USD --subscribe --o --c --h --l --v

# Unsubscribe examples
unsubscribe BTC-USD                    # Cancel ALL BTC-USD subscriptions
unsubscribe md_1757035274634111000     # Cancel specific subscription by reqID
unsubscribe --reqid md_123456789       # Cancel specific subscription (explicit)
md BTC-USD --unsubscribe               # Alternative: cancel ALL BTC-USD subscriptions

# Check active subscriptions
status
```

## Subscription Management

### Multiple Subscriptions
You can have multiple active subscriptions per symbol. For example:
```bash
md BTC-USD --subscribe --trades              # Live trades (reqID: md_123)
md BTC-USD --subscribe --depth 5             # Live L5 bids+offers (5 bids + 5 offers, reqID: md_456)
```

### Subscription Tracking
- **Snapshots** (`--snapshot`) are not tracked (one-time requests)
- **Subscriptions** (`--subscribe`) are tracked in the `status` display
- Each subscription gets a unique `reqID` for precise control

### Unsubscribe Behavior
- **Symbol-based**: `unsubscribe BTC-USD` cancels ALL BTC-USD subscriptions
- **ReqID-based**: `unsubscribe md_123` cancels only that specific subscription
- **Auto-detection**: Inputs starting with "md_" are treated as reqIDs

### Status Display
```bash
FIX-MD> status
Active Subscriptions:
┌─────────────┬──────────────────┬─────────────┬─────────────┬──────────────┬──────────────────┐
│ Symbol      │ Type             │ Status      │ Updates     │ Last Update  │ ReqID            │
├─────────────┼──────────────────┼─────────────┼─────────────┼──────────────┼──────────────────┤
│ BTC-USD     │ Snapshot + Updates │ Active    │ 150         │ 14:23:45     │ ...4111000       │
│             │ Snapshot + Updates │ Active    │ 89          │ 14:23:45     │ ...4222000       │
│ ETH-USD     │ Snapshot + Updates │ Active    │ 45          │ 14:22:10     │ ...4333000       │
└─────────────┴──────────────────┴─────────────┴─────────────┴──────────────┴──────────────────┘
```

## Data Capabilities

### Depth Support
- **Trades**: Always returns ~100 most recent (depth parameter ignored)
- **Order Book (bids/offers)**: Supports L1, L5, L10, L25, etc.  
- **OHLCV**: Always returns ~100 entries (depth parameter ignored)

### Subscription Support
- **Trades**: Supports real-time streaming
- **Order Book (bids/offers)**: Supports real-time streaming
- **OHLCV**: Supports real-time candle updates

## Understanding Market Data

### Order Book Depth Subscriptions
When you subscribe to order book depth (e.g., `--depth 3`), you'll receive incremental updates as market conditions change:

- **Depth 3** means "up to 3 best price levels" on each side
- You may not always see exactly 3 bids and 3 offers due to:
  - Price levels being removed when size reaches zero
  - New price levels appearing within the top N levels
  - Market gaps where fewer than N levels exist
- Updates show real-time changes: new levels, size changes, level removals
- Track position numbers to understand level ranking (1=best, 2=second best, etc.)

### Snapshot vs Live Data
- **Snapshots** (`--snapshot`) - One-time current state, not tracked
- **Subscriptions** (`--subscribe`) - Continuous live updates, tracked in `status`

## Data Storage

Market data is stored in `marketdata.db` (SQLite) with tables for:
- **trades** - Trade executions with price, size, and timestamps
- **order_book** - Bid/offer levels with position and depth
- **ohlcv** - Open, high, low, close, and volume data
- **sessions** - Request metadata and subscription tracking

## Output Format

### Snapshot Display
Snapshots are displayed in formatted tables showing all received data.

### Streaming Display  
Real-time updates are shown as individual lines:

```
Market Data Incremental for BTC-USD (ReqID: md_1234567890, Entries: 2, Seq: 42)
BTC-USD Trade: 50000.00 | Size: 0.1 | Aggressor: Buy
BTC-USD Trade: 50001.00 | Size: 0.05 | Aggressor: Sell
────────────────────────────────────────────────

BTC-USD Bid: 49995.00 | Size: 1.5 | Pos: 1
BTC-USD Offer: 50005.00 | Size: 2.0 | Pos: 1
────────────────────────────────────────────────
```