CREATE TABLE books (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    title            TEXT    NOT NULL,
    author           TEXT    NOT NULL,
    isbn             TEXT    NOT NULL UNIQUE,
    total_copies     INTEGER NOT NULL CHECK (total_copies >= 0),
    available_copies INTEGER NOT NULL CHECK (available_copies >= 0),
    created_at       TIMESTAMP NOT NULL
);

CREATE INDEX books_title_idx ON books (title);
