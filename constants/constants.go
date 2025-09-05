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

package constants

import "github.com/quickfixgo/quickfix"

const (
	MsgTypeLogon                 = "A" // Logon
	MsgTypeMarketDataRequest     = "V" // Market Data Request
	MsgTypeMarketDataSnapshot    = "W" // Market Data Snapshot/Full Refresh
	MsgTypeMarketDataIncremental = "X" // Market Data Incremental Refresh

	FixTimeFormat     = "20060102-15:04:05.000"
	FixBeginString    = "FIXT.1.1"
	EncryptMethodNone = "0"
	HeartBtInterval   = "30"
	DropCopyFlagYes   = "Y"
	MsgSeqNumInit     = "1"

	SubscriptionRequestTypeSnapshot    = "0" // Snapshot
	SubscriptionRequestTypeSubscribe   = "1" // Subscribe
	SubscriptionRequestTypeUnsubscribe = "2" // Unsubscribe

	MdEntryTypeBid    = "0" // Bid
	MdEntryTypeOffer  = "1" // Offer/Ask
	MdEntryTypeTrade  = "2" // Trade
	MdEntryTypeOpen   = "4" // Open
	MdEntryTypeClose  = "5" // Close
	MdEntryTypeHigh   = "7" // High
	MdEntryTypeLow    = "8" // Low
	MdEntryTypeVolume = "B" // Volume

	MdUpdateTypeFullRefresh = "0" // Full refresh
	MdUpdateTypeIncremental = "1" // Incremental refresh

	TagAccount          = quickfix.Tag(1)
	TagBeginString      = quickfix.Tag(8)
	TagSymbol           = quickfix.Tag(55)
	TagText             = quickfix.Tag(58)
	TagSenderCompId     = quickfix.Tag(49)
	TagSendingTime      = quickfix.Tag(52)
	TagTargetCompId     = quickfix.Tag(56)
	TagHmac             = quickfix.Tag(96)
	TagMsgType          = quickfix.Tag(35)
	TagUsername         = quickfix.Tag(553)
	TagPassword         = quickfix.Tag(554)
	TagDropCopyFlag     = quickfix.Tag(9406)
	TagAccessKey        = quickfix.Tag(9407)
	TagEncryptMethod    = quickfix.Tag(98)
	TagHeartBtInt       = quickfix.Tag(108)
	TagDefaultApplVerId = quickfix.Tag(1137)
	TagMsgSeqNum        = quickfix.Tag(34)

	// Market Data Request Tags
	TagNoRelatedSym            = quickfix.Tag(146)
	TagMdReqId                 = quickfix.Tag(262)
	TagSubscriptionRequestType = quickfix.Tag(263)
	TagMarketDepth             = quickfix.Tag(264)
	TagMdUpdateType            = quickfix.Tag(265)
	TagNoMdEntryTypes          = quickfix.Tag(267)
	TagMdEntryType             = quickfix.Tag(269)

	// Market Data Response Tags
	TagMdEntryPx         = quickfix.Tag(270)
	TagMdEntrySize       = quickfix.Tag(271)
	TagMdEntryTime       = quickfix.Tag(273)
	TagMdReqRejReason    = quickfix.Tag(281)
	TagNoMdEntries       = quickfix.Tag(268)
	TagMdEntryPositionNo = quickfix.Tag(290)
	TagAggressorSide     = quickfix.Tag(2446)
)
