ALTER TABLE api_keys
ADD CONSTRAINT api_keys_user_id_label_key UNIQUE (user_id, label);
