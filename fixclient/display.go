/**
 * Copyright 2025-present Coinbase Global, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fixclient

import (
	"fmt"
	"log"

	"prime-fix-md-go/constants"
)

func (a *FixApp) displayHelp() {
	fmt.Print(`Commands:
  md <symbol> [flags...]        - Market data request
  unsubscribe <symbol|reqID>    - Stop subscription(s) (auto-detects symbol vs reqID)
  status                        - Show active subscriptions (live data streams only)
  help                          - Show this help message
  version, exit

Market Data Request Types:
  --snapshot                    - One-time data request (not tracked in status)
  --subscribe                   - Live data stream (tracked in status, requires unsubscribe)
  --unsubscribe                 - Cancel specific subscription by original reqID

Market Data Types:
  --bids, --offers              - Order book data (supports --depth)
  --trades                      - Executed trades (depth ignored, ~100 recent)
  --open, --close, --high, --low, --volume - OHLCV candle data (depth ignored)

Depth Options (for bids/offers only):
  --depth 0                     - Full order book
  --depth 1                     - Top of book (L1)
  --depth N                     - Best N levels (L5, L10, L25, etc.)

Examples:
  md BTC-USD --snapshot --trades                       - Trade snapshot (not tracked)
  md BTC-USD --snapshot --depth 1 --bids --offers     - L1 order book snapshot
  md BTC-USD --snapshot --depth 10 --bids --offers    - L10 order book snapshot
  md BTC-USD --subscribe --trades                     - Live trade stream (tracked)
  md BTC-USD --subscribe --depth 5 --bids --offers    - Live L5 book updates
  md ETH-USD --snapshot --open --close --high --low --volume - OHLCV snapshot
  md BTC-USD --subscribe --open --close --high --low --volume - Live candle updates

Unsubscribe Examples:
  unsubscribe BTC-USD                                 - Cancel ALL BTC-USD subscriptions
  unsubscribe md_1757035274634111000                  - Cancel specific subscription by reqID
  unsubscribe --reqid md_1757035274634111000          - Cancel specific subscription (explicit)
  md BTC-USD --unsubscribe                            - Cancel ALL BTC-USD subscriptions
  status                                              - See active subscriptions with reqIDs

Subscription Management:
• Multiple subscriptions per symbol are supported (e.g., trades + order book)
• Auto-detection: inputs starting with "md_" are treated as reqIDs
• Symbol-based unsubscribe cancels ALL subscriptions for that symbol
• ReqID-based unsubscribe cancels only the specific subscription
• Use 'status' to see reqIDs for granular control
`)
}

func (a *FixApp) displaySnapshotTrades(trades []Trade, symbol string) {
	log.Printf("\n📋 Market Data Snapshot for %s:", symbol)

	// Group entries by type
	byType := make(map[string][]Trade)
	for _, trade := range trades {
		entryType := trade.EntryType
		if entryType == "" {
			entryType = "2" // Default to Trade if not specified
		}
		byType[entryType] = append(byType[entryType], trade)
	}

	// Display each type separately
	for entryType, entries := range byType {
		typeName := getMDEntryTypeName(entryType)
		log.Printf("\n🔹 %s Entries (%d):", typeName, len(entries))

		if entryType == constants.MdEntryTypeBid || entryType == constants.MdEntryTypeOffer {
			// Display bid/offer book format
			fmt.Printf("┌─────┬─────────────┬──────────────┬─────────────┬──────────┐\n")
			fmt.Printf("│ Pos │ Price       │ Size         │ Time        │ Type     │\n")
			fmt.Printf("├─────┼─────────────┼──────────────┼─────────────┼──────────┤\n")

			for _, entry := range entries {
				pos := entry.Position
				if pos == "" {
					pos = "-"
				}
				fmt.Printf("│ %-3s │ %-11s │ %-12s │ %-11s │ %-8s │\n",
					pos, entry.Price, entry.Size, entry.Time, typeName)
			}
			fmt.Printf("└─────┴─────────────┴──────────────┴─────────────┴──────────┘\n")

		} else if entryType == constants.MdEntryTypeTrade {
			// Display trade format
			fmt.Printf("┌─────┬─────────────┬──────────────┬─────────────┬───────────┐\n")
			fmt.Printf("│ #   │ Price       │ Size         │ Time        │ Aggressor │\n")
			fmt.Printf("├─────┼─────────────┼──────────────┼─────────────┼───────────┤\n")

			for i, entry := range entries {
				aggressor := entry.Aggressor
				if aggressor == "" {
					aggressor = "-"
				}
				fmt.Printf("│ %-3d │ %-11s │ %-12s │ %-11s │ %-9s │\n",
					i+1, entry.Price, entry.Size, entry.Time, aggressor)
			}
			fmt.Printf("└─────┴─────────────┴──────────────┴─────────────┴───────────┘\n")

		} else {
			// Display OHLC/Volume format (no size column - not relevant for these data types)
			fmt.Printf("┌─────┬─────────────┬─────────────┐\n")
			fmt.Printf("│ #   │ Value       │ Time        │\n")
			fmt.Printf("├─────┼─────────────┼─────────────┤\n")

			for i, entry := range entries {
				value := entry.Price
				if entryType == constants.MdEntryTypeVolume {
					value = entry.Size // For volume, the "size" field contains the volume
				}

				fmt.Printf("│ %-3d │ %-11s │ %-11s │\n",
					i+1, value, entry.Time)
			}
			fmt.Printf("└─────┴─────────────┴─────────────┘\n")
		}
	}

	log.Printf("\nTotal Entries Displayed: %d", len(trades))
}

func (a *FixApp) displayIncrementalTrades(trades []Trade) {
	for _, trade := range trades {
		a.TradeStore.DisplayRealtimeUpdate(trade)
	}
	// Add visual separator after each batch of incremental updates
	if len(trades) > 0 {
		log.Println("────────────────────────────────────────────────")
	}
}

func (a *FixApp) getSubscriptionTypeDesc(subType string) string {
	switch subType {
	case "0":
		return "Snapshot Only"
	case "1":
		return "Snapshot + Updates"
	case "2":
		return "Unsubscribe"
	default:
		return "Unknown"
	}
}

func getMarketDataTypeName(msgType string) string {
	switch msgType {
	case constants.MsgTypeMarketDataSnapshot:
		return "Snapshot"
	case constants.MsgTypeMarketDataIncremental:
		return "Incremental"
	default:
		return "Unknown"
	}
}

func getMDEntryTypeName(entryType string) string {
	switch entryType {
	case constants.MdEntryTypeBid:
		return "Bid"
	case constants.MdEntryTypeOffer:
		return "Offer"
	case constants.MdEntryTypeTrade:
		return "Trade"
	case constants.MdEntryTypeOpen:
		return "Open"
	case constants.MdEntryTypeClose:
		return "Close"
	case constants.MdEntryTypeHigh:
		return "High"
	case constants.MdEntryTypeLow:
		return "Low"
	case constants.MdEntryTypeVolume:
		return "Volume"
	default:
		return entryType
	}
}

func getAggressorSideDesc(side string) string {
	switch side {
	case "1":
		return "Buy"
	case "2":
		return "Sell"
	default:
		return side
	}
}
