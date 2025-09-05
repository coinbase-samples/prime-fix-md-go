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

package builder

import (
	"time"

	"prime-fix-md-go/constants"
	"prime-fix-md-go/utils"

	"github.com/quickfixgo/quickfix"
)

func BuildLogon(
	body *quickfix.Body,
	ts, apiKey, apiSecret, passphrase, targetCompId, portfolioId string,
) {
	sig := utils.Sign(ts, constants.MsgTypeLogon, constants.MsgSeqNumInit, apiKey, targetCompId, passphrase, apiSecret)

	body.SetField(constants.TagEncryptMethod, quickfix.FIXString(constants.EncryptMethodNone))
	body.SetField(constants.TagHeartBtInt, quickfix.FIXString(constants.HeartBtInterval))

	body.SetField(constants.TagPassword, quickfix.FIXString(passphrase))
	body.SetField(constants.TagAccount, quickfix.FIXString(portfolioId))
	body.SetField(constants.TagHmac, quickfix.FIXString(sig))
	body.SetField(constants.TagUsername, quickfix.FIXString(apiKey))
	body.SetField(constants.TagDropCopyFlag, quickfix.FIXString(constants.DropCopyFlagYes))
}

func BuildMarketDataRequest(
	mdReqId, symbol, subscriptionRequestType, marketDepth, senderCompId, targetCompId string,
	mdEntryTypes []string,
) *quickfix.Message {
	m := quickfix.NewMessage()
	m.Header.SetField(constants.TagBeginString, quickfix.FIXString(constants.FixBeginString))
	m.Header.SetField(constants.TagMsgType, quickfix.FIXString(constants.MsgTypeMarketDataRequest))
	m.Header.SetField(constants.TagSenderCompId, quickfix.FIXString(senderCompId))
	m.Header.SetField(constants.TagTargetCompId, quickfix.FIXString(targetCompId))
	m.Header.SetField(constants.TagSendingTime, quickfix.FIXString(time.Now().UTC().Format(constants.FixTimeFormat)))

	m.Body.SetField(constants.TagMdReqId, quickfix.FIXString(mdReqId))
	m.Body.SetField(constants.TagSubscriptionRequestType, quickfix.FIXString(subscriptionRequestType))
	m.Body.SetField(constants.TagMarketDepth, quickfix.FIXString(marketDepth))

	if subscriptionRequestType == constants.SubscriptionRequestTypeSubscribe {
		m.Body.SetField(constants.TagMdUpdateType, quickfix.FIXString(constants.MdUpdateTypeIncremental))
	}

	mdEntryGroup := quickfix.NewRepeatingGroup(constants.TagNoMdEntryTypes,
		quickfix.GroupTemplate{quickfix.GroupElement(constants.TagMdEntryType)})

	for _, entryType := range mdEntryTypes {
		group := mdEntryGroup.Add()
		group.SetField(constants.TagMdEntryType, quickfix.FIXString(entryType))
	}
	m.Body.SetGroup(mdEntryGroup)

	relatedSymGroup := quickfix.NewRepeatingGroup(constants.TagNoRelatedSym,
		quickfix.GroupTemplate{quickfix.GroupElement(constants.TagSymbol)})

	symbolGroup := relatedSymGroup.Add()
	symbolGroup.SetField(constants.TagSymbol, quickfix.FIXString(symbol))
	m.Body.SetGroup(relatedSymGroup)
	return m
}
