CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,            
    news_id INT NOT NULL,             
    author VARCHAR(255) NOT NULL,     
    text TEXT NOT NULL,               
    parent_id INT,                    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);