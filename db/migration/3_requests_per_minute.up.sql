ALTER TABLE clients CHANGE rate_limit_per_second rate_limit_per_minute INT NOT NULL DEFAULT 60;