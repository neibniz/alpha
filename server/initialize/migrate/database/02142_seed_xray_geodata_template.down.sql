BEGIN;

DELETE FROM "xray_templates"
WHERE "type" = 'geodata' AND "name" = 'Geodata';

COMMIT;
