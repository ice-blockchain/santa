// SPDX-License-Identifier: ice License 1.0

//go:build !test

package seeding

import (
	"fmt"
	"os"
	"strings"
	stdlibtime "time"

	"github.com/ice-blockchain/go-tarantool-client"
	"github.com/ice-blockchain/wintr/log"
)

func StartSeeding() {
	before := stdlibtime.Now()
	db := dbConnector()
	defer func() {
		log.Panic(db.Close()) //nolint:revive // It doesnt really matter.
		log.Info(fmt.Sprintf("seeding finalized in %v", stdlibtime.Since(before).String()))
	}()
	log.Info("TODO: implement seeding")
}

func dbConnector() tarantool.Connector {
	parts := strings.Split(os.Getenv("MASTER_DB_INSTANCE_ADDRESS"), "@")
	userAndPass := strings.Split(parts[0], ":")
	opts := tarantool.Opts{
		Timeout:       20 * stdlibtime.Second, //nolint:gomnd // It doesnt matter here.
		Reconnect:     stdlibtime.Millisecond,
		MaxReconnects: 10, //nolint:gomnd // It doesnt matter here.
		User:          userAndPass[0],
		Pass:          userAndPass[1],
	}
	db, err := tarantool.Connect(parts[1], opts)
	log.Panic(err)

	return db
}
