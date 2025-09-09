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

package main

import (
	"fmt"
	"log"
	"os"

	"prime-fix-md-go/database"
	"prime-fix-md-go/fixclient"
	"prime-fix-md-go/formatter"
	"prime-fix-md-go/utils"

	"github.com/quickfixgo/quickfix"
)

func main() {
	fmt.Printf("%s\n\n", utils.FullVersion())

	settings, err := utils.LoadSettings("fix.cfg")
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewMarketDataDb("marketdata.db")
	if err != nil {
		log.Fatal("Database initialization failed:", err)
	}

	defer func(db *database.MarketDataDb) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	config := fixclient.NewConfig(
		os.Getenv("PRIME_ACCESS_KEY"),
		os.Getenv("PRIME_SIGNING_KEY"),
		os.Getenv("PRIME_PASSPHRASE"),
		os.Getenv("PRIME_SVC_ACCOUNT_ID"),
		os.Getenv("PRIME_TARGET_COMP_ID"),
		os.Getenv("PRIME_PORTFOLIO_ID"),
	)

	app := fixclient.NewFixApp(config, db)

	initiator, err := quickfix.NewInitiator(app,
		quickfix.NewMemoryStoreFactory(),
		settings,
		formatter.NewTableLogFactory(),
	)
	if err != nil {
		log.Fatal("initiator error:", err)
	}

	if err := initiator.Start(); err != nil {
		log.Fatal("start error:", err)
	}
	defer initiator.Stop()

	fixclient.Repl(app)
}
