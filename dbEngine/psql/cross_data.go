// Copyright 2018 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"bytes"
	"database/sql"
	"time"

	"github.com/jackc/pgtype"

	"github.com/ruslanBik4/logs"
)

func TrimQuotes(src []byte) string {

	return string(bytes.Trim(src, `"`))
}

func GetTextDecoder[T pgtype.TextDecoder](ci *pgtype.ConnInfo, src []byte, name string, dto T) T {
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
	}
	return dto
}

// GetInt32FromByte convert data from src into int32
func GetScanner[T sql.Scanner](ci *pgtype.ConnInfo, src []byte, name string, dto T) T {
	err := dto.Scan(src)
	if err != nil {
		logs.ErrorLog(err, name)
	}

	return dto
}

// GetDateFromByte convert date from src into time.Tome
func GetDateFromByte(src []byte, name string) time.Time {
	if len(src) > 0 {
		t, err := time.Parse(time.RFC3339Nano, TrimQuotes(src))
		if err != nil {
			logs.ErrorLog(err, name)
			return time.Time{}
		}
		return t
	}

	return time.Time{}
}

// GetFloat32FromByte convert data from src into float32
func GetFloat32FromByte(ci *pgtype.ConnInfo, src []byte, name string) float32 {
	if len(src) == 0 {
		return 0
	}

	var float4 pgtype.Float4
	err := float4.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return -1
	}

	return float4.Float
}

// GetFloat64FromByte convert data from src into float64
func GetFloat64FromByte(ci *pgtype.ConnInfo, src []byte, name string) float64 {
	if len(src) == 0 {
		return 0
	}

	var float8 pgtype.Float8
	err := float8.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return -1
	}

	return float8.Float
}

// GetFloat32FromByte convert data from src into float32
func GetArrayFloat32FromByte(ci *pgtype.ConnInfo, src []byte, name string) []float32 {
	if len(src) == 0 {
		return nil
	}

	var dto pgtype.Float4Array
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return nil
	}

	res := make([]float32, len(dto.Elements))
	for i, elem := range dto.Elements {
		res[i] = elem.Float
	}

	return res
}

// GetFloat64FromByte convert data from src into float64
func GetArrayFloat64FromByte(ci *pgtype.ConnInfo, src []byte, name string) []float64 {
	if len(src) == 0 {
		return nil
	}

	var dto pgtype.Float8Array
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return nil
	}

	res := make([]float64, len(dto.Elements))
	for i, elem := range dto.Elements {
		res[i] = elem.Float
	}

	return res
}

// GetInt64FromByte convert data from src into int64
func GetInt64FromByte(ci *pgtype.ConnInfo, src []byte, name string) int64 {
	if len(src) == 0 {
		return 0
	}

	var dto pgtype.Int8
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return -1
	}

	return dto.Int
}

// GetInt32FromByte convert data from src into int32
func GetInt32FromByte(ci *pgtype.ConnInfo, src []byte, name string) int32 {
	if len(src) == 0 {
		return 0
	}

	var dto pgtype.Int4
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return -1
	}

	return dto.Int
}

// GetInt32FromByte convert data from src into int32
func GetSqlNullInt32FromByte(ci *pgtype.ConnInfo, src []byte, name string) sql.NullInt32 {
	var dto sql.NullInt32
	err := dto.Scan(src)
	if err != nil {
		logs.ErrorLog(err, name)
	}

	return dto
}

// GetArrayInt16FromByte convert data from src into []int16
func GetArrayInt16FromByte(ci *pgtype.ConnInfo, src []byte, name string) []int16 {
	if len(src) == 0 {
		return nil
	}

	var dto pgtype.Int2Array
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return nil
	}

	res := make([]int16, len(dto.Elements))
	for i, elem := range dto.Elements {
		res[i] = elem.Int
	}

	return res
}

// GetArrayInt32FromByte convert data from src into []int32
func GetArrayInt32FromByte(ci *pgtype.ConnInfo, src []byte, name string) []int32 {
	if len(src) == 0 {
		return nil
	}

	var dto pgtype.Int4Array
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return nil
	}

	res := make([]int32, len(dto.Elements))
	for i, elem := range dto.Elements {
		res[i] = elem.Int
	}

	return res
}

// GetArrayInt64FromByte convert data from src into []int64
func GetArrayInt64FromByte(ci *pgtype.ConnInfo, src []byte, name string) []int64 {
	if len(src) == 0 {
		return nil
	}

	var dto pgtype.Int8Array
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return nil
	}

	res := make([]int64, len(dto.Elements))
	for i, elem := range dto.Elements {
		res[i] = elem.Int
	}

	return res
}

// GetArrayStringFromByte convert data from src into []string
func GetArrayStringFromByte(ci *pgtype.ConnInfo, src []byte, name string) []string {
	if len(src) == 0 {
		return nil
	}

	var dto pgtype.TextArray
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return nil
	}

	res := make([]string, len(dto.Elements))
	for i, elem := range dto.Elements {
		res[i] = elem.String
	}

	return res
}

