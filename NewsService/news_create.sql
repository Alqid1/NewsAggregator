CREATE TABLE IF NOT EXISTS news (
    id SERIAL PRIMARY KEY,            
    title TEXT NOT NULL,             
    author VARCHAR(255) NOT NULL, 
	content text NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);