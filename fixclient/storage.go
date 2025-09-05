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
	"time"

	"prime-fix-md-go/constants"
)

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
		case constants.MdEntryTypeBid: // "0"
			posInt, _ := strconv.Atoi(trade.Position)
			err = a.DB.StoreOrderBookBatch(tx, trade.Symbol, "bid", trade.Price, trade.Size,
				posInt, seqNumInt, trade.MDReqID, isSnapshot)
		case constants.MdEntryTypeOffer: // "1"
			posInt, _ := strconv.Atoi(trade.Position)
			err = a.DB.StoreOrderBookBatch(tx, trade.Symbol, "offer", trade.Price, trade.Size,
				posInt, seqNumInt, trade.MDReqID, isSnapshot)
		case constants.MdEntryTypeTrade: // "2"
			err = a.DB.StoreTradeBatch(tx, trade.Symbol, trade.Price, trade.Size,
				trade.Aggressor, trade.Time, seqNumInt, trade.MDReqID, isSnapshot)
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
	var hasBook bool

	for _, entryType := range entryTypes {
		switch entryType {
		case constants.MdEntryTypeBid, constants.MdEntryTypeOffer:
			if dataTypes == "" {
				dataTypes = "order_book"
				hasBook = true
			}
		case constants.MdEntryTypeTrade:
			if dataTypes == "" {
				dataTypes = "trades"
			}
		case constants.MdEntryTypeOpen, constants.MdEntryTypeClose,
			constants.MdEntryTypeHigh, constants.MdEntryTypeLow, constants.MdEntryTypeVolume:
			if dataTypes == "" {
				dataTypes = "ohlcv"
			}
		}
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
