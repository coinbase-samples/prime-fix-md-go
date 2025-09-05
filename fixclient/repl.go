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
				readline.PcItem("--snapshot", readline.PcItem("--trades"), readline.PcItem("--bids"), readline.PcItem("--offers")),
				readline.PcItem("--subscribe", readline.PcItem("--trades"), readline.PcItem("--bids"), readline.PcItem("--offers")),
			),
			readline.PcItem("ETH-USD",
				readline.PcItem("--snapshot", readline.PcItem("--trades"), readline.PcItem("--bids"), readline.PcItem("--offers")),
				readline.PcItem("--subscribe", readline.PcItem("--trades"), readline.PcItem("--bids"), readline.PcItem("--offers")),
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
			app.handleDirectMDRequest(parts)
		case "unsubscribe":
			app.handleUnsubscribeRequest(parts)
		case "status":
			app.handleStatusRequest()
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

type MDRequestFlags struct {
	subscriptionType string
	marketDepth      string
	entryTypes       []string
}

func (a *FixApp) handleDirectMDRequest(parts []string) {
	if len(parts) < 2 {
		fmt.Print(`Usage: md <symbol> [flags...]

Subscription Flags:
  --snapshot              - Snapshot only
  --subscribe             - Snapshot + live updates
  --unsubscribe           - Stop updates

Depth Flag:
  --depth N               - Market depth (0=full, 1=top, N=best N levels)

Entry Type Flags:
  --bids                  - Bid prices
  --offers                - Offer/ask prices
  --trades                - Executed trades
  --open                  - Opening price
  --close                 - Closing price
  --high                  - High price
  --low                   - Low price
  --volume                - Trading volume

Examples:
  md BTC-USD --snapshot --trades
  md BTC-USD --snapshot --depth 1 --bids --offers
  md BTC-USD --subscribe --depth 10 --bids --offers
  md ETH-USD --snapshot --open --close --high --low
  md BTC-USD --unsubscribe
`)
		return
	}

	symbol := strings.ToUpper(parts[1])
	flags := a.parseMDFlags(parts[2:])

	// Validate we have a subscription type
	if flags.subscriptionType == "" {
		fmt.Println("Error: Must specify subscription type (--snapshot, --subscribe, or --unsubscribe)")
		return
	}

	// For unsubscribe, we don't need depth or entry types
	if flags.subscriptionType == constants.SubscriptionRequestTypeUnsubscribe {
		a.sendUnsubscribeBySymbol(symbol)
		return
	}

	// Default depth to full if not specified
	if flags.marketDepth == "" {
		flags.marketDepth = "0"
	}

	// Default to trades if no entry types specified
	if len(flags.entryTypes) == 0 {
		flags.entryTypes = []string{constants.MdEntryTypeTrade}
	}

	// Determine description
	description := "Snapshot"
	if flags.subscriptionType == constants.SubscriptionRequestTypeSubscribe {
		description = "Live Subscription"
	}

	a.sendMarketDataRequestWithOptions(symbol, flags.subscriptionType, flags.marketDepth, flags.entryTypes, description)
}

func (a *FixApp) parseMDFlags(args []string) MDRequestFlags {
	flags := MDRequestFlags{
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

		// Entry type flags
		case "--bids":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeBid)
		case "--offers":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeOffer)
		case "--trades":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeTrade)
		case "--open":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeOpen)
		case "--close":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeClose)
		case "--high":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeHigh)
		case "--low":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeLow)
		case "--volume":
			flags.entryTypes = append(flags.entryTypes, constants.MdEntryTypeVolume)
		}
	}

	return flags
}

func (a *FixApp) handleUnsubscribeRequest(parts []string) {
	if len(parts) < 2 {
		fmt.Print(`Usage: unsubscribe <symbol|reqID>
Examples: 
  unsubscribe BTC-USD           - Cancel ALL BTC-USD subscriptions
  unsubscribe md_1234567890     - Cancel specific subscription by reqID
  unsubscribe --reqid md_123    - Cancel specific subscription (explicit)
`)
		return
	}

	// Handle --reqid flag for explicit reqID targeting
	if len(parts) >= 3 && parts[1] == "--reqid" {
		a.sendUnsubscribeByReqID(parts[2])
		return
	}

	input := parts[1]

	// Auto-detect: if input looks like reqID, treat as reqID; otherwise as symbol
	if strings.HasPrefix(input, "md_") {
		a.sendUnsubscribeByReqID(input)
	} else {
		symbol := strings.ToUpper(input)
		a.sendUnsubscribeBySymbol(symbol)
	}
}

func (a *FixApp) handleStatusRequest() {
	fmt.Printf("Session: %s ", a.SessionId)
	if a.SessionId.String() != "" {
		fmt.Println("(Connected)")
	} else {
		fmt.Println("(Disconnected)")
	}

	subscriptionsBySymbol := a.TradeStore.GetSubscriptionsBySymbol()
	if len(subscriptionsBySymbol) == 0 {
		fmt.Println("No active subscriptions")
		return
	}

	fmt.Print(`
Active Subscriptions:
┌─────────────┬──────────────────┬─────────────┬─────────────┬──────────────┬──────────────────┐
│ Symbol      │ Type             │ Status      │ Updates     │ Last Update  │ ReqID            │
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

			// Truncate reqID for display
			shortReqID := sub.MDReqID
			if len(shortReqID) > 16 {
				shortReqID = "..." + shortReqID[len(shortReqID)-13:]
			}

			fmt.Printf("│ %-11s │ %-16s │ %-11s │ %-11d │ %-12s │ %-16s │\n",
				displaySymbol, a.getSubscriptionTypeDesc(sub.SubscriptionType), status, sub.TotalUpdates, lastUpdate, shortReqID)
		}
	}

	fmt.Println("└─────────────┴──────────────────┴─────────────┴─────────────┴──────────────┴──────────────────┘")
}
