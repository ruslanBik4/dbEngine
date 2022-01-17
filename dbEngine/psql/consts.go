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
					union
						select c.relname, 'MATERIALIZED VIEW', COALESCE(pg_catalog.obj_description(c.oid, 'pg_class'), '')
						FROM pg_catalog.pg_class c
						WHERE c.relkind = 'm'
					order by 1`
	sqlRoutineList = `select specific_name, routine_name, routine_type, data_type, type_udt_name
					FROM INFORMATION_SCHEMA.routines
					WHERE specific_schema = 'public'`
	sqlGetTablesColumns = `SELECT c.column_name, data_type, column_default, 
		is_nullable='YES' is_nullable, 
        COALESCE(character_set_name, '') character_set_name,
		COALESCE(character_maximum_length, -1) character_maximum_length, 
        udt_name,
		COALESCE(pg_catalog.col_description((SELECT ('"' || $1 || '"')::regclass::oid), c.ordinal_position::int), '')
							   AS column_comment,
  		(select json_object_agg( k.constraint_name,
			CASE WHEN kcu.table_name IS NULL THEN NULL
               ELSE json_build_object('parent', kcu.table_name, 'column', kcu.column_name,
               'update_rule', rc.update_rule, 'delete_rule', rc.delete_rule)
           END)
		FROM  INFORMATION_SCHEMA.key_column_usage k
		    LEFT JOIN INFORMATION_SCHEMA.referential_constraints rc using(constraint_name)
			LEFT JOIN INFORMATION_SCHEMA.key_column_usage kcu ON rc.unique_constraint_name = kcu.constraint_name
		WHERE ( k.table_name=c.table_name AND k.column_name = c.column_name)) as keys
FROM INFORMATION_SCHEMA.COLUMNS c
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
	   COALESCE( pg_get_expr( ix.indexprs, ix.indrelid ), '') as ind_expr,
       ix.indisunique as ind_unique,
       array_agg(a.attname order by a.attnum) filter ( where a.attname > '' )  :: text[] as column_names
FROM pg_index ix left join pg_class t on t.oid = ix.indrelid
     left join pg_class i on i.oid = ix.indexrelid
     left join  pg_attribute a on (a.attrelid = t.oid AND a.attnum = ANY(ix.indkey))
where t.relname = $1
group by 1,2,3
order by 1`
)
