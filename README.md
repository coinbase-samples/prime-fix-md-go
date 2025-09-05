# Prime FIX Market Data Client

A Go-based FIX protocol client for receiving real-time and snapshot market data from Coinbase Prime.

## Features

- **Real-time Market Data**: Subscribe to live trades, order book updates, and OHLCV candles
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
- `--depth N` - Number of price levels (0=full, 1=top, N=best N levels)

**Data Types:**
- `--trades` - Trade executions
- `--bids` - Bid prices
- `--offers` - Ask/offer prices  
- `--open` - Opening price
- `--close` - Closing price
- `--high` - High price
- `--low` - Low price
- `--volume` - Trading volume

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

# Level 1 order book snapshot  
md BTC-USD --snapshot --depth 1 --bids --offers

# Level 10 order book snapshot
md BTC-USD --snapshot --depth 10 --bids --offers

# Subscribe to live trades
md BTC-USD --subscribe --trades

# Subscribe to live Level 5 order book
md BTC-USD --subscribe --depth 5 --bids --offers

# OHLCV snapshot
md ETH-USD --snapshot --open --close --high --low --volume

# Subscribe to live candle updates
md BTC-USD --subscribe --open --close --high --low --volume

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
md BTC-USD --subscribe --depth 5 --bids     # Live L5 bids (reqID: md_456)  
md BTC-USD --subscribe --depth 5 --offers   # Live L5 offers (reqID: md_789)
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
│ BTC-USD     │ Snapshot + Updates │ Active     │ 150         │ 14:23:45     │ ...4111000       │
│             │ Snapshot + Updates │ Active     │ 89          │ 14:23:45     │ ...4222000       │
│ ETH-USD     │ Snapshot + Updates │ Active     │ 45          │ 14:22:10     │ ...4333000       │
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

## Database Schema

Market data is automatically stored in `marketdata.db` with the following structure:

### Tables

**sessions** - Request metadata
- `session_id` - Unique session identifier
- `symbol` - Trading pair (e.g., BTC-USD)
- `request_type` - "snapshot" or "subscribe"
- `data_types` - "trades", "bids,offers", or "ohlcv"  
- `depth` - Order book depth (NULL for trades/ohlcv)
- `md_req_id` - FIX request ID
- `created_at` - Session creation time
- `is_active` - Session status

**trades** - Trade executions
- `symbol` - Trading pair
- `price` - Trade price
- `size` - Trade size
- `aggressor_side` - "Buy" or "Sell"
- `trade_time` - Exchange timestamp
- `seq_num` - FIX sequence number
- `is_snapshot` - TRUE if from snapshot, FALSE if streaming
- `received_at` - Local receive time

**order_book** - Bid/offer levels
- `symbol` - Trading pair
- `side` - "bid" or "offer"
- `price` - Price level
- `size` - Size at price level
- `position` - Book position (1=best, 2=second best, etc.)
- `seq_num` - FIX sequence number
- `is_snapshot` - TRUE if from snapshot, FALSE if streaming
- `received_at` - Local receive time

**ohlcv** - Candle data
- `symbol` - Trading pair
- `data_type` - "open", "high", "low", "close", or "volume"
- `value` - Data value
- `entry_time` - Exchange timestamp
- `seq_num` - FIX sequence number
- `received_at` - Local receive time

### Example Queries

```sql
-- Recent trades for BTC-USD
SELECT price, size, aggressor_side, received_at 
FROM trades 
WHERE symbol = 'BTC-USD' 
ORDER BY received_at DESC 
LIMIT 10;

-- Current order book snapshot
SELECT side, price, size, position 
FROM order_book 
WHERE symbol = 'BTC-USD' AND is_snapshot = 1 
ORDER BY side, position;

-- Latest OHLCV data
SELECT data_type, value, entry_time 
FROM ohlcv 
WHERE symbol = 'BTC-USD' 
ORDER BY received_at DESC 
LIMIT 5;
```

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

## File Structure

```
prime-fix-md-go/
├── cmd/main.go              # Application entry point
├── fixclient/               # FIX protocol client
│   ├── fixapp.go           # Main FIX application logic
│   └── tradestore.go       # In-memory trade storage
├── database/               # Database layer
│   └── marketdata.go       # SQLite operations
├── builder/                # FIX message builders
│   └── messages.go         # Market data request messages
├── constants/              # FIX protocol constants
│   └── constants.go        # Message types and field tags
├── formatter/              # Logging and display
│   └── logfactory.go       # FIX message logging
├── utils/                  # Utilities
│   ├── utils.go           # Helper functions
│   └── version.go         # Version information
├── fix.cfg                 # FIX session configuration
├── marketdata.db          # SQLite database (created at runtime)
├── go.mod                 # Go module dependencies
└── README.md              # This file
```

## Dependencies

- `github.com/quickfixgo/quickfix` - FIX protocol implementation
- `github.com/mattn/go-sqlite3` - SQLite database driver
- `github.com/chzyer/readline` - Interactive CLI with completion

## License

Licensed under the Apache License, Version 2.0.