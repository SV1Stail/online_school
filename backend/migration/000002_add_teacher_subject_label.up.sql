ALTER TABLE users
    ADD COLUMN subject_label text NOT NULL DEFAULT '',
    ADD CONSTRAINT users_subject_label_length CHECK (char_length(subject_label) <= 100);
