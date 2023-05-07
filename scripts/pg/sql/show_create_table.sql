CREATE OR REPLACE FUNCTION show_create_table(p_schema_name TEXT, p_table_name TEXT)
RETURNS TEXT AS
$BODY$
DECLARE
v_table_ddl   TEXT;
BEGIN
SELECT
'CREATE TABLE ' || quote_ident(t.table_schema) || '.' || quote_ident(t.table_name) || E'\n(\n' ||
array_to_string(array_agg('  ' || quote_ident(c.column_name) || ' ' || c.data_type || coalesce('(' || c.character_maximum_length || ')', '') || ' ' || coalesce(c.column_default, '') || ' ' || c.is_nullable), E',\n') || E'\n);\n' ||
array_to_string(array_agg(coalesce('ALTER TABLE ' || quote_ident(t.table_schema) || '.' || quote_ident(t.table_name) || ' ADD ' || r.constraint_type || ' ' || quote_ident(r.constraint_name) || ' ' || r.constraint_keys || ';', '')), E'\n')
INTO
v_table_ddl
FROM
information_schema.tables t
JOIN
information_schema.columns c ON t.table_schema = c.table_schema AND t.table_name = c.table_name
LEFT JOIN (
SELECT
k.table_schema,
k.table_name,
k.constraint_name,
t.constraint_type,
' (' || array_to_string(array_agg(quote_ident(k.column_name)), ', ') || ')'
AS constraint_keys
FROM
information_schema.table_constraints t
JOIN
information_schema.key_column_usage k USING (constraint_name, table_schema, table_name)
GROUP BY
k.table_schema,
k.table_name,
k.constraint_name,
t.constraint_type
) r ON t.table_schema = r.table_schema AND t.table_name = r.table_name
WHERE
t.table_schema = p_schema_name
AND t.table_name = p_table_name
GROUP BY
t.table_schema,
t.table_name;

RETURN v_table_ddl;
END;
$BODY$
LANGUAGE plpgsql;
