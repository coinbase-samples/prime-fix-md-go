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
	"time"

	"prime-fix-md-go/builder"
	"prime-fix-md-go/constants"

	"github.com/quickfixgo/quickfix"
)

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
