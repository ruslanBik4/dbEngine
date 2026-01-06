// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tpl

const (
	moduloPgType  = "github.com/jackc/pgtype"
	moduloGoTools = "github.com/ruslanBik4/gotools"
	moduloSql     = "database/sql"
)

const (
	colFormat    = "\n\t%-21s\t%-13s\t`json:\"%s\"`"
	initFormat   = "\n\t\t%-21s:\t%s,"
	scanFormat   = "\n\t\t%s,"
	paramsFormat = `
				[]any{
					%s
				},`
	caseRefFormat = `
	case "%s":
		return &r.%s
`
	caseColFormat = `
	case "%s":
		return r.%s
`
)
