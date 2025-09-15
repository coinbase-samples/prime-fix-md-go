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
	if a.Db == nil {
		return
	}

	seqNumInt, _ := strconv.Atoi(seqNum)

	tx, err := a.Db.BeginTransaction()
	if err != nil {
		log.Printf("Failed to begin database transaction: %v", err)
		return
	}
	defer tx.Rollback()

	for _, trade := range trades {
		switch trade.EntryType {
		case constants.MdEntryTypeBid: // "0"
			posInt, _ := strconv.Atoi(trade.Position)
			err = a.Db.StoreOrderBookBatch(tx, trade.Symbol, "bid", trade.Price, trade.Size,
				posInt, seqNumInt, trade.MdReqId, isSnapshot)
		case constants.MdEntryTypeOffer: // "1"
			posInt, _ := strconv.Atoi(trade.Position)
			err = a.Db.StoreOrderBookBatch(tx, trade.Symbol, "offer", trade.Price, trade.Size,
				posInt, seqNumInt, trade.MdReqId, isSnapshot)
		case constants.MdEntryTypeTrade: // "2"
			err = a.Db.StoreTradeBatch(tx, trade.Symbol, trade.Price, trade.Size,
				trade.Aggressor, trade.Time, seqNumInt, trade.MdReqId, isSnapshot)
		case constants.MdEntryTypeOpen: // "4"
			err = a.Db.StoreOhlcvBatch(tx, trade.Symbol, "open", trade.Price, trade.Time,
				seqNumInt, trade.MdReqId)
		case constants.MdEntryTypeClose: // "5"
			err = a.Db.StoreOhlcvBatch(tx, trade.Symbol, "close", trade.Price, trade.Time,
				seqNumInt, trade.MdReqId)
		case constants.MdEntryTypeHigh: // "7"
			err = a.Db.StoreOhlcvBatch(tx, trade.Symbol, "high", trade.Price, trade.Time,
				seqNumInt, trade.MdReqId)
		case constants.MdEntryTypeLow: // "8"
			err = a.Db.StoreOhlcvBatch(tx, trade.Symbol, "low", trade.Price, trade.Time,
				seqNumInt, trade.MdReqId)
		case constants.MdEntryTypeVolume: // "B"
			err = a.Db.StoreOhlcvBatch(tx, trade.Symbol, "volume", trade.Size, trade.Time,
				seqNumInt, trade.MdReqId)
		}

		if err != nil {
			log.Printf("Failed to store %s data to database: %v", getMdEntryTypeName(trade.EntryType), err)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit database transaction: %v", err)
	}
}

func (a *FixApp) createDatabaseSession(symbol, subscriptionType, marketDepth string, entryTypes []string, reqId string) {
	if a.Db == nil {
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
	err := a.Db.CreateSession(sessionId, symbol, requestType, dataTypes, reqId, depth)
	if err != nil {
		log.Printf("Failed to create session record: %v", err)
	}
}
