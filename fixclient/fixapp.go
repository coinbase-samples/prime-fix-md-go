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
	Db         *database.MarketDataDb
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

func NewFixApp(config *Config, db *database.MarketDataDb) *FixApp {
	tradeStore := NewTradeStore(10000, "")

	return &FixApp{
		Config:     config,
		TradeStore: tradeStore,
		Db:         db,
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
	a.displayConnectionSuccess()
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
	mdReqId := utils.GetString(msg, constants.TagMdReqId)
	rejReason := utils.GetString(msg, constants.TagMdReqRejReason)
	text := utils.GetString(msg, constants.TagText)

	reasonDesc := getMdReqRejReasonDesc(rejReason)

	a.displayMarketDataReject(mdReqId, rejReason, reasonDesc, text)
	a.TradeStore.RemoveSubscriptionByReqId(mdReqId)
	a.displayMarketDataRejectHelp(rejReason)
}

func getMdReqRejReasonDesc(reason string) string {
	switch reason {
	case constants.MdReqRejReasonUnknownSymbol:
		return "Unknown symbol"
	case constants.MdReqRejReasonDuplicateMdReqId:
		return "Duplicate MdReqId"
	case constants.MdReqRejReasonInsufficientBandwidth:
		return "Insufficient bandwidth"
	case constants.MdReqRejReasonInsufficientPermission:
		return "Insufficient permission"
	case constants.MdReqRejReasonInvalidSubscriptionReqType:
		return "Invalid SubscriptionRequestType"
	case constants.MdReqRejReasonInvalidMarketDepth:
		return "Invalid MarketDepth"
	case constants.MdReqRejReasonUnsupportedMdUpdateType:
		return "Unsupported MdUpdateType"
	case constants.MdReqRejReasonOther:
		return "Other"
	case constants.MdReqRejReasonUnsupportedMdEntryType:
		return "Unsupported MdEntryType"
	default:
		return "Unknown reason"
	}
}

func (a *FixApp) handleMarketDataMessage(msg *quickfix.Message) {
	msgType, _ := msg.Header.GetString(constants.TagMsgType)
	mdReqId := utils.GetString(msg, constants.TagMdReqId)
	symbol := utils.GetString(msg, constants.TagSymbol)
	noMdEntries := utils.GetString(msg, constants.TagNoMdEntries)
	seqNum, _ := msg.Header.GetString(constants.TagMsgSeqNum)

	isSnapshot := msgType == constants.MsgTypeMarketDataSnapshot
	isIncremental := msgType == constants.MsgTypeMarketDataIncremental

	a.displayMarketDataReceived(msgType, symbol, mdReqId, noMdEntries, seqNum)

	trades := a.extractTrades(msg, symbol, mdReqId, isSnapshot, seqNum)

	a.TradeStore.AddTrades(symbol, trades, isSnapshot, mdReqId)

	a.storeTradesToDatabase(trades, seqNum, isSnapshot)

	if isSnapshot {
		a.displaySnapshotTrades(trades, symbol)
	} else if isIncremental {
		a.displayIncrementalTrades(trades)
	}
}
