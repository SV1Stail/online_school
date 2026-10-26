CREATE TABLE student_conference_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (provider IN ('teams', 'zoom', 'telemost', 'google_meet', 'sber_jazz', 'custom')),
    title text NOT NULL,
    url text NOT NULL DEFAULT '',
    active boolean NOT NULL DEFAULT false,
    display_order integer NOT NULL DEFAULT 0 CHECK (display_order >= 0),
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT student_conference_link_title_length CHECK (char_length(title) BETWEEN 1 AND 100),
    CONSTRAINT student_conference_link_active_url CHECK (NOT active OR char_length(url) > 0)
);

CREATE UNIQUE INDEX student_provider_link_unique
    ON student_conference_links (student_id, provider)
    WHERE provider <> 'custom' AND deleted_at IS NULL;

CREATE UNIQUE INDEX student_conference_link_url_unique
    ON student_conference_links (url)
    WHERE url <> '' AND deleted_at IS NULL;

CREATE INDEX student_conference_links_active_idx
    ON student_conference_links (student_id, active, display_order)
    WHERE deleted_at IS NULL;

CREATE TABLE student_conference_link_versions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    link_id uuid NOT NULL REFERENCES student_conference_links(id) ON DELETE CASCADE,
    title text NOT NULL,
    url text NOT NULL,
    active boolean NOT NULL,
    display_order integer NOT NULL,
    valid_from timestamptz NOT NULL,
    valid_to timestamptz,
    CONSTRAINT student_conference_link_version_time_order CHECK (valid_to IS NULL OR valid_to > valid_from),
    CONSTRAINT student_conference_link_versions_no_overlap EXCLUDE USING gist (
        link_id WITH =,
        tstzrange(valid_from, valid_to, '[)') WITH &&
    )
);

CREATE INDEX student_conference_link_versions_lookup_idx
    ON student_conference_link_versions (link_id, valid_from, valid_to);

WITH inserted AS (
    INSERT INTO student_conference_links (student_id, provider, title, display_order)
    SELECT student.id, defaults.provider, defaults.title, defaults.display_order
    FROM users student
    CROSS JOIN (VALUES
        ('teams', 'Microsoft Teams', 10),
        ('zoom', 'Zoom', 20),
        ('telemost', 'Яндекс Телемост', 30),
        ('google_meet', 'Google Meet', 40),
        ('sber_jazz', 'СберJazz', 50)
    ) AS defaults(provider, title, display_order)
    WHERE student.role = 'student'
    RETURNING id, title, url, active, display_order, created_at
)
INSERT INTO student_conference_link_versions (link_id, title, url, active, display_order, valid_from)
SELECT id, title, url, active, display_order, created_at
FROM inserted;
