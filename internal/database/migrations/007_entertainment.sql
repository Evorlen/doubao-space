CREATE TABLE IF NOT EXISTS entertainment_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_type TEXT NOT NULL CHECK (media_type IN ('novel', 'series', 'movie')),
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    creator TEXT NOT NULL DEFAULT '',
    release_year INTEGER CHECK (release_year IS NULL OR release_year BETWEEN 1000 AND 9999),
    source TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'in_progress', 'completed', 'paused', 'dropped')),
    progress_current INTEGER NOT NULL DEFAULT 0 CHECK (progress_current >= 0),
    progress_total INTEGER CHECK (progress_total IS NULL OR progress_total >= 1),
    rating INTEGER NOT NULL DEFAULT 0 CHECK (rating BETWEEN 0 AND 5),
    review TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    favorite INTEGER NOT NULL DEFAULT 0 CHECK (favorite IN (0, 1)),
    started_on TEXT,
    completed_on TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (progress_total IS NULL OR progress_current <= progress_total)
);

CREATE INDEX IF NOT EXISTS idx_entertainment_items_type ON entertainment_items(media_type);
CREATE INDEX IF NOT EXISTS idx_entertainment_items_status ON entertainment_items(status);
CREATE INDEX IF NOT EXISTS idx_entertainment_items_rating ON entertainment_items(rating);
CREATE INDEX IF NOT EXISTS idx_entertainment_items_updated ON entertainment_items(updated_at);

CREATE TABLE IF NOT EXISTS entertainment_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS entertainment_item_tags (
    item_id INTEGER NOT NULL REFERENCES entertainment_items(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES entertainment_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (item_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_entertainment_item_tags_tag_id ON entertainment_item_tags(tag_id);
