CREATE TABLE IF NOT EXISTS alerts (
	id INT NOT NULL PRIMARY KEY AUTO_INCREMENT,
	client VARCHAR(255) NOT NULL,
	alert_type int NOT NULL,
	message VARCHAR(512) NOT NULL,
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

INSERT INTO alert_types 
	(title, message) 
VALUES 
	('Ratelimit hit', 'You have hit your rate limit of %v requests per minute.');