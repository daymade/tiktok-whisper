CREATE TABLE transcriptions
(
    id                   SERIAL PRIMARY KEY,
    input_dir            VARCHAR   NOT NULL,
    file_name            VARCHAR   NOT NULL,
    mp3_file_name        VARCHAR   NOT NULL,
    audio_duration       INTEGER   NOT NULL,
    transcription        VARCHAR   NOT NULL,
    last_conversion_time TIMESTAMP NOT NULL,
    has_error            INTEGER   NOT NULL,
    error_message        VARCHAR,
    user_nickname        VARCHAR
);
