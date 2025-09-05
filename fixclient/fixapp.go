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
	"strconv"
	"strings"
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

func (a *FixApp) extractTrades(msg *quickfix.Message, symbol, mdReqID string, isSnapshot bool, seqNum string) []Trade {
	return a.extractTradesImproved(msg, symbol, mdReqID, isSnapshot, seqNum)
}

func (a *FixApp) extractTradesImproved(msg *quickfix.Message, symbol, mdReqID string, isSnapshot bool, seqNum string) []Trade {
	rawMsg := msg.String()

	noMDEntriesStr := utils.GetString(msg, constants.TagNoMdEntries)
	if noMDEntriesStr == "" || noMDEntriesStr == "0" {
		return []Trade{}
	}

	entryStarts := a.findEntryBoundaries(rawMsg)

	var trades []Trade
	for i, startPos := range entryStarts {
		endPos := a.getEntryEndPos(entryStarts, i, len(rawMsg))
		entrySegment := rawMsg[startPos:endPos]

		trade := a.parseTradeFromSegment(entrySegment, symbol, mdReqID, isSnapshot, seqNum, i)
		trades = append(trades, trade)
	}

	return trades
}

func (a *FixApp) findEntryBoundaries(rawMsg string) []int {
	var entryStarts []int
	searchFrom := 0
	for {
		pos := strings.Index(rawMsg[searchFrom:], "269=")
		if pos == -1 {
			break
		}
		entryStarts = append(entryStarts, searchFrom+pos)
		searchFrom += pos + 4
	}
	return entryStarts
}

func (a *FixApp) getEntryEndPos(entryStarts []int, currentIndex, msgLen int) int {
	if currentIndex < len(entryStarts)-1 {
		return entryStarts[currentIndex+1]
	}
	return msgLen
}

func (a *FixApp) parseTradeFromSegment(segment, symbol, mdReqID string, isSnapshot bool, seqNum string, entryIndex int) Trade {
	trade := Trade{
		Timestamp:  time.Now(),
		Symbol:     symbol,
		MDReqID:    mdReqID,
		IsSnapshot: isSnapshot,
		IsUpdate:   !isSnapshot,
		SeqNum:     seqNum,
	}

	if entryType := extractSingleFieldValue(segment, "269="); entryType != "" {
		trade.EntryType = entryType
	}
	if price := extractSingleFieldValue(segment, "270="); price != "" {
		trade.Price = price
	}
	if size := extractSingleFieldValue(segment, "271="); size != "" {
		trade.Size = size
	}
	if timeVal := extractSingleFieldValue(segment, "273="); timeVal != "" {
		trade.Time = timeVal
	}

	if position := extractSingleFieldValue(segment, "290="); position != "" {
		trade.Position = position
	} else {
		if trade.EntryType == "0" || trade.EntryType == "1" { // Bids or Offers
			trade.Position = fmt.Sprintf("%d", entryIndex+1)
		}
	}

	if aggressor := extractSingleFieldValue(segment, "2446="); aggressor != "" {
		trade.Aggressor = getAggressorSideDesc(aggressor)
	}

	return trade
}

