package psql

import (
	"testing"

	"github.com/jackc/pgconn"

	"github.com/ruslanBik4/logs"
)

func TestPrintNotice(t *testing.T) {
	type args struct {
		c pgconn.PgConn
		n pgconn.Notice
	}

	logs.SetDebug(true)
	c := pgconn.PgConn{}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
		{
			name: "TestPrintNotice",
			args: args{
				c,
				(pgconn.Notice)(pgconn.PgError{
					Severity:         "",
					Code:             "",
					Message:          "this is skipping test",
					Detail:           "",
					Hint:             "",
					Position:         0,
					InternalPosition: 0,
					InternalQuery:    "",
					Where:            "",
					SchemaName:       "",
					TableName:        "",
					ColumnName:       "",
					DataTypeName:     "",
					ConstraintName:   "",
					File:             "test.prn",
					Line:             1,
					Routine:          "",
				}),
			},
		},
		{
			name: "TestPrintNotice",
			args: args{
				c,
				(pgconn.Notice)(pgconn.PgError{
					Severity:         "",
					Code:             "42P07",
					Message:          "this is notice test",
					Detail:           "",
					Hint:             "",
					Position:         0,
					InternalPosition: 0,
					InternalQuery:    "",
					Where:            "",
					SchemaName:       "",
					TableName:        "",
					ColumnName:       "",
					DataTypeName:     "",
					ConstraintName:   "",
					File:             "test.prn",
					Line:             2,
					Routine:          "",
				}),
			},
		},
		{
			name: "TestPrintINFO",
			args: args{
				c,
				(pgconn.Notice)(pgconn.PgError{
					Severity:         "",
					Code:             "",
					Message:          "INFO",
					Detail:           "",
					Hint:             "",
					Position:         0,
					InternalPosition: 0,
					InternalQuery:    "",
					Where:            "",
					SchemaName:       "",
					TableName:        "",
					ColumnName:       "",
					DataTypeName:     "",
					ConstraintName:   "",
					File:             "test.prn",
					Line:             3,
					Routine:          "",
				}),
			},
		},
		{
			name: "TestPrintErrorCode",
			args: args{
				c,
				(pgconn.Notice)(pgconn.PgError{
					Severity:         "severity",
					Code:             "00001",
					Message:          "message",
					Detail:           "detail",
					Hint:             "pls read hint anywhere!",
					Position:         0,
					InternalPosition: 0,
					InternalQuery:    "",
					Where:            "test case ErrorCode",
					SchemaName:       "",
					TableName:        "table",
					ColumnName:       "column",
					DataTypeName:     "",
					ConstraintName:   "",
					File:             "test.prn",
					Line:             4,
					Routine:          "test routine",
				}),
			},
		},
		{
			name: "TestPrintCustomError",
			args: args{
				c,
				(pgconn.Notice)(pgconn.PgError{
					Severity:         "test severity",
					Code:             "00000",
					Message:          "[[ERROR]] this is custom test error",
					Detail:           "",
					Hint:             "",
					Position:         0,
					InternalPosition: 0,
					InternalQuery:    "",
					Where:            "",
					SchemaName:       "",
					TableName:        "",
					ColumnName:       "",
					DataTypeName:     "",
					ConstraintName:   "",
					File:             "test.prn",
					Line:             5,
					Routine:          "",
				}),
			},
		},
		{
			name: "TestPrintDebug",
			args: args{
				c,
				(pgconn.Notice)(pgconn.PgError{
					Severity:         "debugSeverity",
					Code:             "00000",
					Message:          "all others messgae print as debug",
					Detail:           "",
					Hint:             "",
					Position:         0,
					InternalPosition: 0,
					InternalQuery:    "",
					Where:            "",
					SchemaName:       "",
					TableName:        "",
					ColumnName:       "",
					DataTypeName:     "",
					ConstraintName:   "",
					File:             "test.prn",
					Line:             8,
					Routine:          "",
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintNotice(&tt.args.c, &tt.args.n)
		})
	}
}
