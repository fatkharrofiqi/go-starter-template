CREATE TABLE roles (
    uuid VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL UNIQUE
);

CREATE TABLE permissions (
    uuid VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL UNIQUE
);

CREATE TABLE user_roles (
    user_uuid VARCHAR NOT NULL,
    role_uuid VARCHAR NOT NULL,
    PRIMARY KEY (user_uuid, role_uuid),
    FOREIGN KEY (user_uuid) REFERENCES users (uuid),
    FOREIGN KEY (role_uuid) REFERENCES roles (uuid)
);

CREATE TABLE role_permissions (
    role_uuid VARCHAR NOT NULL,
    permission_uuid VARCHAR NOT NULL,
    PRIMARY KEY (role_uuid, permission_uuid),
    FOREIGN KEY (role_uuid) REFERENCES roles (uuid),
    FOREIGN KEY (permission_uuid) REFERENCES permissions (uuid)
);

CREATE TABLE user_permissions (
    user_uuid VARCHAR NOT NULL,
    permission_uuid VARCHAR NOT NULL,
    PRIMARY KEY (user_uuid, permission_uuid),
    FOREIGN KEY (user_uuid) REFERENCES users (uuid),
    FOREIGN KEY (permission_uuid) REFERENCES permissions (uuid)
);