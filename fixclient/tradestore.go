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
	"sync"
	"time"
)

type Trade struct {
	Timestamp  time.Time `json:"timestamp"`
	Symbol     string    `json:"symbol"`
	Price      string    `json:"price"`
	Size       string    `json:"size"`
	Time       string    `json:"time"`
	Aggressor  string    `json:"aggressor"`
	MDReqID    string    `json:"mdReqId"`
	IsSnapshot bool      `json:"isSnapshot"`
	IsUpdate   bool      `json:"isUpdate"`
	EntryType  string    `json:"entryType"` // MDEntryType (0=Bid, 1=Offer, 2=Trade, 4=Open, 5=Close, 7=High, 8=Low, B=Volume)
	Position   string    `json:"position"`  // Position in book (for bids/offers)
	SeqNum     string    `json:"seqNum"`    // FIX MsgSeqNum for ordering
}

type TradeStore struct {
	mu            sync.RWMutex
	trades        []Trade
	subscriptions map[string]*Subscription // reqID -> subscription info
	updateCount   int64
	maxSize       int
}

type Subscription struct {
	Symbol           string
	SubscriptionType string // "0"=snapshot, "1"=subscribe, "2"=unsubscribe
	MDReqID          string
	Active           bool
	LastUpdate       time.Time
	TotalUpdates     int64
	SnapshotReceived bool
}

func NewTradeStore(maxSize int, persistenceFile string) *TradeStore {
	return &TradeStore{
		trades:        make([]Trade, 0),
		subscriptions: make(map[string]*Subscription),
		maxSize:       maxSize,
	}
}

func (ts *TradeStore) AddTrades(symbol string, trades []Trade, isSnapshot bool, mdReqID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if sub, exists := ts.subscriptions[mdReqID]; exists {
		sub.LastUpdate = time.Now()
		sub.TotalUpdates += int64(len(trades))
		if isSnapshot {
			sub.SnapshotReceived = true
		}
	}

	for _, trade := range trades {
		trade.Timestamp = time.Now()
		trade.Symbol = symbol
		trade.MDReqID = mdReqID
		trade.IsSnapshot = isSnapshot
		trade.IsUpdate = !isSnapshot

		if len(ts.trades) >= ts.maxSize {
			ts.trades = ts.trades[1:]
		}
		ts.trades = append(ts.trades, trade)
		ts.updateCount++
	}
}

func (ts *TradeStore) GetRecentTrades(symbol string, limit int) []Trade {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var recent []Trade
	count := 0

	// Get trades for symbol in reverse order (newest first)
	for i := len(ts.trades) - 1; i >= 0 && count < limit; i-- {
		if ts.trades[i].Symbol == symbol {
			recent = append([]Trade{ts.trades[i]}, recent...) // Prepend to maintain chronological order
			count++
		}
	}

	return recent
}

func (ts *TradeStore) GetAllTrades() []Trade {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make([]Trade, len(ts.trades))
	copy(result, ts.trades)
	return result
}

func (ts *TradeStore) AddSubscription(symbol, subscriptionType, mdReqID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.subscriptions[mdReqID] = &Subscription{
		Symbol:           symbol,
		SubscriptionType: subscriptionType,
		MDReqID:          mdReqID,
		Active:           true,
		LastUpdate:       time.Now(),
		TotalUpdates:     0,
		SnapshotReceived: false,
	}

	log.Printf("Added subscription: %s (type=%s, reqID=%s)", symbol, getSubscriptionTypeDesc(subscriptionType), mdReqID)
}

func (ts *TradeStore) RemoveSubscription(symbol string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Find all subscriptions for this symbol and remove them
	for reqID, sub := range ts.subscriptions {
		if sub.Symbol == symbol {
			delete(ts.subscriptions, reqID)
			log.Printf("Removed subscription: %s (reqID: %s, total updates: %d)", symbol, reqID, sub.TotalUpdates)
		}
	}
}

func (ts *TradeStore) RemoveSubscriptionByReqID(reqID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if sub, exists := ts.subscriptions[reqID]; exists {
		delete(ts.subscriptions, reqID)
		log.Printf("Removed subscription: %s (ReqID: %s)", sub.Symbol, reqID)
	}
}

func (ts *TradeStore) GetSubscriptionStatus() map[string]*Subscription {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make(map[string]*Subscription)
	for reqID, v := range ts.subscriptions {
		// Create copy to avoid race conditions
		sub := *v
		result[reqID] = &sub
	}
	return result
}

func (ts *TradeStore) GetSubscriptionsBySymbol() map[string][]*Subscription {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make(map[string][]*Subscription)
	for _, sub := range ts.subscriptions {
		// Create copy to avoid race conditions
		subCopy := *sub
		result[sub.Symbol] = append(result[sub.Symbol], &subCopy)
	}
	return result
}

func getSubscriptionTypeDesc(subType string) string {
	switch subType {
	case "0":
		return "Snapshot Only"
	case "1":
		return "Snapshot + Updates"
	case "2":
		return "Unsubscribe"
	default:
		return "Unknown"
	}
}

// DisplayRealtimeUpdate shows a single line update for streaming mode
func (ts *TradeStore) DisplayRealtimeUpdate(trade Trade) {
	entryType := trade.EntryType
	if entryType == "" {
		entryType = "2" // Default to Trade
	}

	switch entryType {
	case "0": // Bid
		log.Printf("%s Bid: %s | Size: %s | Pos: %s",
			trade.Symbol, trade.Price, trade.Size, trade.Position)
	case "1": // Offer
		log.Printf("%s Offer: %s | Size: %s | Pos: %s",
			trade.Symbol, trade.Price, trade.Size, trade.Position)
	case "2": // Trade
		aggressor := trade.Aggressor
		if aggressor == "" {
			aggressor = "-"
		}
		log.Printf("%s Trade: %s | Size: %s | Aggressor: %s",
			trade.Symbol, trade.Price, trade.Size, aggressor)
	case "4": // Open
		log.Printf("%s Open: %s", trade.Symbol, trade.Price)
	case "5": // Close
		log.Printf("%s Close: %s", trade.Symbol, trade.Price)
	case "7": // High
		log.Printf("%s High: %s", trade.Symbol, trade.Price)
	case "8": // Low
		log.Printf("%s Low: %s", trade.Symbol, trade.Price)
	case "B": // Volume
		log.Printf("%s Volume: %s", trade.Symbol, trade.Size)
	default: // Unknown
		log.Printf("%s [%s]: %s | Size: %s",
			trade.Symbol, entryType, trade.Price, trade.Size)
	}
}
