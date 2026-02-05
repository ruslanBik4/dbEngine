create DATABASE test;


alter table users drop column geo, drop column id_themes, drop column avatar, drop column hash;
update users set avatar=DEFAULT WHERE avatar IS NULL;
ALTER TABLE users ADD COLUMN id_themes    integer              default 0, ADD COLUMN hash         bigint              , ADD COLUMN geo point, ADD COLUMN avatar          bytea              default '\x';