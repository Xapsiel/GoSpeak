CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS conferences(
    conference_id uuid DEFAULT gen_random_uuid() PRIMARY KEY ,
    title VARCHAR(255) NOT NULL ,
    description TEXT NOT NULL DEFAULT '',
    creater_id INT NOT NULL  ,
    start_time DATE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_time DATE ,
    status STATUS_ENUM,
    join_url TEXT UNIQUE NOT NULL ,
    password TEXT,
    max_participants int,
    UNIQUE (title, description, creater_id),
    FOREIGN KEY (creater_id) REFERENCES users(user_id)

);