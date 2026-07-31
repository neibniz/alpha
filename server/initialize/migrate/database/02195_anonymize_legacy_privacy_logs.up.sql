UPDATE "system_logs"
SET "ip" = CASE WHEN "ip" = '' THEN '' ELSE '***' END,
    "user_agent" = CASE WHEN "user_agent" = '' THEN '' ELSE '***' END,
    "identifier" = CASE WHEN "identifier" = '' THEN '' ELSE '***' END,
    "content" = CASE WHEN "type" IN (10, 11, 20, 30, 31, 35) THEN '{}' ELSE "content" END;

UPDATE "subscribe_access_logs"
SET "client_ip" = CASE WHEN "client_ip" = '' THEN '' ELSE '***' END,
    "user_agent" = CASE WHEN "user_agent" = '' THEN '' ELSE '***' END;

UPDATE "order"
SET "client_ip" = CASE WHEN "client_ip" = '' THEN '' ELSE '***' END;

UPDATE "user_device_online_record"
SET "identifier" = CASE WHEN "identifier" = '' THEN '' ELSE '***' END;
