CREATE SCHEMA IF NOT EXISTS users_schema;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE users_schema.user_role AS ENUM ('ROLE_UNSPECIFIED', 'ROLE_ADMIN', 'ROLE_CUSTOMER');

CREATE TABLE IF NOT EXISTS users_schema.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    avatar_url VARCHAR(255),
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role users_schema.user_role NOT NULL,
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    city VARCHAR(100) NOT NULL DEFAULT 'Москва',
    street VARCHAR(100) NOT NULL DEFAULT 'Арбат',
    building VARCHAR(10) NOT NULL DEFAULT '44с1',
    porch INTEGER,
    floor INTEGER,
    flat INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users_schema.users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users_schema.users(role);

ALTER DATABASE "Hailow" SET TIMEZONE TO 'Europe/Moscow';
