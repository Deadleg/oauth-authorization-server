CREATE TABLE IF NOT EXISTS alerts (
	client VARCHAR(255) NOT NULL,
	alert_type int NOT NULL,
	time TIMESTAMP NOT NULL DEFAULT NOW(),
	FOREIGN KEY (client)
		REFERENCES clients(id)
		ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS alert_types (
	id INT NOT NULL PRIMARY KEY AUTO_INCREMENT,
	title VARCHAR(255) NOT NULL UNIQUE,
	message VARCHAR(512) NOT NULL
);

INSERT INTO alert_type 
	(title, message) 
VALUES 
	('Ratelimit hit', 'You have hit your rate limit of %v requests per minute.');