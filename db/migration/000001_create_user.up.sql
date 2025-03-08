CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    uid VARCHAR NOT NULL UNIQUE,
    email VARCHAR UNIQUE,
    password VARCHAR,
    name VARCHAR
);

CREATE INDEX idx_users_deleted_at ON users (deleted_at);