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
	"strings"
	"time"

	"prime-fix-md-go/constants"
	"prime-fix-md-go/utils"

	"github.com/quickfixgo/quickfix"
)

func (a *FixApp) extractTrades(msg *quickfix.Message, symbol, mdReqId string, isSnapshot bool, seqNum string) []Trade {
	return a.extractTradesImproved(msg, symbol, mdReqId, isSnapshot, seqNum)
}

func (a *FixApp) extractTradesImproved(msg *quickfix.Message, symbol, mdReqId string, isSnapshot bool, seqNum string) []Trade {
	rawMsg := msg.String()

	noMdEntriesStr := utils.GetString(msg, constants.TagNoMdEntries)
	if noMdEntriesStr == "" || noMdEntriesStr == "0" {
		return []Trade{}
	}

	entryStarts := a.findEntryBoundaries(rawMsg)

	var trades []Trade
	for i, startPos := range entryStarts {
		endPos := a.getEntryEndPos(entryStarts, i, len(rawMsg))
		entrySegment := rawMsg[startPos:endPos]

		trade := a.parseTradeFromSegment(entrySegment, symbol, mdReqId, isSnapshot, seqNum, i)
		trades = append(trades, trade)
	}

	return trades
}

func (a *FixApp) findEntryBoundaries(rawMsg string) []int {
	var entryStarts []int
	searchFrom := 0
	for {
		pos := strings.Index(rawMsg[searchFrom:], "269=")
		if pos == -1 {
			break
		}
		entryStarts = append(entryStarts, searchFrom+pos)
		searchFrom += pos + 4
	}
	return entryStarts
}

func (a *FixApp) getEntryEndPos(entryStarts []int, currentIndex, msgLen int) int {
	if currentIndex < len(entryStarts)-1 {
		return entryStarts[currentIndex+1]
	}
	return msgLen
}

func (a *FixApp) parseTradeFromSegment(segment, symbol, mdReqId string, isSnapshot bool, seqNum string, entryIndex int) Trade {
	trade := Trade{
		Timestamp:  time.Now(),
		Symbol:     symbol,
		MdReqId:    mdReqId,
		IsSnapshot: isSnapshot,
		IsUpdate:   !isSnapshot,
		SeqNum:     seqNum,
	}

	if entryType := extractSingleFieldValue(segment, "269="); entryType != "" {
		trade.EntryType = entryType
	}
	if price := extractSingleFieldValue(segment, "270="); price != "" {
		trade.Price = price
	}
	if size := extractSingleFieldValue(segment, "271="); size != "" {
		trade.Size = size
	}
	if timeVal := extractSingleFieldValue(segment, "273="); timeVal != "" {
		trade.Time = timeVal
	}

	if position := extractSingleFieldValue(segment, "290="); position != "" {
		trade.Position = position
	} else {
		if trade.EntryType == "0" || trade.EntryType == "1" { // Bids or Offers
			trade.Position = fmt.Sprintf("%d", entryIndex+1)
		}
	}

	if aggressor := extractSingleFieldValue(segment, "2446="); aggressor != "" {
		trade.Aggressor = getAggressorSideDesc(aggressor)
	}

	return trade
}

func extractSingleFieldValue(fixSegment, tagPrefix string) string {
	start := strings.Index(fixSegment, tagPrefix)
	if start == -1 {
		return ""
	}

	start += len(tagPrefix)
	end := strings.Index(fixSegment[start:], "\x01") // FIX field delimiter
	if end == -1 {
		return fixSegment[start:]
	}

	return fixSegment[start : start+end]
}
