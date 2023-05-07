CREATE TABLE IF NOT EXISTS transcriptions
(
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    user                 TEXT     NOT NULL,
    input_dir            TEXT     NOT NULL,
    file_name            TEXT     NOT NULL,
    mp3_file_name        TEXT     NOT NULL,
    audio_duration       INTEGER  NOT NULL,
    transcription        TEXT     NOT NULL,
    last_conversion_time DATETIME NOT NULL,
    has_error            INTEGER  NOT NULL,
    error_message        TEXT
);