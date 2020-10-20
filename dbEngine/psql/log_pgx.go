// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"github.com/ruslanBik4/logs"
	"golang.org/x/net/context"
)

type pgxLog struct {
}

func (l *pgxLog) Log(ctx context.Context, ll pgx.LogLevel, msg string, data map[string]interface{}) {
	switch ll {
	case pgx.LogLevelTrace, pgx.LogLevelDebug:
		logs.DebugLog("[[PGX]] %s %+v", msg, data)
	case pgx.LogLevelInfo:
		logs.StatusLog("[[PGX]] %s %+v", msg, data)
	case pgx.LogLevelWarn, pgx.LogLevelError:
		if err, ok := data["err"].(*pgconn.PgError); ok {
			logs.ErrorLog(err, msg, data["sql"], data["args"])
		}
		logs.DebugLog(data)
	case pgx.LogLevelNone:

	default:
		logs.ErrorLog(errors.New("invalid level "), ll.String())
	}
}

func SetLogLevel(lvl string) pgx.LogLevel {
	logLvl, err := pgx.LogLevelFromString(lvl)
	if err == nil {
		return logLvl
	}

	switch lvl {
	case "WARNING":
		return pgx.LogLevelWarn
	case "INFO":
		return pgx.LogLevelInfo
	case "DEBUG":
		return pgx.LogLevelDebug
	case "TRACE":
		return pgx.LogLevelTrace
	default:
		return pgx.LogLevelError
	}

}
