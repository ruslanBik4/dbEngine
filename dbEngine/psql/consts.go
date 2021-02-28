// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

const (
	sqlTableList = `SELECT table_name, table_type,
					COALESCE(pg_catalog.col_description((SELECT ('"' || TABLE_NAME || '"')::regclass::oid), 0), '')
				   		AS comment
					FROM INFORMATION_SCHEMA.tables
					WHERE table_schema = 'public' 
					order by 1`
	sqlFuncList = `select specific_name, routine_name, routine_type, data_type, type_udt_name
					FROM INFORMATION_SCHEMA.routines
					WHERE specific_schema = 'public'`
	sqlGetTablesColumns = `SELECT c.column_name, data_type, column_default,
								is_nullable='YES', COALESCE(character_set_name, ''),
								COALESCE(character_maximum_length, -1), udt_name,
								k.constraint_name, k.position_in_unique_constraint is null,
								COALESCE(pg_catalog.col_description((SELECT ('"' || $1 || '"')::regclass::oid), c.ordinal_position::int), '')
							   AS column_comment
							FROM INFORMATION_SCHEMA.COLUMNS c
								LEFT JOIN INFORMATION_SCHEMA.key_column_usage k 
									on (k.table_name=c.table_name AND k.column_name = c.column_name)
							WHERE c.table_schema='public' AND c.table_name=$1`
	sqlGetFuncParams = `SELECT coalesce(parameter_name, 'noName') as parameter_name, data_type, udt_name,
								COALESCE(character_set_name, '') as character_set_name,
								COALESCE(character_maximum_length, -1) as character_maximum_length, 
								COALESCE(parameter_default, '') as parameter_default,
								ordinal_position, parameter_mode
								FROM INFORMATION_SCHEMA.parameters
						WHERE specific_schema='public' AND specific_name=$1`
	sqlGetColumnAttr = `SELECT data_type, 
							column_default,
							is_nullable='YES' as is_nullable, 
							COALESCE(character_set_name, '') as character_set_name,
							COALESCE(character_maximum_length, -1) as character_maximum_length, 
							udt_name,
							COALESCE(pg_catalog.col_description((SELECT ('"' || $1 || '"')::regclass::oid), c.ordinal_position::int), '')
							   AS column_comment
						FROM INFORMATION_SCHEMA.COLUMNS C
						WHERE C.table_schema='public' AND C.table_name=$1 AND C.COLUMN_NAME = $2`
	sqlTypeExists = "SELECT exists(select null FROM pg_type WHERE typname::text=ANY($1))"
	sqlGetTypes   = "SELECT typname, oid FROM pg_type WHERE typname::text=ANY($1)"
	sqlTypesList  = "SELECT typname, typcategory FROM pg_type"
	sqlGetIndexes = `SELECT i.relname as index_name, 
		t.relname, 
	   COALESCE( pg_get_expr( ix.indexprs, ix.indrelid ), '') as ind_expr,
       ix.indisunique as ind_unique,
       array_agg(a.attname order by a.attnum) filter ( where a.attname > '' )  :: text[] as column_names
FROM pg_index ix left join pg_class t on t.oid = ix.indrelid
     left join pg_class i on i.oid = ix.indexrelid
     left join  pg_attribute a on (a.attrelid = t.oid AND a.attnum = ANY(ix.indkey))
where t.relname = $1
group by 1,2,3,4
order by 1`
)
