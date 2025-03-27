CREATE TABLE users (
    uuid VARCHAR PRIMARY KEY NOT NULL UNIQUE,
    name VARCHAR,
    email VARCHAR UNIQUE,
    password VARCHAR,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_deleted_at ON users (deleted_at);