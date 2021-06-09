CREATE TABLE compromised_emails (
	email_id serial PRIMARY KEY,
	email VARCHAR(100),
	date_added timestamp default NULL,
	user_id INT,
	CONSTRAINT user_check_email
      FOREIGN KEY(user_id) 
	  REFERENCES accounts(user_id)
	  ON DELETE SET NULL
);

CREATE TABLE domains(
   domain_id serial PRIMARY KEY,
   domain_name VARCHAR(255),
   email_id INT,
   CONSTRAINT dm_emails
      FOREIGN KEY(email_id) 
	  REFERENCES compromised_emails(email_id)
	  ON DELETE CASCADE
);