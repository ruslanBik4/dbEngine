// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"fmt"
	"strings"

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
	case *pgconn.PgError:
		sql = l.printPgError(msg, data, sql, err)
		return
	case xerrors.Wrapper:
		logs.ErrorLog(err.Unwrap(), msg, data)
	case error:
		logs.ErrorLog(err, msg, data)

	default:
		logs.DebugLog("%v, %s, %v, %[1]T", err, msg, data)
	}

	if hasSQL {
		logs.CustomLog(logs.DEBUG, "[PGX]", "sgl", 0, sql, logs.FgErr)
	}
}

func (l *pgxLog) printPgError(msg string, data map[string]any, sql string, err *pgconn.PgError) string {
	logPgError(msg, data["args"], sql, err)

	l.pool.addNotice(data["pid"].(uint32), (*pgconn.Notice)(err))
	return sql
}

func logPgError(msg string, args any, sql string, err *pgconn.PgError) {
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
				gotools.StartEndString(sql, 125),
				gotools.StartEndString(err.Where, 100),
				err.Hint,
				args, err),
			logs.FgErr,
		)
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

// PrintNotice logging some psql messages (invoked command 'RAISE ...')
func PrintNotice(c *pgconn.PgConn, n *pgconn.Notice) {
	level := logs.CRITICAL
	fgErr := logs.FgDebug
	msg := n.Message

	switch {
	case n.Code == "42P07" || strings.Contains(n.Message, "skipping"):
		level = logs.NOTICE
		msg = fmt.Sprintf("skip operation: %s", n.Message)

	case n.Severity == "INFO":
		level = logs.INFO
		fgErr = logs.FgInfo

	case n.Code > "00000":
		err := (*pgconn.PgError)(n)
		fgErr = logs.FgErr
		msg = fmt.Sprintf(
			"%v, hint: %s, where: %s, %s %s",
			err,
			n.Hint,
			gotools.StartEndString(n.Where, 100),
			err.SQLState(),
			err.Routine,
		)

	case strings.HasPrefix(n.Message, "[[ERROR]]"):
		level = logs.ERROR
		fgErr = logs.FgErr
		msg = strings.TrimPrefix(n.Message, "[[ERROR]]") + n.Severity

	default: // DEBUG
		level = logs.DEBUG
		msg = fmt.Sprintf("%+v %s (PID:%d)", n.Severity, n.Message, c.PID())
	}

	logs.CustomLog(
		level,
		"DB_EXEC",
		n.File,
		int(n.Line),
		msg,
		fgErr,
	)
}
