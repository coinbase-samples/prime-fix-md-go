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

type FieldSetter interface {
	SetField(tag quickfix.Tag, field quickfix.FieldValueWriter) *quickfix.FieldMap
}

func setString(fs FieldSetter, tag quickfix.Tag, value string) {
	fs.SetField(tag, quickfix.FIXString(value))
}

func BuildLogon(
	body *quickfix.Body,
	ts, apiKey, apiSecret, passphrase, targetCompId, portfolioId string,
) {
	sig := utils.Sign(ts, constants.MsgTypeLogon, constants.MsgSeqNumInit, apiKey, targetCompId, passphrase, apiSecret)

	setString(body, constants.TagEncryptMethod, constants.EncryptMethodNone)
	setString(body, constants.TagHeartBtInt, constants.HeartBtInterval)

	setString(body, constants.TagPassword, passphrase)
	setString(body, constants.TagAccount, portfolioId)
	setString(body, constants.TagHmac, sig)
	setString(body, constants.TagUsername, apiKey)
	setString(body, constants.TagDropCopyFlag, constants.DropCopyFlagYes)
}

func BuildMarketDataRequest(
	mdReqId string,
	symbols []string,
	subscriptionRequestType string,
	marketDepth string,
	senderCompId string,
	targetCompId string,
	mdEntryTypes []string,
) *quickfix.Message {
	m := quickfix.NewMessage()
	setString(&m.Header, constants.TagBeginString, constants.FixBeginString)
	setString(&m.Header, constants.TagMsgType, constants.MsgTypeMarketDataRequest)
	setString(&m.Header, constants.TagSenderCompId, senderCompId)
	setString(&m.Header, constants.TagTargetCompId, targetCompId)
	setString(&m.Header, constants.TagSendingTime, time.Now().UTC().Format(constants.FixTimeFormat))

	setString(&m.Body, constants.TagMdReqId, mdReqId)
	setString(&m.Body, constants.TagSubscriptionRequestType, subscriptionRequestType)
	setString(&m.Body, constants.TagMarketDepth, marketDepth)

	if subscriptionRequestType == constants.SubscriptionRequestTypeSubscribe {
		setString(&m.Body, constants.TagMdUpdateType, constants.MdUpdateTypeIncremental)
	}

	mdEntryGroup := quickfix.NewRepeatingGroup(
		constants.TagNoMdEntryTypes,
		quickfix.GroupTemplate{quickfix.GroupElement(constants.TagMdEntryType)},
	)

	for _, entryType := range mdEntryTypes {
		setString(mdEntryGroup.Add(), constants.TagMdEntryType, entryType)
	}
	m.Body.SetGroup(mdEntryGroup)

	relatedSymGroup := quickfix.NewRepeatingGroup(
		constants.TagNoRelatedSym,
		quickfix.GroupTemplate{quickfix.GroupElement(constants.TagSymbol)},
	)

	for _, symbol := range symbols {
		setString(relatedSymGroup.Add(), constants.TagSymbol, symbol)
	}
	m.Body.SetGroup(relatedSymGroup)
	return m
}
