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
	"strings"

	"prime-fix-md-go/constants"
	"prime-fix-md-go/utils"

	"github.com/chzyer/readline"
)

func Repl(app *FixApp) {
	// Setup readline with command completion
	completer := readline.NewPrefixCompleter(
		readline.PcItem("md",
			readline.PcItem("BTC-USD",
				readline.PcItem("--snapshot", readline.PcItem("--trades"), readline.PcItem("--depth")),
				readline.PcItem("--subscribe", readline.PcItem("--trades"), readline.PcItem("--depth")),
			),
			readline.PcItem("ETH-USD",
				readline.PcItem("--snapshot", readline.PcItem("--trades"), readline.PcItem("--depth")),
				readline.PcItem("--subscribe", readline.PcItem("--trades"), readline.PcItem("--depth")),
			),
		),
		readline.PcItem("unsubscribe", readline.PcItem("BTC-USD"), readline.PcItem("ETH-USD")),
		readline.PcItem("status"),
		readline.PcItem("help"),
		readline.PcItem("version"),
		readline.PcItem("exit"),
	)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "FIX-MD> ",
		HistoryFile:     "/tmp/fixmd_history",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		log.Printf("Failed to create readline: %v", err)
		return
	}
	defer rl.Close()

	for {
		if app.ShouldExit() {
			fmt.Println("Exiting due to authentication failures. Please check your credentials.")
			return
		}

		line, err := rl.Readline()
		if err != nil {
			break
		}

		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToLower(parts[0])
		switch cmd {
		case "md":
			app.handleDirectMdRequest(parts)
		case "unsubscribe":
			app.handleUnsubscribeRequest(parts)
		case "status":
			if !app.handleStatusRequest() {
				return
			}
		case "help":
			app.displayHelp()
		case "version":
			fmt.Println(utils.FullVersion())
		case "exit":
			return
		default:
			fmt.Println("Unknown command. Type 'help' for available commands.")
		}
	}
}

type MdRequestFlags struct {
	subscriptionType string
	marketDepth      string
	entryTypes       []string
}

func (a *FixApp) handleDirectMdRequest(parts []string) {
	if len(parts) < 2 {
		fmt.Print(`Usage: md <symbol1> [symbol2 symbol3 ...] [flags...]

Subscription Flags:
  --snapshot              - Snapshot only
  --subscribe             - Snapshot + live updates
  --unsubscribe           - Stop updates

Depth Flag:
  --depth N               - Market depth (0=full, 1=top, N=best N levels)
                            Automatically includes both bids and offers

Entry Type Flags:
  --trades                - Executed trades
  --o                     - Opening price
  --c                     - Closing price
  --h                     - High price
  --l                     - Low price
  --v                     - Trading volume

Examples:
  md BTC-USD --snapshot --trades
  md BTC-USD ETH-USD --snapshot --depth 1
  md BTC-USD ETH-USD SOL-USD --subscribe --depth 10
  md ETH-USD --snapshot --o --c --h --l --v
  md BTC-USD --unsubscribe
`)
		return
	}

	// Parse symbols and flags
	var symbols []string
	var flagStart int

	// Find where flags start (first argument starting with --)
	for i, part := range parts[1:] {
		if strings.HasPrefix(part, "--") {
			flagStart = i + 1 // offset since we skipped parts[0]
			break
		}
		symbols = append(symbols, strings.ToUpper(part))
	}

	// If no flags found, all remaining parts are symbols
	if flagStart == 0 {
		flagStart = len(parts)
	}

	// Parse flags from flagStart onwards
	var flagArgs []string
	if flagStart < len(parts) {
		flagArgs = parts[flagStart:]
	}

	flags := a.parseMdFlags(flagArgs)

	// Validate we have a subscription type
	if flags.subscriptionType == "" {
		fmt.Println("Error: Must specify subscription type (--snapshot, --subscribe, or --unsubscribe)")
		return
	}

	// For unsubscribe, we don't need depth or entry types
	if flags.subscriptionType == constants.SubscriptionRequestTypeUnsubscribe {
		for _, symbol := range symbols {
			a.sendUnsubscribeBySymbol(symbol)
		}
		return
	}

	// Default depth to full if not specified
	if flags.marketDepth == "" {
		flags.marketDepth = "0"
	}

	// Default entry types based on context
	if len(flags.entryTypes) == 0 {
		// If depth is specified, default to bids and offers (order book data)
		if flags.marketDepth != "" {
			flags.entryTypes = []string{constants.MdEntryTypeBid, constants.MdEntryTypeOffer}
		} else {
			// Otherwise default to trades
			flags.entryTypes = []string{constants.MdEntryTypeTrade}
		}
	}

	// Determine description
	description := "Snapshot"
	if flags.subscriptionType == constants.SubscriptionRequestTypeSubscribe {
		description = "Live Subscription"
	}

	a.sendMarketDataRequestWithOptions(symbols, flags.subscriptionType, flags.marketDepth, flags.entryTypes, description)
}

