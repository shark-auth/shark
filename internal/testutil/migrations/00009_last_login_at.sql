-- +goose Up
-- 00009: Add last_login_at to users table for dashboard Last Active column.
ALTER TABLE users ADD COLUMN last_login_at TEXT;
