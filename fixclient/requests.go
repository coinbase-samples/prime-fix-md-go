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
			fmt.Printf("  %d. ReqId: %s, Type: %s, Updates: %d\n",
				i+1, sub.MdReqId, a.getSubscriptionTypeDesc(sub.SubscriptionType), sub.TotalUpdates)
		}
		fmt.Printf("Unsubscribing from all %d subscriptions for %s\n", len(symbolSubs), symbol)
	}

	for _, sub := range symbolSubs {
		msg := builder.BuildMarketDataRequest(
			sub.MdReqId,
			[]string{symbol},
			constants.SubscriptionRequestTypeUnsubscribe,
			"0",
			a.Config.SenderCompId,
			a.Config.TargetCompId,
			[]string{constants.MdEntryTypeTrade},
		)

		if err := quickfix.Send(msg); err != nil {
			log.Printf("Error sending unsubscribe request for reqId %s: %v", sub.MdReqId, err)
		} else {
			fmt.Printf("Unsubscribe request sent for %s (reqId: %s)\n", symbol, sub.MdReqId)
			a.TradeStore.RemoveSubscriptionByReqId(sub.MdReqId)
		}
	}
}

func (a *FixApp) sendUnsubscribeByReqId(reqId string) {
	subscriptions := a.TradeStore.GetSubscriptionStatus()

	sub, exists := subscriptions[reqId]
	if !exists {
		fmt.Printf("No active subscription found with reqId: %s\n", reqId)
		return
	}

	msg := builder.BuildMarketDataRequest(
		reqId,
		[]string{sub.Symbol},
		constants.SubscriptionRequestTypeUnsubscribe,
		"0",
		a.Config.SenderCompId,
		a.Config.TargetCompId,
		[]string{constants.MdEntryTypeTrade},
	)

	if err := quickfix.Send(msg); err != nil {
		log.Printf("Error sending unsubscribe request for reqId %s: %v", reqId, err)
		fmt.Printf("Failed to send unsubscribe request for reqId: %s\n", reqId)
	} else {
		fmt.Printf("Unsubscribe request sent for %s (reqId: %s)\n", sub.Symbol, reqId)
		a.TradeStore.RemoveSubscriptionByReqId(reqId)
	}
}

func (a *FixApp) sendMarketDataRequest(symbols []string, subscriptionType, description string) {
	a.sendMarketDataRequestWithOptions(symbols, subscriptionType, "0", []string{constants.MdEntryTypeTrade}, description)
}

func (a *FixApp) sendMarketDataRequestWithOptions(symbols []string, subscriptionType, marketDepth string, entryTypes []string, description string) {
	reqId := fmt.Sprintf("md_%d", time.Now().UnixNano())

	if subscriptionType == constants.SubscriptionRequestTypeSubscribe {
		for _, symbol := range symbols {
			a.TradeStore.AddSubscription(symbol, subscriptionType, reqId)
		}
	}

	for _, symbol := range symbols {
		a.createDatabaseSession(symbol, subscriptionType, marketDepth, entryTypes, reqId)
	}

	msg := builder.BuildMarketDataRequest(
		reqId,
		symbols,
		subscriptionType,
		marketDepth,
		a.Config.SenderCompId,
		a.Config.TargetCompId,
		entryTypes,
	)

	if err := quickfix.Send(msg); err != nil {
		log.Printf("Error sending market data request: %v", err)
		fmt.Printf("Failed to send %s request for %v\n", description, symbols)
		for _, symbol := range symbols {
			a.TradeStore.RemoveSubscription(symbol)
		}
	} else {
		entryTypesStr := ""
		for i, et := range entryTypes {
			if i > 0 {
				entryTypesStr += ", "
			}
			entryTypesStr += getMdEntryTypeName(et)
		}
		fmt.Printf("%s request sent for %v (depth=%s, types=[%s], reqId=%s)\n",
			description, symbols, marketDepth, entryTypesStr, reqId)
	}
}
