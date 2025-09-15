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

package formatter

import (
	"testing"

	"github.com/quickfixgo/quickfix"
)

func TestNewTableLogFactory(t *testing.T) {
	factory := NewTableLogFactory()

	if factory == nil {
		t.Fatal("TableLogFactory should not be nil")
	}
}

func TestTableLogFactoryCreate(t *testing.T) {
	factory := NewTableLogFactory()

	log, err := factory.Create()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if log == nil {
		t.Fatal("Log should not be nil")
	}

	// Verify it's the correct type
	_, ok := log.(*TableLog)
	if !ok {
		t.Fatal("Expected TableLog type")
	}
}

func TestTableLogFactoryCreateSessionLog(t *testing.T) {
	factory := NewTableLogFactory()

	sessionId := quickfix.SessionID{
		BeginString:  "FIX.4.4",
		TargetCompID: "TARGET",
		SenderCompID: "SENDER",
	}

	log, err := factory.CreateSessionLog(sessionId)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if log == nil {
		t.Fatal("Log should not be nil")
	}

	// Verify it's the correct type
	tableLog, ok := log.(*TableLog)
	if !ok {
		t.Fatal("Expected TableLog type")
	}

	// Verify session ID is set correctly
	if tableLog.SessionId != sessionId {
		t.Fatalf("Expected session ID %v, got %v", sessionId, tableLog.SessionId)
	}
}

func TestTableLogOnIncoming(t *testing.T) {
	log := &TableLog{}

	testMessage := "8=FIX.4.4|9=142|35=W|49=SENDER|56=TARGET|34=2|52=20250101-12:00:00|10=123|"

	// This should not panic or return error
	log.OnIncoming([]byte(testMessage))

	// Test with empty message
	log.OnIncoming([]byte(""))

	// Test with nil
	log.OnIncoming(nil)
}

func TestTableLogOnOutgoing(t *testing.T) {
	log := &TableLog{}

	testMessage := "8=FIX.4.4|9=142|35=V|49=SENDER|56=TARGET|34=1|52=20250101-12:00:00|10=123|"

	// This should not panic or return error
	log.OnOutgoing([]byte(testMessage))

	// Test with empty message
	log.OnOutgoing([]byte(""))

	// Test with nil
	log.OnOutgoing(nil)
}

func TestTableLogOnEvent(t *testing.T) {
	log := &TableLog{}

	testEvent := "Session created"

	// This should not panic or return error
	log.OnEvent(testEvent)

	// Test with empty string
	log.OnEvent("")
}

func TestTableLogOnEventWithSessionId(t *testing.T) {
	sessionId := quickfix.SessionID{
		BeginString:  "FIX.4.4",
		TargetCompID: "TARGET",
		SenderCompID: "SENDER",
	}

	log := &TableLog{SessionId: sessionId}

	testEvent := "Session logged in"

	// This should not panic or return error
	log.OnEvent(testEvent)
}

func TestMessageFormatting(t *testing.T) {
	log := &TableLog{}

	// Test the message formatting behavior
	testMessage := "8=FIX.4.4|9=50|35=W|49=SENDER|56=TARGET|10=123|"

	// These methods should handle the input gracefully
	log.OnIncoming([]byte(testMessage))
	log.OnOutgoing([]byte(testMessage))

	// Test with malformed message
	malformedMessage := "not-a-fix-message"
	log.OnIncoming([]byte(malformedMessage))
	log.OnOutgoing([]byte(malformedMessage))
}

func TestConcurrentLogOperations(t *testing.T) {
	log := &TableLog{}

	// Test concurrent access to log methods
	done := make(chan bool, 3)

	// Concurrent incoming messages
	go func() {
		for i := 0; i < 100; i++ {
			log.OnIncoming([]byte("8=FIX.4.4|35=W|10=123|"))
		}
		done <- true
	}()

	// Concurrent outgoing messages
	go func() {
		for i := 0; i < 100; i++ {
			log.OnOutgoing([]byte("8=FIX.4.4|35=V|10=123|"))
		}
		done <- true
	}()

	// Concurrent events
	go func() {
		for i := 0; i < 100; i++ {
			log.OnEvent("Test event")
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestLogFactoryMultipleInstances(t *testing.T) {
	factory := NewTableLogFactory()

	// Create multiple logs
	log1, err1 := factory.Create()
	log2, err2 := factory.Create()

	if err1 != nil || err2 != nil {
		t.Fatalf("Expected no errors, got %v, %v", err1, err2)
	}

	// Verify they are separate instances
	if log1 == log2 {
		t.Fatal("Expected separate log instances")
	}

	// Create multiple session logs
	sessionId1 := quickfix.SessionID{BeginString: "FIX.4.4", TargetCompID: "TARGET1", SenderCompID: "SENDER"}
	sessionId2 := quickfix.SessionID{BeginString: "FIX.4.4", TargetCompID: "TARGET2", SenderCompID: "SENDER"}

	sessionLog1, err1 := factory.CreateSessionLog(sessionId1)
	sessionLog2, err2 := factory.CreateSessionLog(sessionId2)

	if err1 != nil || err2 != nil {
		t.Fatalf("Expected no errors, got %v, %v", err1, err2)
	}

	// Verify they are separate instances with different session IDs
	if sessionLog1 == sessionLog2 {
		t.Fatal("Expected separate session log instances")
	}

	tableLog1 := sessionLog1.(*TableLog)
	tableLog2 := sessionLog2.(*TableLog)

	if tableLog1.SessionId == tableLog2.SessionId {
		t.Fatal("Expected different session IDs")
	}
}
