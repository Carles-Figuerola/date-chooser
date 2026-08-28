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

-- Phase 2: voting end-to-end. participants + responses are normalized so a
-- resubmit updates in place (no edit history in v1) and never overwrites a
-- different participant merely because the display name collides.
CREATE TABLE IF NOT EXISTS participants (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    poll_id      INTEGER NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    comment      TEXT NOT NULL DEFAULT '',
    cookie_token TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_participants_poll_id ON participants(poll_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_participants_poll_cookie ON participants(poll_id, cookie_token);

CREATE TABLE IF NOT EXISTS responses (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    participant_id INTEGER NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    slot_id        INTEGER NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    answer         TEXT NOT NULL CHECK (answer IN ('yes', 'no', 'maybe'))
);

CREATE INDEX IF NOT EXISTS idx_responses_participant_id ON responses(participant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_responses_participant_slot ON responses(participant_id, slot_id);

-- Phase 7: instance-admin page. A single-row-per-key settings store; the
-- instance-admin secret (key='instance_admin_secret') is the only value
-- kept here today, generated on first startup if absent (ADMIN-01) and
-- never exposed or rotated through the web UI — sqlite3-only.
CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- A fresh session row is created on every successful /admin/login (not
-- reused across logins); each is valid for 24h from created_at, checked at
-- auth time regardless of whether this row has been pruned yet. Pruning
-- itself (deleting rows older than 24h) is lazy garbage collection, run on
-- every /admin and /admin/login request — not the security boundary.
CREATE TABLE IF NOT EXISTS admin_sessions (
    token      TEXT PRIMARY KEY,
    created_at TEXT NOT NULL
);
