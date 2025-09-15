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
	"testing"
	"time"
)

func createTestFixApp() *FixApp {
	return &FixApp{
		TradeStore: NewTradeStore(1000, ""),
	}
}

func TestFindEntryBoundaries(t *testing.T) {
	app := createTestFixApp()

	// Test message with multiple MD entries
	rawMsg := "8=FIX.4.4|9=142|35=W|49=SENDER|56=TARGET|34=2|52=20250101-12:00:00|55=BTC-USD|268=2|269=0|270=50000.00|271=1.5|269=1|270=49999.00|271=2.0|10=123|"

	boundaries := app.findEntryBoundaries(rawMsg)

	if len(boundaries) != 2 {
		t.Fatalf("Expected 2 entry boundaries, got %d", len(boundaries))
	}
}

func TestGetEntryEndPos(t *testing.T) {
	app := createTestFixApp()

	entryStarts := []int{10, 50, 90}

	// Test middle entry
	endPos := app.getEntryEndPos(entryStarts, 1, 120)
	if endPos != 90 {
		t.Fatalf("Expected end position 90, got %d", endPos)
	}

	// Test last entry
	endPos = app.getEntryEndPos(entryStarts, 2, 120)
	if endPos != 120 {
		t.Fatalf("Expected end position 120 (message length), got %d", endPos)
	}
}

func TestParseTradeFromSegment(t *testing.T) {
	// Test the parsing helper function directly
	segment := "269=2|270=50000.00|271=1.5|2446=1|273=20250101-12:30:45|"

	price := parseValueFromSegment(segment, "270")
	if price != "50000.00" {
		t.Fatalf("Expected price 50000.00, got %s", price)
	}

	size := parseValueFromSegment(segment, "271")
	if size != "1.5" {
		t.Fatalf("Expected size 1.5, got %s", size)
	}

	entryType := parseValueFromSegment(segment, "269")
	if entryType != "2" {
		t.Fatalf("Expected entry type 2, got %s", entryType)
	}
}

func TestParseSegmentValues(t *testing.T) {
	testCases := []struct {
		name     string
		segment  string
		tag      string
		expected string
	}{
		{
			name:     "Parse price",
			segment:  "269=2|270=50000.00|271=1.5|",
			tag:      "270",
			expected: "50000.00",
		},
		{
			name:     "Parse size",
			segment:  "269=2|270=50000.00|271=1.5|",
			tag:      "271",
			expected: "1.5",
		},
		{
			name:     "Parse entry type",
			segment:  "269=2|270=50000.00|271=1.5|",
			tag:      "269",
			expected: "2",
		},
		{
			name:     "Missing tag",
			segment:  "269=2|270=50000.00|271=1.5|",
			tag:      "999",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseValueFromSegment(tc.segment, tc.tag)
			if result != tc.expected {
				t.Fatalf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestTimeParsingEdgeCases(t *testing.T) {
	testCases := []struct {
		name      string
		timeStr   string
		shouldErr bool
	}{
		{
			name:      "Valid UTC time",
			timeStr:   "20250101-12:30:45",
			shouldErr: false,
		},
		{
			name:      "Invalid format",
			timeStr:   "invalid-time",
			shouldErr: true,
		},
		{
			name:      "Empty string",
			timeStr:   "",
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseTradeTime(tc.timeStr)

			if tc.shouldErr {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error but got: %v", err)
				}
				if result.IsZero() {
					t.Fatal("Expected valid time but got zero time")
				}
			}
		})
	}
}

func TestAggressorSideMapping(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"1", "Buy"},
		{"2", "Sell"},
		{"0", "Unknown"},
		{"", "Unknown"},
		{"invalid", "Unknown"},
	}

	for _, tc := range testCases {
		t.Run("AggressorSide_"+tc.input, func(t *testing.T) {
			result := mapAggressorSide(tc.input)
			if result != tc.expected {
				t.Fatalf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// Helper functions used by tests
func parseValueFromSegment(segment, tag string) string {
	tagPrefix := tag + "="
	startIdx := strings.Index(segment, tagPrefix)
	if startIdx == -1 {
		return ""
	}

	startIdx += len(tagPrefix)
	endIdx := strings.Index(segment[startIdx:], "|")
	if endIdx == -1 {
		return segment[startIdx:]
	}

	return segment[startIdx : startIdx+endIdx]
}

func parseTradeTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	// FIX standard time format: YYYYMMDD-HH:MM:SS
	layout := "20060102-15:04:05"
	return time.Parse(layout, timeStr)
}

func mapAggressorSide(side string) string {
	switch side {
	case "1":
		return "Buy"
	case "2":
		return "Sell"
	default:
		return "Unknown"
	}
}