func (a *FixApp) storeTradesToDatabase(trades []Trade, seqNum string, isSnapshot bool) {
	if a.DB == nil {
		return
	}

	seqNumInt, _ := strconv.Atoi(seqNum)

	tx, err := a.DB.BeginTransaction()
	if err != nil {
		log.Printf("Failed to begin database transaction: %v", err)
		return
	}
	defer tx.Rollback()

	for _, trade := range trades {
		switch trade.EntryType {
		case constants.MdEntryTypeTrade: // "2"
			err = a.DB.StoreTradeBatch(tx, trade.Symbol, trade.Price, trade.Size,
				trade.Aggressor, trade.Time, seqNumInt, trade.MDReqID, isSnapshot)
		case constants.MdEntryTypeBid: // "0"
			posInt, _ := strconv.Atoi(trade.Position)
			err = a.DB.StoreOrderBookBatch(tx, trade.Symbol, "bid", trade.Price, trade.Size,
				posInt, seqNumInt, trade.MDReqID, isSnapshot)
		case constants.MdEntryTypeOffer: // "1"
			posInt, _ := strconv.Atoi(trade.Position)
			err = a.DB.StoreOrderBookBatch(tx, trade.Symbol, "offer", trade.Price, trade.Size,
				posInt, seqNumInt, trade.MDReqID, isSnapshot)
		case constants.MdEntryTypeOpen: // "4"
			err = a.DB.StoreOHLCVBatch(tx, trade.Symbol, "open", trade.Price, trade.Time,
				seqNumInt, trade.MDReqID)
		case constants.MdEntryTypeClose: // "5"
			err = a.DB.StoreOHLCVBatch(tx, trade.Symbol, "close", trade.Price, trade.Time,
				seqNumInt, trade.MDReqID)
		case constants.MdEntryTypeHigh: // "7"
			err = a.DB.StoreOHLCVBatch(tx, trade.Symbol, "high", trade.Price, trade.Time,
				seqNumInt, trade.MDReqID)
		case constants.MdEntryTypeLow: // "8"
			err = a.DB.StoreOHLCVBatch(tx, trade.Symbol, "low", trade.Price, trade.Time,
				seqNumInt, trade.MDReqID)
		case constants.MdEntryTypeVolume: // "B"
			err = a.DB.StoreOHLCVBatch(tx, trade.Symbol, "volume", trade.Size, trade.Time,
				seqNumInt, trade.MDReqID)
		}

		if err != nil {
			log.Printf("Failed to store %s data to database: %v", getMDEntryTypeName(trade.EntryType), err)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit database transaction: %v", err)
	}
}

func (a *FixApp) createDatabaseSession(symbol, subscriptionType, marketDepth string, entryTypes []string, reqID string) {
	if a.DB == nil {
		return
	}

	requestType := "snapshot"
	if subscriptionType == constants.SubscriptionRequestTypeSubscribe {
		requestType = "subscribe"
	}

	var dataTypes string
	hasBook := false
	hasTrades := false
	hasOhlcv := false

	for _, entryType := range entryTypes {
		switch entryType {
		case constants.MdEntryTypeBid, constants.MdEntryTypeOffer:
			hasBook = true
		case constants.MdEntryTypeTrade:
			hasTrades = true
		case constants.MdEntryTypeOpen, constants.MdEntryTypeClose,
			constants.MdEntryTypeHigh, constants.MdEntryTypeLow, constants.MdEntryTypeVolume:
			hasOhlcv = true
		}
	}

	if hasBook {
		dataTypes = "bids,offers"
	} else if hasTrades {
		dataTypes = "trades"
	} else if hasOhlcv {
		dataTypes = "ohlcv"
	}

	var depth *int
	if hasBook && marketDepth != "0" {
		if d, err := strconv.Atoi(marketDepth); err == nil {
			depth = &d
		}
	}

	sessionId := fmt.Sprintf("%s_%s_%d", symbol, requestType, time.Now().Unix())
	err := a.DB.CreateSession(sessionId, symbol, requestType, dataTypes, reqID, depth)
	if err != nil {
		log.Printf("Failed to create session record: %v", err)
	}
}

func extractSingleFieldValue(fixSegment, tagPrefix string) string {
	start := strings.Index(fixSegment, tagPrefix)
	if start == -1 {
		return ""
	}

	start += len(tagPrefix)
	end := strings.Index(fixSegment[start:], "\x01") // FIX field delimiter
	if end == -1 {
		return fixSegment[start:]
	}

	return fixSegment[start : start+end]
}

func (a *FixApp) sendUnsubscribeBySymbol(symbol string) {
	subscriptions := a.TradeStore.GetSubscriptionStatus()

	var symbolSubs []*Subscription
	for _, sub := range subscriptions {
		if sub.Symbol == symbol {
			symbolSubs = append(symbolSubs, sub)
		}
	}

	if len(symbolSubs) == 0 {
		fmt.Printf("No active subscriptions found for %s\n", symbol)
		return
	}

	if len(symbolSubs) > 1 {
		fmt.Printf("Multiple active subscriptions for %s:\n", symbol)
		for i, sub := range symbolSubs {
			fmt.Printf("  %d. ReqID: %s, Type: %s, Updates: %d\n",
				i+1, sub.MDReqID, a.getSubscriptionTypeDesc(sub.SubscriptionType), sub.TotalUpdates)
		}
		fmt.Printf("Unsubscribing from all %d subscriptions for %s\n", len(symbolSubs), symbol)
	}

	for _, sub := range symbolSubs {
		msg := builder.BuildMarketDataRequest(
			sub.MDReqID,
			symbol,
			constants.SubscriptionRequestTypeUnsubscribe,
			"0",
			a.Config.SenderCompId,
			a.Config.TargetCompId,
			[]string{constants.MdEntryTypeTrade},
		)

		if err := quickfix.Send(msg); err != nil {
			log.Printf("Error sending unsubscribe request for reqID %s: %v", sub.MDReqID, err)
		} else {
			fmt.Printf("Unsubscribe request sent for %s (reqID: %s)\n", symbol, sub.MDReqID)
			a.TradeStore.RemoveSubscriptionByReqID(sub.MDReqID)
		}
	}
}

func (a *FixApp) sendUnsubscribeByReqID(reqID string) {
	subscriptions := a.TradeStore.GetSubscriptionStatus()

	sub, exists := subscriptions[reqID]
	if !exists {
		fmt.Printf("No active subscription found with reqID: %s\n", reqID)
		return
	}

	msg := builder.BuildMarketDataRequest(
		reqID,
		sub.Symbol,
		constants.SubscriptionRequestTypeUnsubscribe,
		"0",
		a.Config.SenderCompId,
		a.Config.TargetCompId,
		[]string{constants.MdEntryTypeTrade},
	)

	if err := quickfix.Send(msg); err != nil {
		log.Printf("Error sending unsubscribe request for reqID %s: %v", reqID, err)
		fmt.Printf("Failed to send unsubscribe request for reqID: %s\n", reqID)
	} else {
		fmt.Printf("Unsubscribe request sent for %s (reqID: %s)\n", sub.Symbol, reqID)
		a.TradeStore.RemoveSubscriptionByReqID(reqID)
	}
}

func (a *FixApp) sendMarketDataRequest(symbol, subscriptionType, description string) {
	a.sendMarketDataRequestWithOptions(symbol, subscriptionType, "0", []string{constants.MdEntryTypeTrade}, description)
}

func (a *FixApp) sendMarketDataRequestWithOptions(symbol, subscriptionType, marketDepth string, entryTypes []string, description string) {
	reqID := fmt.Sprintf("md_%d", time.Now().UnixNano())

	if subscriptionType == constants.SubscriptionRequestTypeSubscribe {
		a.TradeStore.AddSubscription(symbol, subscriptionType, reqID)
	}

	a.createDatabaseSession(symbol, subscriptionType, marketDepth, entryTypes, reqID)

	msg := builder.BuildMarketDataRequest(
		reqID,
		symbol,
		subscriptionType,
		marketDepth,
		a.Config.SenderCompId,
		a.Config.TargetCompId,
		entryTypes,
	)

	if err := quickfix.Send(msg); err != nil {
		log.Printf("Error sending market data request: %v", err)
		fmt.Printf("Failed to send %s request for %s\n", description, symbol)
		a.TradeStore.RemoveSubscription(symbol)
	} else {
		entryTypesStr := ""
		for i, et := range entryTypes {
			if i > 0 {
				entryTypesStr += ", "
			}
			entryTypesStr += getMDEntryTypeName(et)
		}
		fmt.Printf("%s request sent for %s (depth=%s, types=[%s], reqID=%s)\n",
			description, symbol, marketDepth, entryTypesStr, reqID)
	}
}
