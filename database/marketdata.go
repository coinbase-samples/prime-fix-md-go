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

type MarketDataDB struct {
	db *sql.DB
}

func NewMarketDataDB(dbPath string) (*MarketDataDB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	mdb := &MarketDataDB{db: db}
	if err := mdb.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}

	log.Printf("SQLite database initialized at %s", dbPath)
	return mdb, nil
}

func (mdb *MarketDataDB) Close() error {
	return mdb.db.Close()
}

func (mdb *MarketDataDB) initSchema() error {
	schema := `
	-- Track all market data sessions/subscriptions
	CREATE TABLE IF NOT EXISTS sessions (
		session_id TEXT PRIMARY KEY,
		symbol TEXT NOT NULL,
		request_type TEXT NOT NULL, -- 'snapshot' or 'subscribe'  
		data_types TEXT NOT NULL,   -- 'trades', 'order_book', 'ohlcv'
		depth INTEGER,              -- NULL for trades/ohlcv, number for order book
		md_req_id TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		is_active BOOLEAN DEFAULT 1
	);

	-- All trade data (snapshots + streaming)
	CREATE TABLE IF NOT EXISTS trades (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		price REAL NOT NULL,
		size REAL NOT NULL,
		aggressor_side TEXT,        -- 'Buy', 'Sell'
		trade_time TEXT,           -- Exchange timestamp
		seq_num INTEGER,           -- FIX sequence number
		md_req_id TEXT,
		is_snapshot BOOLEAN,
		received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- All order book data (bids/offers, snapshots + streaming)  
	CREATE TABLE IF NOT EXISTS order_book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		side TEXT NOT NULL,        -- 'bid' or 'offer'
		price REAL NOT NULL,
		size REAL NOT NULL,
		position INTEGER,          -- Book level (1=best, 2=second, etc.)
		seq_num INTEGER,
		md_req_id TEXT,
		is_snapshot BOOLEAN,
		received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- OHLCV data (snapshots only)
	CREATE TABLE IF NOT EXISTS ohlcv (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		data_type TEXT NOT NULL,   -- 'open', 'high', 'low', 'close', 'volume'
		value REAL NOT NULL,
		entry_time TEXT,           -- Exchange timestamp  
		seq_num INTEGER,
		md_req_id TEXT,
		received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_trades_symbol_time ON trades(symbol, received_at);
	CREATE INDEX IF NOT EXISTS idx_orderbook_symbol_time ON order_book(symbol, received_at);
	CREATE INDEX IF NOT EXISTS idx_ohlcv_symbol_time ON ohlcv(symbol, received_at);
	CREATE INDEX IF NOT EXISTS idx_orderbook_symbol_side_pos ON order_book(symbol, side, position, received_at);
	`

	_, err := mdb.db.Exec(schema)
	return err
}

// Session management
func (mdb *MarketDataDB) CreateSession(sessionID, symbol, requestType, dataTypes, mdReqID string, depth *int) error {
	query := `INSERT INTO sessions (session_id, symbol, request_type, data_types, depth, md_req_id) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := mdb.db.Exec(query, sessionID, symbol, requestType, dataTypes, depth, mdReqID)
	return err
}

// Trade data storage
func (mdb *MarketDataDB) StoreTrade(symbol, price, size, aggressorSide, tradeTime string, seqNum int, mdReqID string, isSnapshot bool) error {
	query := `INSERT INTO trades (symbol, price, size, aggressor_side, trade_time, seq_num, md_req_id, is_snapshot) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := mdb.db.Exec(query, symbol, price, size, aggressorSide, tradeTime, seqNum, mdReqID, isSnapshot)
	return err
}

// Order book data storage
func (mdb *MarketDataDB) StoreOrderBookEntry(symbol, side, price, size string, position, seqNum int, mdReqID string, isSnapshot bool) error {
	query := `INSERT INTO order_book (symbol, side, price, size, position, seq_num, md_req_id, is_snapshot) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := mdb.db.Exec(query, symbol, side, price, size, position, seqNum, mdReqID, isSnapshot)
	return err
}

// OHLCV data storage
func (mdb *MarketDataDB) StoreOHLCV(symbol, dataType, value, entryTime string, seqNum int, mdReqID string) error {
	query := `INSERT INTO ohlcv (symbol, data_type, value, entry_time, seq_num, md_req_id) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := mdb.db.Exec(query, symbol, dataType, value, entryTime, seqNum, mdReqID)
	return err
}

// Batch operations for better performance
func (mdb *MarketDataDB) BeginTransaction() (*sql.Tx, error) {
	return mdb.db.Begin()
}

func (mdb *MarketDataDB) StoreTradeBatch(tx *sql.Tx, symbol, price, size, aggressorSide, tradeTime string, seqNum int, mdReqID string, isSnapshot bool) error {
	query := `INSERT INTO trades (symbol, price, size, aggressor_side, trade_time, seq_num, md_req_id, is_snapshot) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := tx.Exec(query, symbol, price, size, aggressorSide, tradeTime, seqNum, mdReqID, isSnapshot)
	return err
}

func (mdb *MarketDataDB) StoreOrderBookBatch(tx *sql.Tx, symbol, side, price, size string, position, seqNum int, mdReqID string, isSnapshot bool) error {
	query := `INSERT INTO order_book (symbol, side, price, size, position, seq_num, md_req_id, is_snapshot) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := tx.Exec(query, symbol, side, price, size, position, seqNum, mdReqID, isSnapshot)
	return err
}

func (mdb *MarketDataDB) StoreOHLCVBatch(tx *sql.Tx, symbol, dataType, value, entryTime string, seqNum int, mdReqID string) error {
	query := `INSERT INTO ohlcv (symbol, data_type, value, entry_time, seq_num, md_req_id) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := tx.Exec(query, symbol, dataType, value, entryTime, seqNum, mdReqID)
	return err
}
