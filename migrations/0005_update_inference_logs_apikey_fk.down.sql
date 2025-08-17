-- 0005_update_inference_logs_apikey_fk.down.sql
ALTER TABLE inference_logs DROP CONSTRAINT IF EXISTS inference_logs_api_key_id_fkey;
ALTER TABLE inference_logs
    ADD CONSTRAINT inference_logs_api_key_id_fkey
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id);
