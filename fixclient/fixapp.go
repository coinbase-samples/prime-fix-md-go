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
	"time"

	"prime-fix-md-go/builder"
	"prime-fix-md-go/constants"
	"prime-fix-md-go/database"
	"prime-fix-md-go/utils"

	"github.com/quickfixgo/quickfix"
)

type Config struct {
	ApiKey       string
	ApiSecret    string
	Passphrase   string
	SenderCompId string
	TargetCompId string
	PortfolioId  string
}

type FixApp struct {
	Config *Config

	SessionId  quickfix.SessionID
	TradeStore *TradeStore
	DB         *database.MarketDataDB
}

func NewConfig(apiKey, apiSecret, passphrase, senderCompId, targetCompId, portfolioId string) *Config {
	return &Config{
		ApiKey:       apiKey,
		ApiSecret:    apiSecret,
		Passphrase:   passphrase,
		SenderCompId: senderCompId,
		TargetCompId: targetCompId,
		PortfolioId:  portfolioId,
	}
}

func NewFixApp(config *Config, db *database.MarketDataDB) *FixApp {
	tradeStore := NewTradeStore(10000, "")

	return &FixApp{
		Config:     config,
		TradeStore: tradeStore,
		DB:         db,
	}
}

func (a *FixApp) OnCreate(sid quickfix.SessionID) {
	a.SessionId = sid
}

func (a *FixApp) OnLogout(sid quickfix.SessionID) {
	log.Println("Logout", sid)
}

func (a *FixApp) FromAdmin(_ *quickfix.Message, _ quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}

func (a *FixApp) ToApp(_ *quickfix.Message, _ quickfix.SessionID) error {
	return nil
}

func (a *FixApp) OnLogon(sid quickfix.SessionID) {
	a.SessionId = sid
	log.Println("✓ FIX logon", sid)
	fmt.Print("Connected! Market data connection established.\n\n")
	a.displayHelp()
}

func (a *FixApp) ToAdmin(msg *quickfix.Message, _ quickfix.SessionID) {
	if t, _ := msg.Header.GetString(constants.TagMsgType); t == constants.MsgTypeLogon {
		ts := time.Now().UTC().Format(constants.FixTimeFormat)
		builder.BuildLogon(
			&msg.Body,
			ts,
			a.Config.ApiKey,
			a.Config.ApiSecret,
			a.Config.Passphrase,
			a.Config.TargetCompId,
			a.Config.PortfolioId,
		)
	}
}

func (a *FixApp) FromApp(msg *quickfix.Message, _ quickfix.SessionID) quickfix.MessageRejectError {
	if t, _ := msg.Header.GetString(constants.TagMsgType); t == constants.MsgTypeMarketDataSnapshot || t == constants.MsgTypeMarketDataIncremental {
		a.handleMarketDataMessage(msg)
	} else if t == "Y" { // Market Data Request Reject
		a.handleMarketDataReject(msg)
	} else {
		log.Printf("Received application message type %s", t)
	}
	return nil
}

func (a *FixApp) handleMarketDataReject(msg *quickfix.Message) {
	mdReqID := utils.GetString(msg, constants.TagMdReqId)
	rejReason := utils.GetString(msg, constants.TagMdReqRejReason)
	text := utils.GetString(msg, constants.TagText)

	reasonDesc := getMDReqRejReasonDesc(rejReason)

	log.Printf("Market Data Request REJECTED")
	log.Printf("   MDReqID: %s", mdReqID)
	log.Printf("   Reason: %s (%s)", rejReason, reasonDesc)
	if text != "" {
		log.Printf("   Text: %s", text)
	}

	a.TradeStore.RemoveSubscriptionByReqID(mdReqID)

	switch rejReason {
	case "0":
		log.Printf("Try a different symbol format (e.g., BTCUSD vs BTC-USD)")
	case "3":
		log.Printf("Check if your account has market data permissions")
	case "5":
		log.Printf("Try MarketDepth=0 (full depth) or MarketDepth=1 (top of book)")
	case "8":
		log.Printf("Try different MDEntryType: 0=Bids, 1=Offers, 2=Trades")
	}
}

func getMDReqRejReasonDesc(reason string) string {
	switch reason {
	case "0":
		return "Unknown symbol"
	case "1":
		return "Duplicate MDReqID"
	case "2":
		return "Insufficient bandwidth"
	case "3":
		return "Insufficient permission"
	case "4":
		return "Invalid SubscriptionRequestType"
	case "5":
		return "Invalid MarketDepth"
	case "6":
		return "Unsupported MDUpdateType"
	case "7":
		return "Other"
	case "8":
		return "Unsupported MDEntryType"
	default:
		return "Unknown reason"
	}
}

func (a *FixApp) handleMarketDataMessage(msg *quickfix.Message) {
	msgType, _ := msg.Header.GetString(constants.TagMsgType)
	mdReqID := utils.GetString(msg, constants.TagMdReqId)
	symbol := utils.GetString(msg, constants.TagSymbol)
	noMDEntries := utils.GetString(msg, constants.TagNoMdEntries)
	seqNum, _ := msg.Header.GetString(constants.TagMsgSeqNum)

	isSnapshot := msgType == constants.MsgTypeMarketDataSnapshot
	isIncremental := msgType == constants.MsgTypeMarketDataIncremental

	log.Printf("Market Data %s for %s (ReqID: %s, Entries: %s, Seq: %s)",
		getMarketDataTypeName(msgType), symbol, mdReqID, noMDEntries, seqNum)

	trades := a.extractTrades(msg, symbol, mdReqID, isSnapshot, seqNum)

	a.TradeStore.AddTrades(symbol, trades, isSnapshot, mdReqID)

	a.storeTradesToDatabase(trades, seqNum, isSnapshot)

	if isSnapshot {
		a.displaySnapshotTrades(trades, symbol)
	} else if isIncremental {
		a.displayIncrementalTrades(trades)
	}
}
