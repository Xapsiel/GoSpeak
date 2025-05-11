
CREATE TABLE IF NOT EXISTS messages(
    message_id SERIAL PRIMARY KEY ,
    conference_id UUID NOT NULL ,
    sender_id INT NOT NULL ,
    content TEXT NOT NULL ,
    sent_at DATE DEFAULT CURRENT_TIMESTAMP,
    content_type TEXT DEFAULT 'text',
    FOREIGN KEY (conference_id) REFERENCES conferences(conference_id) ON DELETE CASCADE ,
    FOREIGN KEY (sender_id) REFERENCES users(user_id)

);