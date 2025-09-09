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
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNewTradeStore(t *testing.T) {
	store := NewTradeStore(1000, "")

	if store == nil {
		t.Fatal("TradeStore should not be nil")
	}
}

func TestAddTrades(t *testing.T) {
	store := NewTradeStore(1000, "")

	trades := []Trade{
		{
			Symbol:    "BTC-USD",
			Price:     "50000.00",
			Size:      "1.5",
			Aggressor: "Buy",
			Time:      time.Now().Format(time.RFC3339),
			SeqNum:    "1",
		},
	}

	store.AddTrades("BTC-USD", trades, false, "req-123")

	recentTrades := store.GetRecentTrades("BTC-USD", 10)
	if len(recentTrades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(recentTrades))
	}

	if recentTrades[0].Symbol != "BTC-USD" {
		t.Fatalf("Expected symbol BTC-USD, got %s", recentTrades[0].Symbol)
	}
}

func TestGetRecentTrades(t *testing.T) {
	store := NewTradeStore(1000, "")

	// Add multiple trades
	trades := []Trade{}
	for i := 0; i < 5; i++ {
		trade := Trade{
			Symbol:    "BTC-USD",
			Price:     strconv.Itoa(50000 + i),
			Size:      "1.0",
			Aggressor: "Buy",
			Time:      time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			SeqNum:    strconv.Itoa(i + 1),
		}
		trades = append(trades, trade)
	}

	store.AddTrades("BTC-USD", trades, false, "req-123")

	recentTrades := store.GetRecentTrades("BTC-USD", 3)
	if len(recentTrades) > 3 {
		t.Fatalf("Expected at most 3 trades, got %d", len(recentTrades))
	}
}

func TestMaxSizeLimit(t *testing.T) {
	maxSize := 10
	store := NewTradeStore(maxSize, "")

	// Add more trades than the max size
	tradesCount := 15
	allTrades := []Trade{}

	for i := 0; i < tradesCount; i++ {
		trade := Trade{
			Symbol:    "BTC-USD",
			Price:     strconv.Itoa(50000 + i),
			Size:      "1.0",
			Aggressor: "Buy",
			Time:      time.Now().Add(time.Duration(i) * time.Millisecond).Format(time.RFC3339),
			SeqNum:    strconv.Itoa(i + 1),
		}
		allTrades = append(allTrades, trade)
	}

	store.AddTrades("BTC-USD", allTrades, false, "req-123")

	recentTrades := store.GetRecentTrades("BTC-USD", 100) // Ask for more than we have
	if len(recentTrades) > maxSize {
		t.Fatalf("Expected trade store to be limited to %d trades, got %d", maxSize, len(recentTrades))
	}
}

func TestSubscriptionManagement(t *testing.T) {
	store := NewTradeStore(1000, "")

	// Add subscription
	store.AddSubscription("BTC-USD", "1", "req-123")

	subs := store.GetSubscriptionStatus()
	if len(subs) != 1 {
		t.Fatalf("Expected 1 subscription, got %d", len(subs))
	}

	// Remove subscription by ReqId
	store.RemoveSubscriptionByReqId("req-123")

	subs = store.GetSubscriptionStatus()
	if len(subs) != 0 {
		t.Fatalf("Expected 0 subscriptions after removal, got %d", len(subs))
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := NewTradeStore(1000, "")

	var wg sync.WaitGroup
	numGoroutines := 10
	tradesPerGoroutine := 10

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineId int) {
			defer wg.Done()
			trades := []Trade{}
			for j := 0; j < tradesPerGoroutine; j++ {
				trade := Trade{
					Symbol:    "BTC-USD",
					Price:     strconv.Itoa(50000 + goroutineId*1000 + j),
					Size:      "1.0",
					Aggressor: "Buy",
					Time:      time.Now().Format(time.RFC3339),
					SeqNum:    strconv.Itoa(goroutineId*1000 + j),
				}
				trades = append(trades, trade)
			}
			store.AddTrades("BTC-USD", trades, false, "req-"+strconv.Itoa(goroutineId))
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				trades := store.GetRecentTrades("BTC-USD", 10)
				_ = trades // Use the result to avoid compiler optimization
			}
		}()
	}

	wg.Wait()

	// Verify we have some trades (exact count depends on timing and max size)
	trades := store.GetRecentTrades("BTC-USD", 1000)
	if len(trades) == 0 {
		t.Fatal("Expected some trades after concurrent access")
	}
}

func TestSnapshotHandling(t *testing.T) {
	store := NewTradeStore(1000, "")

	// Add subscription
	store.AddSubscription("BTC-USD", "1", "req-123")

	// Add snapshot trades
	snapshotTrades := []Trade{
		{
			Symbol:    "BTC-USD",
			Price:     "50000.00",
			Size:      "1.0",
			Aggressor: "Buy",
			Time:      time.Now().Format(time.RFC3339),
			SeqNum:    "1",
		},
	}

	store.AddTrades("BTC-USD", snapshotTrades, true, "req-123")

	subs := store.GetSubscriptionStatus()
	if len(subs) != 1 {
		t.Fatalf("Expected 1 subscription, got %d", len(subs))
	}

	// Verify snapshot was recorded in subscription
	for _, sub := range subs {
		if !sub.SnapshotReceived {
			t.Fatal("Expected snapshot to be recorded")
		}
		break
	}
}

func TestMultipleSymbols(t *testing.T) {
	store := NewTradeStore(1000, "")

	// Add trades for different symbols
	btcTrades := []Trade{
		{
			Symbol:    "BTC-USD",
			Price:     "50000.00",
			Size:      "1.0",
			Aggressor: "Buy",
			Time:      time.Now().Format(time.RFC3339),
			SeqNum:    "1",
		},
	}

	ethTrades := []Trade{
		{
			Symbol:    "ETH-USD",
			Price:     "3000.00",
			Size:      "5.0",
			Aggressor: "Sell",
			Time:      time.Now().Format(time.RFC3339),
			SeqNum:    "1",
		},
	}

	store.AddTrades("BTC-USD", btcTrades, false, "req-btc")
	store.AddTrades("ETH-USD", ethTrades, false, "req-eth")

	// Verify we can get trades for each symbol separately
	btcRecent := store.GetRecentTrades("BTC-USD", 10)
	ethRecent := store.GetRecentTrades("ETH-USD", 10)

	if len(btcRecent) != 1 {
		t.Fatalf("Expected 1 BTC trade, got %d", len(btcRecent))
	}

	if len(ethRecent) != 1 {
		t.Fatalf("Expected 1 ETH trade, got %d", len(ethRecent))
	}

	if btcRecent[0].Symbol != "BTC-USD" {
		t.Fatalf("Expected BTC-USD, got %s", btcRecent[0].Symbol)
	}

	if ethRecent[0].Symbol != "ETH-USD" {
		t.Fatalf("Expected ETH-USD, got %s", ethRecent[0].Symbol)
	}
}
