// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

// ROUTINE_TYPE_PROC & ROUTINE_TYPE_FUNC are named POsqgreSQL routine
const (
	ROUTINE_TYPE_PROC = "PROCEDURE"
	ROUTINE_TYPE_FUNC = "FUNCTION"
)

const (
	sqlDBSetting = `SELECT current_database() as db_name, current_schema() as db_schema,
       current_setting('work_mem') as work_mem, current_setting('datestyle') as datestyle,
       current_setting('port') as db_port,
       current_user as db_user`
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
	sqlGetTable = `SELECT table_name, table_type,
						COALESCE(pg_catalog.col_description((SELECT ('"' || TABLE_NAME || '"')::regclass::oid), 0), '')
							AS comment
						FROM INFORMATION_SCHEMA.tables
						WHERE table_schema = 'public' AND table_name = $1
					union
						select c.relname, 'MATERIALIZED VIEW', COALESCE(pg_catalog.obj_description(c.oid, 'pg_class'), '')
						FROM pg_catalog.pg_class c
						WHERE c.relkind = 'm' AND c.relname = $1`
	sqlRoutineList = `select specific_name, routine_name, routine_type, data_type, type_udt_name, d.description
					FROM INFORMATION_SCHEMA.routines r JOIN pg_proc p ON p.proname = r.routine_name
						 LEFT JOIN pg_description d
								   ON d.objoid = p.oid
                                   left join pg_language l on p.prolang = l.oid
					WHERE specific_schema = 'public' and prokind != 'a'  and data_type != 'trigger' and l.lanname = 'plpgsql'`
	sqlGetTablesColumns = `SELECT c.column_name, data_type, column_default,  is_nullable='YES' is_nullable, 
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
			WHERE ( k.table_name=c.table_name AND k.column_name = c.column_name)
		) as keys,
		c.ordinal_position
FROM INFORMATION_SCHEMA.COLUMNS c
WHERE c.table_schema='public' AND c.table_name=$1
UNION ALL
    SELECT a.attname::information_schema.sql_identifier, 
       CASE
           WHEN t.typtype = 'd'::"char" THEN
               CASE
                   WHEN bt.typelem <> 0::oid AND bt.typlen = '-1'::integer THEN 'ARRAY'::text
                   WHEN nbt.nspname = 'pg_catalog'::name THEN format_type(t.typbasetype, NULL::integer)
                   ELSE 'USER-DEFINED'::text
                   END
           ELSE
               CASE
                   WHEN t.typelem <> 0::oid AND t.typlen = '-1'::integer THEN 'ARRAY'::text
                   WHEN nt.nspname = 'pg_catalog'::name THEN format_type(a.atttypid, NULL::integer)
                   ELSE 'USER-DEFINED'::text
                   END
           END::information_schema.character_data                                                                                      AS data_type,
		NULL,  true,  '', -1, 
        COALESCE(bt.typname, t.typname)::information_schema.sql_identifier,
		COALESCE(pg_catalog.col_description((SELECT ('"' || $1 || '"')::regclass::oid), ordinal_position::int), ''),
		NULL::json, ordinal_position
	FROM pg_attribute a
		JOIN LATERAL CAST(a.attnum as information_schema.cardinal_number) ordinal_position ON true
         JOIN (pg_class c JOIN pg_namespace nc ON c.relnamespace = nc.oid) ON a.attrelid = c.oid
         JOIN (pg_type t JOIN pg_namespace nt ON t.typnamespace = nt.oid) ON a.atttypid = t.oid
		LEFT JOIN (pg_type bt JOIN pg_namespace nbt ON bt.typnamespace = nbt.oid) 
				ON t.typtype = 'd'::"char" AND t.typbasetype = bt.oid
	WHERE nc.nspname='public' AND c.relkind = 'm' AND c.relname=$1
  			AND a.attnum > 0  AND NOT a.attisdropped
			  AND (pg_has_role(c.relowner, 'USAGE'::text) OR
				   has_column_privilege(c.oid, a.attnum, 'SELECT, INSERT, UPDATE, REFERENCES'::text))
ORDER BY ordinal_position`
	sqlGetFuncParams = `SELECT coalesce(parameter_name, 'noName') as parameter_name, data_type, udt_name,
								COALESCE(character_set_name, '') as character_set_name,
								COALESCE(character_maximum_length, -1) as character_maximum_length, 
								parameter_default as parameter_default,
								ordinal_position, parameter_mode
						FROM INFORMATION_SCHEMA.parameters
						WHERE specific_schema='public' AND specific_name=$1`
	sqlGetColumnAttr = `SELECT data_type, 
							column_default,
							is_nullable='YES' as is_nullable, 
							COALESCE(character_set_name, '') as character_set_name,
							COALESCE(character_maximum_length, -1) as character_maximum_length, 
							udt_name,
  							(select json_object_agg( k.constraint_name,
								CASE WHEN kcu.table_name IS NULL THEN NULL
								   ELSE json_build_object('parent', kcu.table_name, 'column', kcu.column_name,
								   'update_rule', rc.update_rule, 'delete_rule', rc.delete_rule)
							    END)
							FROM  INFORMATION_SCHEMA.key_column_usage k
								LEFT JOIN INFORMATION_SCHEMA.referential_constraints rc using(constraint_name)
								LEFT JOIN INFORMATION_SCHEMA.key_column_usage kcu ON rc.unique_constraint_name = kcu.constraint_name
							WHERE ( k.table_name=c.table_name AND k.column_name = c.column_name)
							) as keys,
							COALESCE(pg_catalog.col_description((SELECT ('"' || $1 || '"')::regclass::oid), c.ordinal_position::int), '')
							   AS column_comment
						FROM INFORMATION_SCHEMA.COLUMNS C
						WHERE C.table_schema='public' AND C.table_name=$1 AND C.COLUMN_NAME = $2`
	sqlTypesList = `SELECT pg_type.oid,  typname, typtype,
		CASE pg_type.typtype
          WHEN 'r' THEN (select json_build_array(json_build_object(
                                            'name',
                                           'upper',
                                           'type',
                                           g.typname
                                         ),
                                 json_build_object(
                                         'name',
                                         'lower',
                                         'type',
                                         g.typname
                                 ))
                      from pg_range r JOIN pg_type g ON r.rngsubtype = g.oid
                                      left join pg_class c1 on c1.relname = typname
                      where r.rngtypid = pg_type.oid)
        WHEN 'd' THEN (select json_build_array(json_build_object(
                                            'name', 'domain',
                                            'type', (select bt.typname FROM pg_type bt where  pg_type.typbasetype = bt.oid)
                                 )))
            
		ELSE (select json_agg( 
						json_build_object(
							'name',
							a.attname,
						   'type',
						   CASE
							WHEN t.typtype = 'd'::"char" THEN
								CASE
									WHEN bt.typelem <> 0::oid AND bt.typlen = '-1'::integer OR nbt.nspname = 'pg_catalog'::name 
										THEN COALESCE(bt.typname, t.typname)::information_schema.sql_identifier
									ELSE 'USER-DEFINED'::text
									END
							ELSE
								CASE
									WHEN t.typelem <> 0::oid AND t.typlen = '-1'::integer OR nt.nspname = 'pg_catalog'::name 
										THEN COALESCE(bt.typname, t.typname)::information_schema.sql_identifier
									ELSE 'USER-DEFINED'::text
									END
							END::information_schema.character_data,
							'is_not_null'::text,
							a.attnotnull
						) order by a.attnum
                   )
        	from pg_attribute a
                 JOIN (pg_type t JOIN pg_namespace nt ON t.typnamespace = nt.oid) ON a.atttypid = t.oid
                 LEFT JOIN (pg_type bt JOIN pg_namespace nbt ON bt.typnamespace = nbt.oid)
                           ON t.typtype = 'd'::"char" AND t.typbasetype = bt.oid       
        	where a.attrelid = c.oid
        ) END as attr, relkind,
       array(select e.enumlabel FROM pg_enum e where e.enumtypid = pg_type.oid)::varchar[] as enumerates
FROM pg_type LEFT JOIN (pg_class c JOIN pg_namespace nc ON c.relnamespace = nc.oid) on relname = typname
where typtype = ANY(array['e','d']) or (nc.nspname='public' AND c.relkind = ANY(array['d','c']))`

	sqlGetIndexes = `SELECT i.relname as index_name,
	   COALESCE( pg_get_expr( ix.indexprs, ix.indrelid ), '') as ind_expr,
       ix.indisunique as ind_unique,
       array_agg(a.attname order by array_positions(ix.indkey, a.attnum)) filter ( where a.attname > '' )  :: text[] as column_names
FROM pg_index ix left join pg_class t on t.oid = ix.indrelid
     left join pg_class i on i.oid = ix.indexrelid
     left join  pg_attribute a on (a.attrelid = t.oid AND a.attnum = ANY(ix.indkey))
where t.relname = $1
group by 1,2,3
UNION
SELECT
    tc.constraint_name, 
    '',
    false,
    array_agg(kcu.column_name)
FROM
    information_schema.table_constraints AS tc
        JOIN information_schema.key_column_usage AS kcu
             USING (constraint_schema, constraint_name, table_name)
WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_name=$1
group by 1,2,3
order by 1`
)
