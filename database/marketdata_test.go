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
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*MarketDataDb, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewMarketDataDb(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func TestNewMarketDataDb(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewMarketDataDb(dbPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer db.Close()

	if db.db == nil {
		t.Fatal("Database connection should not be nil")
	}
}

func TestCreateSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sessionId := "test-session-1"
	symbol := "BTC-USD"
	requestType := "snapshot"
	dataTypes := "trades,order_book"
	mdReqId := "req-123"
	depth := 10

	err := db.CreateSession(sessionId, symbol, requestType, dataTypes, mdReqId, &depth)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session was created
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE session_id = ?", sessionId).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query session: %v", err)
	}

	if count != 1 {
		t.Fatalf("Expected 1 session, found %d", count)
	}
}

func TestStoreTrade(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	symbol := "BTC-USD"
	price := "50000.00"
	size := "1.5"
	aggressorSide := "Buy"
	tradeTime := time.Now().Format(time.RFC3339)
	seqNum := 123
	mdReqId := "req-123"
	isSnapshot := false

	err := db.StoreTrade(symbol, price, size, aggressorSide, tradeTime, seqNum, mdReqId, isSnapshot)
	if err != nil {
		t.Fatalf("Failed to store trade: %v", err)
	}

	// Verify trade was stored
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM trades WHERE symbol = ? AND price = ?", symbol, price).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query trade: %v", err)
	}

	if count != 1 {
		t.Fatalf("Expected 1 trade, found %d", count)
	}
}

func TestStoreOrderBookEntry(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	symbol := "BTC-USD"
	side := "bid"
	price := "49999.99"
	size := "2.0"
	position := 1
	seqNum := 124
	mdReqId := "req-124"
	isSnapshot := true

	err := db.StoreOrderBookEntry(symbol, side, price, size, position, seqNum, mdReqId, isSnapshot)
	if err != nil {
		t.Fatalf("Failed to store order book entry: %v", err)
	}

	// Verify order book entry was stored
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM order_book WHERE symbol = ? AND side = ? AND position = ?", symbol, side, position).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query order book: %v", err)
	}

	if count != 1 {
		t.Fatalf("Expected 1 order book entry, found %d", count)
	}
}

func TestStoreOHLCV(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	symbol := "BTC-USD"
	dataType := "open"
	value := "50000.00"
	entryTime := time.Now().Format(time.RFC3339)
	seqNum := 125
	mdReqId := "req-125"

	err := db.StoreOHLCV(symbol, dataType, value, entryTime, seqNum, mdReqId)
	if err != nil {
		t.Fatalf("Failed to store OHLCV: %v", err)
	}

	// Verify OHLCV was stored
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM ohlcv WHERE symbol = ? AND data_type = ?", symbol, dataType).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query OHLCV: %v", err)
	}

	if count != 1 {
		t.Fatalf("Expected 1 OHLCV entry, found %d", count)
	}
}

func TestBatchOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tx, err := db.BeginTransaction()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	symbol := "ETH-USD"
	price := "3000.00"
	size := "5.0"
	aggressorSide := "Sell"
	tradeTime := time.Now().Format(time.RFC3339)
	seqNum := 200
	mdReqId := "batch-req-1"
	isSnapshot := false

	// Test batch trade storage
	err = db.StoreTradeBatch(tx, symbol, price, size, aggressorSide, tradeTime, seqNum, mdReqId, isSnapshot)
	if err != nil {
		t.Fatalf("Failed to store trade in batch: %v", err)
	}

	// Test batch order book storage
	err = db.StoreOrderBookBatch(tx, symbol, "offer", "3001.00", "3.0", 1, seqNum+1, mdReqId, isSnapshot)
	if err != nil {
		t.Fatalf("Failed to store order book in batch: %v", err)
	}

	// Test batch OHLCV storage
	err = db.StoreOhlcvBatch(tx, symbol, "close", "3000.50", tradeTime, seqNum+2, mdReqId)
	if err != nil {
		t.Fatalf("Failed to store OHLCV in batch: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify all entries were committed
	var tradeCount, orderBookCount, ohlcvCount int

	err = db.db.QueryRow("SELECT COUNT(*) FROM trades WHERE symbol = ?", symbol).Scan(&tradeCount)
	if err != nil {
		t.Fatalf("Failed to query trades: %v", err)
	}

	err = db.db.QueryRow("SELECT COUNT(*) FROM order_book WHERE symbol = ?", symbol).Scan(&orderBookCount)
	if err != nil {
		t.Fatalf("Failed to query order book: %v", err)
	}

	err = db.db.QueryRow("SELECT COUNT(*) FROM ohlcv WHERE symbol = ?", symbol).Scan(&ohlcvCount)
	if err != nil {
		t.Fatalf("Failed to query OHLCV: %v", err)
	}

	if tradeCount != 1 || orderBookCount != 1 || ohlcvCount != 1 {
		t.Fatalf("Expected 1 entry each, got trades: %d, order_book: %d, ohlcv: %d", tradeCount, orderBookCount, ohlcvCount)
	}
}

func TestDatabaseConnectionFailure(t *testing.T) {
	// Test with invalid path
	_, err := NewMarketDataDb("/invalid/path/that/does/not/exist/test.db")
	if err == nil {
		t.Fatal("Expected error for invalid database path")
	}
}

func TestTransactionRollback(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tx, err := db.BeginTransaction()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	symbol := "ROLLBACK-TEST"
	price := "1000.00"
	size := "1.0"
	aggressorSide := "Buy"
	tradeTime := time.Now().Format(time.RFC3339)
	seqNum := 999
	mdReqId := "rollback-test"
	isSnapshot := false

	err = db.StoreTradeBatch(tx, symbol, price, size, aggressorSide, tradeTime, seqNum, mdReqId, isSnapshot)
	if err != nil {
		t.Fatalf("Failed to store trade in transaction: %v", err)
	}

	// Rollback instead of commit
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify trade was not persisted
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM trades WHERE symbol = ?", symbol).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query trades after rollback: %v", err)
	}

	if count != 0 {
		t.Fatalf("Expected 0 trades after rollback, found %d", count)
	}
}
