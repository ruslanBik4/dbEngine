// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"fmt"

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
	sql, hasSQL := data["sql"].(string)
	err, isPgErr := data["err"].(*pgconn.PgError)
	if !hasSQL && isPgErr {
		sql = err.Where
	}
	if len(sql) > 255 {
		sql = sql[:255]
	}
	if isPgErr {
		sql += fmt.Sprintf("(%s) %s %s:%d", err.Hint, err.Where, err.File, err.Line)
	}

	switch ll {
	case pgx.LogLevelTrace, pgx.LogLevelDebug:
		logs.DebugLog("[[PGX]] %s %+v", msg, data)
	case pgx.LogLevelInfo:
		logs.StatusLog("[[PGX]] %s %+v", msg, data)
	case pgx.LogLevelWarn:
		if isPgErr {
			logs.ErrorLog(err, msg, sql, data["args"])
		} else {
			logs.DebugLog(msg, data)
		}

	case pgx.LogLevelError:
		if isPgErr {
			if dbEngine.IsErrorAlreadyExists(err) {
				submatch := dbEngine.RegAlreadyExists.FindStringSubmatch(err.Error())
				fileName := submatch[2] + ".ddl"
				//switch submatch[0] {
				//case "role":
				//	fileName = err.
				//}
				logs.CustomLog(logs.WARNING, "ALREADY_EXISTS", fileName, int(err.Line), err.Message, logs.FgInfo)
			} else {
				logs.ErrorLog(err, "%s: %s, '%s', args: %+v", msg, err.Detail, sql, data["args"])
			}

			l.pool.addNotice(data["pid"].(uint32), (*pgconn.Notice)(err))
		} else if err, ok := data["err"].(error); ok {
			logs.ErrorLog(err, msg, data)
		} else {
			logs.DebugLog(msg, data)
		}

	case pgx.LogLevelNone:
		if ch, ok := ctx.Value("debugChan").(chan any); ok {
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
