// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type pgxLog struct {
	pool *Conn
}

func (l *pgxLog) Log(ctx context.Context, ll pgx.LogLevel, msg string, data map[string]any) {
	sql, ok := data["sql"].(string)
	err, isPgErr := data["err"].(*pgconn.PgError)
	if !ok && isPgErr {
		sql = err.Where
	}
	if isPgErr {
		sql += err.Hint
	}
	switch ll {
	case pgx.LogLevelTrace, pgx.LogLevelDebug:
		logs.DebugLog("[[PGX]] %s %+v", msg, data)
	case pgx.LogLevelInfo:
		logs.StatusLog("[[PGX]] %s %+v", msg, data)
	case pgx.LogLevelWarn:
		if isPgErr {
			logs.ErrorLog(err, msg, sql, ok, data["args"])
		} else {
			logs.DebugLog(msg, data)
		}
	case pgx.LogLevelError:
		if isPgErr {
			if !dbEngine.IsErrorAlreadyExists(err) {
				logs.ErrorLog(err, "%s, '%s', args: %+v", msg, sql, ok, data["args"])
			}

			l.pool.addNotice(data["pid"].(uint32), (*pgconn.Notice)(err))
		} else if isPgErr {
			logs.ErrorLog(err, msg, data)
		} else {
			logs.DebugLog(msg, data)
		}
	case pgx.LogLevelNone:
		ch, ok := ctx.Value("debugChan").(chan any)
		if ok {
			ch <- data
		}
	default:
		logs.ErrorLog(errors.New("invalid level "), ll.String())
	}
}

// SetLogLevel set logs level DB operations
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