// GetInt16FromByte convert data from src into int16
func GetInt16FromByte(ci *pgtype.ConnInfo, src []byte, name string) int16 {
	if len(src) == 0 {
		return 0
	}

	var int2 pgtype.Int2
	err := int2.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return -1
	}

	return int2.Int
}

// GetInetFromByte convert data from src into pgtype.Inet
func GetInetFromByte(ci *pgtype.ConnInfo, src []byte, name string) pgtype.Inet {
	if len(src) == 0 {
		return pgtype.Inet{Status: pgtype.Null}
	}

	var dto pgtype.Inet
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return pgtype.Inet{Status: pgtype.Null}
	}

	return dto
}

// GetNumericFromByte convert data from src into Numeric
func GetNumericFromByte(ci *pgtype.ConnInfo, src []byte, name string) Numeric {
	if len(src) == 0 {
		return NewNumericNull()
	}

	var dto pgtype.Numeric
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return NewNumericNull()
	}

	return Numeric{&dto}
}

// GetBoolFromByte convert data from src into bool
func GetBoolFromByte(ci *pgtype.ConnInfo, src []byte, name string) bool {
	if len(src) == 0 {
		return false
	}

	var dto pgtype.Bool
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return false
	}

	return dto.Bool
}

// GetStringFromByte convert data (As Text!) from src into string
func GetStringFromByte(ci *pgtype.ConnInfo, src []byte, name string) string {
	if len(src) == 0 {
		return ""
	}

	// todo: split according psql text type (varchar, bchar, etc.)
	var dto pgtype.Text
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return ""
	}

	return dto.String
}

// GetStringFromByte convert data (As Text!) from src into string
func GetSqlNullStringFromByte(ci *pgtype.ConnInfo, src []byte, name string) sql.NullString {
	var dto sql.NullString
	err := dto.Scan(src)
	if err != nil {
		logs.ErrorLog(err, name)
	}

	return dto
}

// GetJsonFromByte convert data from src into json
func GetJsonFromByte(ci *pgtype.ConnInfo, src []byte, name string) interface{} {
	if len(src) == 0 {
		return nil
	}

	var dto pgtype.JSON
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return nil
	}

	return dto.Bytes
}

// GetTimeFromByte convert data from src into time.Time
func GetTimeFromByte(ci *pgtype.ConnInfo, src []byte, name string) time.Time {
	if len(src) == 0 {
		return time.Time{}
	}

	// todo: split according to psql time types
	var dto pgtype.Timestamptz
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return time.Time{}
	}

	return dto.Time
}

// GetTimeTimeFromByte convert data from src into *time.Time (alias for GetTimeFromByte)
func GetTimeTimeFromByte(ci *pgtype.ConnInfo, src []byte, name string) time.Time {
	return GetTimeFromByte(ci, src, name)
}

// GetRefTimeFromByte convert data from src into *time.Time
func GetRefTimeFromByte(ci *pgtype.ConnInfo, src []byte, name string) *time.Time {
	return new(GetTimeFromByte(ci, src, name))
}

// GetArrayTimeTimeFromByte convert data from src into []time.Time (alias for GetArrayTimeFromByte)
func GetArrayTimeTimeFromByte(ci *pgtype.ConnInfo, src []byte, name string) []time.Time {
	return GetArrayTimeFromByte(ci, src, name)
}

// GetArrayTimeFromByte convert data from src into []time.Time
func GetArrayTimeFromByte(ci *pgtype.ConnInfo, src []byte, name string) []time.Time {
	if len(src) == 0 {
		return nil
	}

	var dto pgtype.TimestampArray
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return nil
	}

	res := make([]time.Time, len(dto.Elements))
	for i, elem := range dto.Elements {
		res[i] = elem.Time
	}

	return res
}

// GetArrayTimeFromByte convert data from src into []time.Time
func GetArrayRefTimeFromByte(ci *pgtype.ConnInfo, src []byte, name string) []*time.Time {
	if len(src) == 0 {
		return nil
	}

	var dto pgtype.TimestampArray
	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return nil
	}

	res := make([]*time.Time, len(dto.Elements))
	for i, elem := range dto.Elements {
		res[i] = &elem.Time
	}

	return res
}

// GetIntervalFromByte convert data from src into []time.Time
func GetIntervalFromByte(ci *pgtype.ConnInfo, src []byte, name string) (dto pgtype.Interval) {
	if len(src) == 0 {
		return
	}

	err := dto.DecodeText(ci, src)
	if err != nil {
		logs.ErrorLog(err, name)
		return
	}

	return
}

// GetArrayByteFromByte convert data from src into sql.RawBytes
func GetArrayByteFromByte(ci *pgtype.ConnInfo, src []byte, name string) (dto sql.RawBytes) {
	return GetRawBytesFromByte(ci, src, name)
}

// GetRawBytesFromByte convert data from src into sql.RawBytes
func GetRawBytesFromByte(ci *pgtype.ConnInfo, src []byte, name string) (dto sql.RawBytes) {
	if len(src) == 0 {
		return
	}

	dto = make([]byte, len(src))
	copy(dto, src)

	return
}
