-- Schema for Date Chooser (Phase 1: poll creation only).
-- Idempotent: safe to run against an existing database on every startup so a
-- fresh volume self-initializes and existing data is left untouched.

CREATE TABLE IF NOT EXISTS polls (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    participant_token TEXT NOT NULL UNIQUE,
    admin_token       TEXT NOT NULL UNIQUE,
    title             TEXT NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    organizer_name    TEXT NOT NULL DEFAULT '',
    poll_type         TEXT NOT NULL CHECK (poll_type IN ('all_day', 'date_time')),
    created_at        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS slots (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    poll_id    INTEGER NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    starts_at  TEXT NOT NULL,
    ends_at    TEXT
);

CREATE INDEX IF NOT EXISTS idx_slots_poll_id ON slots(poll_id);
