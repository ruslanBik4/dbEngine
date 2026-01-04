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
	"golang.org/x/xerrors"

	"github.com/ruslanBik4/gotools"
	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type pgxLog struct {
	pool *Conn
}

func (l *pgxLog) Log(ctx context.Context, ll pgx.LogLevel, msg string, data map[string]any) {

	switch ll {
	case pgx.LogLevelTrace, pgx.LogLevelDebug:
		logs.DebugLog("[[PGX]] %s %+v", msg, data)

	case pgx.LogLevelInfo:
		logs.StatusLog("[[PGX]] %s %+v", msg, data)

	case pgx.LogLevelWarn, pgx.LogLevelError:
		l.chkError(msg, data)

	case pgx.LogLevelNone:
		if ch, ok := ctx.Value("debugChan").(chan any); ok {
			ch <- data
		}

	default:
		logs.ErrorLog(errors.New("invalid level "), ll.String())
	}
}

func (l *pgxLog) chkError(msg string, data map[string]any) {
	sql, hasSQL := data["sql"].(string)
	if !hasSQL && len(sql) > 255 {
		sql = gotools.StartEndString(sql, 200)
	}

	switch err := data["err"].(type) {
	case nil:
		logs.DebugLog(msg, data)
	case xerrors.Wrapper:
		logs.ErrorLog(err.Unwrap(), msg, data)
	case *pgconn.PgError:
		sql = l.printPgError(msg, data, sql, err)
		return
	case error:
		logs.ErrorLog(err, msg, data)

	default:
		logs.DebugLog("%v, %s, %v, %[1]T", err, msg, data)
	}

	if hasSQL {
		logs.StatusLog("[PGX]", sql)
	}
}

func (l *pgxLog) printPgError(msg string, data map[string]any, sql string, err *pgconn.PgError) string {
	if dbEngine.IsErrorAlreadyExists(err) {
		submatch := dbEngine.RegAlreadyExists.FindStringSubmatch(err.Error())
		fileName := submatch[2] + ".ddl"
		logs.CustomLog(logs.WARNING, "ALREADY_EXISTS", fileName, int(err.Line), err.Message, logs.FgInfo)
	} else {
		logs.CustomLog(
			logs.ERROR,
			"PGX_ERROR",
			err.File, int(err.Line),
			fmt.Sprintf("%s: %s, '%s %s(%s)', args: %+v, %v",
				msg,
				err.Detail,
				sql,
				gotools.StartEndString(err.Where, 100),
				err.Hint,
				data["args"], err),
			logs.FgErr,
		)
	}

	l.pool.addNotice(data["pid"].(uint32), (*pgconn.Notice)(err))
	return sql
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
