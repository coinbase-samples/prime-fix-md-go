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

package main

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"prime-fix-md-go/database"
	"prime-fix-md-go/fixclient"
)

func TestDatabaseIntegration(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration_test.db")

	db, err := database.NewMarketDataDb(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test session creation
	sessionId := "integration-session-1"
	symbol := "BTC-USD"
	requestType := "snapshot"
	dataTypes := "trades"
	mdReqId := "integration-req-1"
	depth := 5

	err = db.CreateSession(sessionId, symbol, requestType, dataTypes, mdReqId, &depth)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test trade storage
	err = db.StoreTrade(symbol, "50000.00", "1.5", "Buy",
		time.Now().Format(time.RFC3339), 1, mdReqId, true)
	if err != nil {
		t.Fatalf("Failed to store trade: %v", err)
	}

	// Test order book storage
	err = db.StoreOrderBookEntry(symbol, "bid", "49999.99", "2.0", 1, 2, mdReqId, true)
	if err != nil {
		t.Fatalf("Failed to store order book entry: %v", err)
	}

	// Test OHLCV storage
	err = db.StoreOHLCV(symbol, "open", "50000.00", time.Now().Format(time.RFC3339), 3, mdReqId)
	if err != nil {
		t.Fatalf("Failed to store OHLCV: %v", err)
	}
}

func TestTradeStoreIntegration(t *testing.T) {
	tradeStore := fixclient.NewTradeStore(1000, "")

	// Create test trades
	trades := []fixclient.Trade{
		{
			Symbol:    "BTC-USD",
			Price:     "50000.00",
			Size:      "1.0",
			Aggressor: "Buy",
			Time:      time.Now().Format(time.RFC3339),
			SeqNum:    "1",
		},
		{
			Symbol:    "ETH-USD",
			Price:     "3000.00",
			Size:      "5.0",
			Aggressor: "Sell",
			Time:      time.Now().Add(time.Second).Format(time.RFC3339),
			SeqNum:    "2",
		},
		{
			Symbol:    "BTC-USD",
			Price:     "50001.00",
			Size:      "0.5",
			Aggressor: "Buy",
			Time:      time.Now().Add(2 * time.Second).Format(time.RFC3339),
			SeqNum:    "3",
		},
	}

	// Add all trades for each symbol
	btcTrades := []fixclient.Trade{trades[0], trades[2]}
	ethTrades := []fixclient.Trade{trades[1]}

	tradeStore.AddTrades("BTC-USD", btcTrades, false, "req-btc")
	tradeStore.AddTrades("ETH-USD", ethTrades, false, "req-eth")

	// Verify BTC trades
	btcStored := tradeStore.GetRecentTrades("BTC-USD", 10)
	if len(btcStored) != 2 {
		t.Fatalf("Expected 2 BTC trades, got %d", len(btcStored))
	}

	// Verify ETH trades
	ethStored := tradeStore.GetRecentTrades("ETH-USD", 10)
	if len(ethStored) != 1 {
		t.Fatalf("Expected 1 ETH trade, got %d", len(ethStored))
	}
}

func TestDatabaseTradeStoreIntegration(t *testing.T) {
	// Setup database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration_test.db")

	db, err := database.NewMarketDataDb(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Setup trade store
	tradeStore := fixclient.NewTradeStore(1000, "")

	// Create a session
	sessionId := "integration-session"
	symbol := "BTC-USD"
	mdReqId := "integration-req"

	err = db.CreateSession(sessionId, symbol, "subscribe", "trades", mdReqId, nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Simulate receiving trades and storing them in both systems
	testTrades := []struct {
		price     string
		size      string
		aggressor string
		seqNum    int
	}{
		{"50000.00", "1.0", "Buy", 1},
		{"50001.00", "0.5", "Sell", 2},
		{"49999.00", "2.0", "Buy", 3},
	}

	fixTrades := []fixclient.Trade{}
	for i, trade := range testTrades {
		tradeTime := time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339)

		// Store in database
		err = db.StoreTrade(symbol, trade.price, trade.size, trade.aggressor,
			tradeTime, trade.seqNum, mdReqId, false)
		if err != nil {
			t.Fatalf("Failed to store trade in database: %v", err)
		}

		// Prepare for trade store
		fixTrade := fixclient.Trade{
			Symbol:    symbol,
			Price:     trade.price,
			Size:      trade.size,
			Aggressor: trade.aggressor,
			Time:      tradeTime,
			SeqNum:    strconv.Itoa(trade.seqNum),
		}
		fixTrades = append(fixTrades, fixTrade)
	}

	// Store all trades in trade store
	tradeStore.AddTrades(symbol, fixTrades, false, mdReqId)

	// Verify trade store
	storedTrades := tradeStore.GetRecentTrades(symbol, 10)
	if len(storedTrades) != len(testTrades) {
		t.Fatalf("Expected %d trades in store, got %d", len(testTrades), len(storedTrades))
	}
}

func TestConcurrentDatabaseOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "concurrent_test.db")

	db, err := database.NewMarketDataDb(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create session
	sessionId := "concurrent-session"
	symbol := "BTC-USD"
	mdReqId := "concurrent-req"

	err = db.CreateSession(sessionId, symbol, "subscribe", "trades", mdReqId, nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test concurrent trade storage
	numGoroutines := 5
	tradesPerGoroutine := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineId int) {
			defer func() { done <- true }()

			for j := 0; j < tradesPerGoroutine; j++ {
				err := db.StoreTrade(symbol, "50000.00", "1.0", "Buy",
					time.Now().Format(time.RFC3339), goroutineId*100+j, mdReqId, false)
				if err != nil {
					t.Errorf("Failed to store trade: %v", err)
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestEndToEndMarketDataFlow(t *testing.T) {
	// This test simulates the complete flow of receiving market data
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e_test.db")

	// Initialize components
	db, err := database.NewMarketDataDb(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	tradeStore := fixclient.NewTradeStore(1000, "")

	// Simulate market data session
	sessionId := "e2e-session"
	symbol := "BTC-USD"
	mdReqId := "e2e-req"

	// 1. Create session
	err = db.CreateSession(sessionId, symbol, "subscribe", "trades,order_book", mdReqId, nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// 2. Process initial snapshot
	snapshotTrades := []fixclient.Trade{
		{Symbol: symbol, Price: "50000.00", Size: "1.0", Aggressor: "Buy", Time: time.Now().Format(time.RFC3339), SeqNum: "1"},
		{Symbol: symbol, Price: "50001.00", Size: "0.5", Aggressor: "Sell", Time: time.Now().Format(time.RFC3339), SeqNum: "2"},
	}

	for _, trade := range snapshotTrades {
		// Store in database
		seqNum, _ := strconv.Atoi(trade.SeqNum)
		err = db.StoreTrade(trade.Symbol, trade.Price, trade.Size, trade.Aggressor,
			trade.Time, seqNum, mdReqId, true)
		if err != nil {
			t.Fatalf("Failed to store snapshot trade: %v", err)
		}
	}

	// Store in memory
	tradeStore.AddTrades(symbol, snapshotTrades, true, mdReqId)

	// 3. Process streaming updates
	streamingTrades := []fixclient.Trade{
		{Symbol: symbol, Price: "50002.00", Size: "2.0", Aggressor: "Buy", Time: time.Now().Format(time.RFC3339), SeqNum: "3"},
		{Symbol: symbol, Price: "50003.00", Size: "1.5", Aggressor: "Sell", Time: time.Now().Format(time.RFC3339), SeqNum: "4"},
	}

	for _, trade := range streamingTrades {
		// Store in database
		seqNum, _ := strconv.Atoi(trade.SeqNum)
		err = db.StoreTrade(trade.Symbol, trade.Price, trade.Size, trade.Aggressor,
			trade.Time, seqNum, mdReqId, false)
		if err != nil {
			t.Fatalf("Failed to store streaming trade: %v", err)
		}
	}

	// Store in memory
	tradeStore.AddTrades(symbol, streamingTrades, false, mdReqId)

	// 4. Verify final state
	allTrades := tradeStore.GetRecentTrades(symbol, 10)
	expectedTradeCount := len(snapshotTrades) + len(streamingTrades)

	if len(allTrades) != expectedTradeCount {
		t.Fatalf("Expected %d total trades, got %d", expectedTradeCount, len(allTrades))
	}
}
