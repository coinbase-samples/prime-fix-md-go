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

package database

import (
	_ "embed"
)

//go:embed schema.sql
var schemaSQL string

const (
	insertSessionQuery = `INSERT INTO sessions (session_id, symbol, request_type, data_types, depth, md_req_id) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	insertTradeQuery = `INSERT INTO trades (symbol, price, size, aggressor_side, trade_time, seq_num, md_req_id, is_snapshot) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	insertOrderBookQuery = `INSERT INTO order_book (symbol, side, price, size, position, seq_num, md_req_id, is_snapshot) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	insertOHLCVQuery = `INSERT INTO ohlcv (symbol, data_type, value, entry_time, seq_num, md_req_id) 
			  VALUES (?, ?, ?, ?, ?, ?)`
)

func (mdb *MarketDataDb) initSchema() error {
	_, err := mdb.db.Exec(schemaSQL)
	return err
}
