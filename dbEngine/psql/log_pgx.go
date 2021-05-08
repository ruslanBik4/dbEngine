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

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type pgxLog struct {
	pool *Conn
}

func (l *pgxLog) Log(ctx context.Context, ll pgx.LogLevel, msg string, data map[string]interface{}) {
	switch ll {
	case pgx.LogLevelTrace, pgx.LogLevelDebug:
		logs.DebugLog("[[PGX]] %s %+v", msg, data)
	case pgx.LogLevelInfo:
		logs.StatusLog("[[PGX]] %s %+v", msg, data)
	case pgx.LogLevelWarn:
		if err, ok := data["err"].(*pgconn.PgError); ok {
			logs.ErrorLog(err, msg, data["sql"], data["args"])
		} else {
			logs.DebugLog(msg, data)
		}
	case pgx.LogLevelError:
		if err, ok := data["err"].(*pgconn.PgError); ok {
			if !dbEngine.IsErrorAlreadyExists(err) {
				logs.ErrorLog(err, msg, data["sql"], data["args"])
			}

			l.pool.addNotice(data["pid"].(uint32), (*pgconn.Notice)(err))
		} else if err, ok := data["err"].(error); ok {
			logs.ErrorLog(err, msg, data)
		} else {
			logs.DebugLog(msg, data)
		}
	case pgx.LogLevelNone:
		ch, ok := ctx.Value("debugChan").(chan interface{})
		if ok {
			ch <- data
		}
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
