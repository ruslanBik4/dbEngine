// Copyright 2018 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"bytes"
	"strings"
	"time"

	"github.com/jackc/pgtype"
	"github.com/pkg/errors"
	"github.com/ruslanBik4/logs"
)

type (
	CrossDataRecord struct {
		TradesMade           int64     `json:"trades_made"`
		TotalVolume          int64     `json:"total_volume"`
		TotalGross           float64   `json:"total_gross"`
		TotalGrossProfitDays float64   `json:"total_gross_profit_days"`
		TotalGrossLossDays   float64   `json:"total_gross_loss_days"`
		TotalNetProfitDays   float64   `json:"total_net_profit_days"`
		TotalNetLossDays     float64   `json:"total_net_loss_days"`
		MaxTradeProfit       float64   `json:"max_trade_profit"`
		MaxTradeLoss         float64   `json:"max_trade_loss"`
		AvgTradeProfit       float64   `json:"avg_trade_profit"`
		AvgTradeLoss         float64   `json:"avg_trade_loss"`
		AvgTimeProfit        string    `json:"avg_time_profit"` // time.Time
		AvgTimeLoss          string    `json:"avg_time_loss"`   // time.Time
		TradesMadeProfit     int32     `json:"trades_made_profit"`
		TradesMadeLoss       int32     `json:"trades_made_loss"`
		LongProfitCount      int32     `json:"long_profit_count"`
		LongLossCount        int32     `json:"long_loss_count"`
		MaxProfitDaysRow     int64     `json:"max_profit_days_row"`
		SumPlus              float64   `json:"sum_plus"`
		QntPlus              int32     `json:"qnt_plus"`
		MaxLossDaysRow       int64     `json:"max_loss_days_row"`
		SumMinus             float64   `json:"sum_minus"`
		QntMinus             int32     `json:"qnt_minus"`
		ProfitDays           int32     `json:"profit_days"`
		LossDays             int32     `json:"loss_days"`
		MaxGrossChain        float64   `json:"max_gross_chain"`
		MinGrossChain        float64   `json:"min_gross_chain"`
		ShortProfitCount     int32     `json:"short_profit_count"`
		ShortLossCount       int32     `json:"short_loss_count"`
		TotalNet             float64   `json:"total_net"`
		NetPerShare          float64   `json:"net_per_share"`
		NetPerTrade          float64   `json:"net_per_trade"`
		StopLossOpenings     int32     `json:"stop_loss_openings"`
		ClosedPositive       int32     `json:"closed_positive"`
		ClosedFlat           int32     `json:"closed_flat"`
		DecreasedLossBy      int32     `json:"decreased_loss_by"`
		Last3StopLosses      int32     `json:"last_3_stop_losses"`
		BestTradeNet         float64   `json:"best_trade_net"`
		BestDayNet           float64   `json:"best_day_net"`
		BestWeekNet          float64   `json:"best_week_net"`
		BestMonthNet         float64   `json:"best_month_net"`
		BestMonth            time.Time `json:"best_month"`
		BestWeek             time.Time `json:"best_week"`
		BestDay              time.Time `json:"best_day"`
		BestTrade            string    `json:"best_trade"`
		NetPerDay            float64   `json:"net_per_day"`
		MaxDayProfit         float64   `json:"max_day_profit"`
		MaxDayLoss           float64   `json:"max_day_loss"`
		AvgDayLoss           float64   `json:"avg_day_loss"`
		AvgDayProfit         float64   `json:"avg_day_profit"`
		AvgHighFixRate       float64   `json:"avg_high_fix_rate"`
		AvgVolumeProfitDays  float64   `json:"avg_volume_profit_days"`
		AvgVolumeLossDays    float64   `json:"avg_volume_loss_days"`
		TotalTradingDays     int32     `json:"total_trading_days"`
		BestDayDesc          string    `json:"best_day_desc"`
		MaxProfitWeekRow     int16     `json:"max_profit_week_row"`
		MaxProfitMonthRow    int16     `json:"max_profit_month_row"`
		MaxLossMonthRow      int16     `json:"max_loss_month_row"`
		MaxLossWeekRow       int16     `json:"max_loss_week_row"`
		MinVolume            int64     `json:"min_volume"`
		MaxVolume            int64     `json:"max_volume"`
		AvgNetProfitDays     float64   `json:"avg_net_profit_days"`
		AvgNetLossDays       float64   `json:"avg_net_loss_days"`
		MaxGross             float64   `json:"max_gross"`
		MinGross             float64   `json:"min_gross"`
	}

	CrossData struct {
		CrossDataRecord
		Status      pgtype.Status `json:"-"`
		convertErrs []string
	}
)