func (a *FixApp) parseMdFlags(args []string) MdRequestFlags {
	flags := MdRequestFlags{
		entryTypes: []string{},
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		// Subscription type flags
		case "--snapshot":
			flags.subscriptionType = constants.SubscriptionRequestTypeSnapshot
		case "--subscribe":
			flags.subscriptionType = constants.SubscriptionRequestTypeSubscribe
		case "--unsubscribe":
			flags.subscriptionType = constants.SubscriptionRequestTypeUnsubscribe

		// Depth flag (requires next argument)
		case "--depth":
			if i+1 < len(args) {
				i++
				flags.marketDepth = args[i]
			}

		case "--trades":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeTrade)
		case "--o":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeOpen)
		case "--c":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeClose)
		case "--h":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeHigh)
		case "--l":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeLow)
		case "--v":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeVolume)
		}
	}

	return flags
}

func (a *FixApp) handleUnsubscribeRequest(parts []string) {
	if len(parts) < 2 {
		fmt.Print(`Usage: unsubscribe <symbol|reqId>
Examples: 
  unsubscribe BTC-USD           - Cancel ALL BTC-USD subscriptions
  unsubscribe md_1234567890     - Cancel specific subscription by reqId
  unsubscribe --reqid md_123    - Cancel specific subscription (explicit)
`)
		return
	}

	// Handle --reqid flag for explicit reqId targeting
	if len(parts) >= 3 && parts[1] == "--reqid" {
		a.sendUnsubscribeByReqId(parts[2])
		return
	}

	input := parts[1]

	// Auto-detect: if input looks like reqId, treat as reqId; otherwise as symbol
	if strings.HasPrefix(input, "md_") {
		a.sendUnsubscribeByReqId(input)
	} else {
		symbol := strings.ToUpper(input)
		a.sendUnsubscribeBySymbol(symbol)
	}
}

func (a *FixApp) handleStatusRequest() bool {
	if a.ShouldExit() {
		fmt.Println("Exiting due to authentication failures. Please check your credentials.")
		return false
	}

	fmt.Printf("Session: %s ", a.SessionId)
	if a.SessionId.String() != "" {
		fmt.Println("(Connected)")
	} else {
		fmt.Println("(Disconnected)")
	}

	subscriptionsBySymbol := a.TradeStore.GetSubscriptionsBySymbol()
	if len(subscriptionsBySymbol) == 0 {
		fmt.Println("No active subscriptions")
		return true
	}

	fmt.Print(`
Active Subscriptions:
┌─────────────┬──────────────────┬─────────────┬─────────────┬──────────────┬──────────────────┐
│ Symbol      │ Type             │ Status      │ Updates     │ Last Update  │ ReqId            │
├─────────────┼──────────────────┼─────────────┼─────────────┼──────────────┼──────────────────┤
`)

	for symbol, subs := range subscriptionsBySymbol {
		for i, sub := range subs {
			status := "Active"
			if !sub.Active {
				status = "Inactive"
			}

			lastUpdate := "Never"
			if !sub.LastUpdate.IsZero() {
				lastUpdate = sub.LastUpdate.Format("15:04:05")
			}

			// Show symbol only on first line for multiple subscriptions
			displaySymbol := symbol
			if i > 0 {
				displaySymbol = ""
			}

			// Truncate reqId for display
			shortReqId := sub.MdReqId
			if len(shortReqId) > 16 {
				shortReqId = "..." + shortReqId[len(shortReqId)-13:]
			}

			fmt.Printf("│ %-11s │ %-16s │ %-11s │ %-11d │ %-12s │ %-16s │\n",
				displaySymbol, a.getSubscriptionTypeDesc(sub.SubscriptionType), status, sub.TotalUpdates, lastUpdate, shortReqId)
		}
	}

	fmt.Println("└─────────────┴──────────────────┴─────────────┴─────────────┴──────────────┴──────────────────┘")

	return true
}
