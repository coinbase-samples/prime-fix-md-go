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

package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type MarketDataDb struct {
	db *sql.DB
}

func NewMarketDataDb(dbPath string) (*MarketDataDb, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	mdb := &MarketDataDb{db: db}
	if err := mdb.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}

	log.Printf("SQLite database initialized at %s", dbPath)
	return mdb, nil
}

func (mdb *MarketDataDb) Close() error {
	return mdb.db.Close()
}

// Session management
func (mdb *MarketDataDb) CreateSession(sessionId, symbol, requestType, dataTypes, mdReqId string, depth *int) error {
	_, err := mdb.db.Exec(insertSessionQuery, sessionId, symbol, requestType, dataTypes, depth, mdReqId)
	return err
}

// Trade data storage
func (mdb *MarketDataDb) StoreTrade(symbol, price, size, aggressorSide, tradeTime string, seqNum int, mdReqId string, isSnapshot bool) error {
	_, err := mdb.db.Exec(insertTradeQuery, symbol, price, size, aggressorSide, tradeTime, seqNum, mdReqId, isSnapshot)
	return err
}

// Order book data storage
func (mdb *MarketDataDb) StoreOrderBookEntry(symbol, side, price, size string, position, seqNum int, mdReqId string, isSnapshot bool) error {
	_, err := mdb.db.Exec(insertOrderBookQuery, symbol, side, price, size, position, seqNum, mdReqId, isSnapshot)
	return err
}

// OHLCV data storage
func (mdb *MarketDataDb) StoreOHLCV(symbol, dataType, value, entryTime string, seqNum int, mdReqId string) error {
	_, err := mdb.db.Exec(insertOHLCVQuery, symbol, dataType, value, entryTime, seqNum, mdReqId)
	return err
}

// Batch operations for better performance
func (mdb *MarketDataDb) BeginTransaction() (*sql.Tx, error) {
	return mdb.db.Begin()
}

func (mdb *MarketDataDb) StoreTradeBatch(tx *sql.Tx, symbol, price, size, aggressorSide, tradeTime string, seqNum int, mdReqId string, isSnapshot bool) error {
	_, err := tx.Exec(insertTradeQuery, symbol, price, size, aggressorSide, tradeTime, seqNum, mdReqId, isSnapshot)
	return err
}

func (mdb *MarketDataDb) StoreOrderBookBatch(tx *sql.Tx, symbol, side, price, size string, position, seqNum int, mdReqId string, isSnapshot bool) error {
	_, err := tx.Exec(insertOrderBookQuery, symbol, side, price, size, position, seqNum, mdReqId, isSnapshot)
	return err
}

func (mdb *MarketDataDb) StoreOhlcvBatch(tx *sql.Tx, symbol, dataType, value, entryTime string, seqNum int, mdReqId string) error {
	_, err := tx.Exec(insertOHLCVQuery, symbol, dataType, value, entryTime, seqNum, mdReqId)
	return err
}
