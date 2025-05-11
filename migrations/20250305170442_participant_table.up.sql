CREATE TABLE IF NOT EXISTS participants(
    participant_id SERIAL PRIMARY KEY ,
    conference_id uuid NOT NULL ,
    user_id INT NOT NULL UNIQUE ,
    role ROLE_ENUM DEFAULT 'viewer',
    joined_at DATE DEFAULT CURRENT_TIMESTAMP,
    left_at DATE,
    FOREIGN KEY (conference_id) REFERENCES conferences(conference_id) ON DELETE CASCADE ,
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);