func (dst *CrossData) Set(src interface{}) error {
	// untyped nil and typed nil interfaces are different
	if src == nil {
		*dst = CrossData{Status: pgtype.Null}
		return nil
	}

	switch value := src.(type) {
	case CrossData:
		*dst = CrossData{
			CrossDataRecord: value.CrossDataRecord,
			Status:          pgtype.Present,
		}

	default:
	}

	return nil
}

func (dst *CrossData) Get() interface{} {
	switch dst.Status {
	case pgtype.Present:
		return *dst
	case pgtype.Null:
		return nil
	default:
		return dst.Status
	}
}

func (src *CrossData) AssignTo(dst interface{}) error {
	switch src.Status {
	case pgtype.Present:
		switch v := dst.(type) {
		case *CrossData:
			(*v).CrossDataRecord = src.CrossDataRecord
			return nil

		default:
			if nextDst, retry := pgtype.GetAssignToDstType(dst); retry {
				return src.AssignTo(nextDst)
			}
		}
	case pgtype.Null:
		return pgtype.NullAssignTo(dst)
	}

	return errors.Errorf("cannot decode %v into %T", src, dst)
}

func (dst *CrossData) DecodeText(ci *pgtype.ConnInfo, src []byte) error {
	if len(src) == 0 {
		*dst = CrossData{Status: pgtype.Null}
		return nil
	}

	srcPart := bytes.Split(src[1:len(src)-1], []byte(","))

	record := CrossDataRecord{
		TradesMade:           GetInt64FromByte(ci, srcPart[0], "tradesMade"),
		TotalVolume:          GetInt64FromByte(ci, srcPart[1], "totalVolume"),
		TotalGross:           GetFloat64FromByte(ci, srcPart[2], "totalGross"),
		TotalGrossProfitDays: GetFloat64FromByte(ci, srcPart[3], "TotalGrossProfitDays"),
		TotalGrossLossDays:   GetFloat64FromByte(ci, srcPart[4], "TotalGrossLossDays"),
		MaxTradeProfit:       GetFloat64FromByte(ci, srcPart[5], "MaxTradeProfit"),
		MaxTradeLoss:         GetFloat64FromByte(ci, srcPart[6], "MaxTradeLoss"),
		AvgTradeProfit:       GetFloat64FromByte(ci, srcPart[7], "AvgTradeProfit"),
		AvgTradeLoss:         GetFloat64FromByte(ci, srcPart[8], "AvgTradeLoss"),
		AvgTimeProfit:        trimQuotes(srcPart[9]),
		AvgTimeLoss:          trimQuotes(srcPart[10]),
		TradesMadeProfit:     GetInt32FromByte(ci, srcPart[11], "TradesMadeProfit"),
		TradesMadeLoss:       GetInt32FromByte(ci, srcPart[12], "TradesMadeLoss"),
		LongProfitCount:      GetInt32FromByte(ci, srcPart[13], "LongProfitCount"),
		LongLossCount:        GetInt32FromByte(ci, srcPart[14], "LongLossCount"),
		MaxProfitDaysRow:     GetInt64FromByte(ci, srcPart[15], "maxProfitDaysRow"),
		SumPlus:              GetFloat64FromByte(ci, srcPart[16], "sumPlus"),
		QntPlus:              GetInt32FromByte(ci, srcPart[17], "qntPlus"),
		MaxLossDaysRow:       GetInt64FromByte(ci, srcPart[18], "maxLossDaysRow"),
		SumMinus:             GetFloat64FromByte(ci, srcPart[19], "sumMinus"),
		QntMinus:             GetInt32FromByte(ci, srcPart[20], "qntMinus"),
		ProfitDays:           GetInt32FromByte(ci, srcPart[21], "profitDays"),
		LossDays:             GetInt32FromByte(ci, srcPart[22], "lossDays"),
		MaxGrossChain:        GetFloat64FromByte(ci, srcPart[23], "maxGrossChain"),
		MinGrossChain:        GetFloat64FromByte(ci, srcPart[24], "minGrossChain"),
		ShortProfitCount:     GetInt32FromByte(ci, srcPart[25], "ShortProfitCount"),
		ShortLossCount:       GetInt32FromByte(ci, srcPart[26], "ShortLossCount"),
		TotalNet:             GetFloat64FromByte(ci, srcPart[27], "TotalNet"),
		NetPerShare:          GetFloat64FromByte(ci, srcPart[28], "NetPerShare"),
		NetPerTrade:          GetFloat64FromByte(ci, srcPart[29], "NetPerTrade"),
		StopLossOpenings:     GetInt32FromByte(ci, srcPart[30], "StopLossOpenings"),
		ClosedPositive:       GetInt32FromByte(ci, srcPart[31], "ClosedPositive"),
		ClosedFlat:           GetInt32FromByte(ci, srcPart[32], "ClosedFlat"),
		DecreasedLossBy:      GetInt32FromByte(ci, srcPart[33], "DecreasedLossBy"),
		Last3StopLosses:      GetInt32FromByte(ci, srcPart[34], "Last3StopLosses"),
		BestTradeNet:         GetFloat64FromByte(ci, srcPart[35], "BestTradeNet"),
		BestDayNet:           GetFloat64FromByte(ci, srcPart[36], "BestDayNet"),
		BestWeekNet:          GetFloat64FromByte(ci, srcPart[37], "BestWeekNet"),
		BestMonthNet:         GetFloat64FromByte(ci, srcPart[38], "BestMonthNet"),
		BestMonth:            GetDateFromByte(srcPart[39], "BestMonth"),
		BestWeek:             GetDateFromByte(srcPart[40], "BestWeek"),
		BestDay:              GetDateFromByte(srcPart[41], "BestDay"),
		BestTrade:            trimQuotes(srcPart[42]),
		NetPerDay:            GetFloat64FromByte(ci, srcPart[43], "NetPerDay"),
		MaxDayProfit:         GetFloat64FromByte(ci, srcPart[44], "MaxDayProfit"),
		MaxDayLoss:           GetFloat64FromByte(ci, srcPart[45], "MaxDayLoss"),
		AvgDayLoss:           GetFloat64FromByte(ci, srcPart[46], "AvgDayLoss"),
		AvgDayProfit:         GetFloat64FromByte(ci, srcPart[47], "AvgDayProfit"),
		AvgHighFixRate:       GetFloat64FromByte(ci, srcPart[48], "AvgHighFixRate"),
		TotalTradingDays:     GetInt32FromByte(ci, srcPart[49], "TotalTradingDays"),
		BestDayDesc:          trimQuotes(srcPart[50]),
		MaxProfitWeekRow:     GetInt16FromByte(ci, srcPart[51], "MaxProfitWeekRow"),
		MaxProfitMonthRow:    GetInt16FromByte(ci, srcPart[52], "MaxProfitMonthRow"),
		MaxLossMonthRow:      GetInt16FromByte(ci, srcPart[53], "MaxLossMonthRow"),
		MaxLossWeekRow:       GetInt16FromByte(ci, srcPart[54], "MaxLossWeekRow"),
		AvgVolumeProfitDays:  GetFloat64FromByte(ci, srcPart[55], "AvgVolumeProfitDays"),
		AvgVolumeLossDays:    GetFloat64FromByte(ci, srcPart[56], "AvgVolumeLossDays"),
		TotalNetProfitDays:   GetFloat64FromByte(ci, srcPart[57], "TotalNetProfitDays"),
		TotalNetLossDays:     GetFloat64FromByte(ci, srcPart[58], "TotalNetLossDays"),
		MinVolume:            GetInt64FromByte(ci, srcPart[59], "MinVolume"),
		MaxVolume:            GetInt64FromByte(ci, srcPart[60], "MaxVolume"),
		AvgNetProfitDays:     GetFloat64FromByte(ci, srcPart[61], "AvgNetProfitDays"),
		AvgNetLossDays:       GetFloat64FromByte(ci, srcPart[62], "AvgNetLossDays"),
		MaxGross:             GetFloat64FromByte(ci, srcPart[63], "MaxGross"),
		MinGross:             GetFloat64FromByte(ci, srcPart[64], "MinGross"),
	}

	*dst = CrossData{CrossDataRecord: record, Status: pgtype.Present}

	return nil
}

func trimQuotes(src []byte) string {

	return strings.Trim(string(src), `"`)
}

func GetDateFromByte(src []byte, name string) time.Time {
	if str := string(src); str > "" {
		t, err := time.Parse("2006-01-02 00:00:00", strings.Trim(str, `"`))
		if err != nil {
			logs.ErrorLog(err, name)
			return time.Time{}
		}
		return t
	}

	return time.Time{}
}

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
