CREATE TABLE IF NOT EXISTS users
(
    user_id SERIAL PRIMARY KEY,
    email varchar(255) UNIQUE NOT NULL ,
    password_hash TEXT NOT NULL ,
    full_name VARCHAR(255) NOT NULL ,
    avatar_url TEXT NOT NULL DEFAULT '',
    created_at DATE DEFAULT current_timestamp,
    last_login DATE,
    is_online BOOLEAN DEFAULT FALSE
);
CREATE  TYPE   STATUS_ENUM as ENUM('scheduled', 'live', 'ended')  ;
CREATE  TYPE ROLE_ENUM as ENUM('host', 'streamer', 'viewer');
CREATE  TYPE CONTENT_TYPE AS ENUM('text','file','link') ;
