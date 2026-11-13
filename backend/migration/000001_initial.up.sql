CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    login text NOT NULL,
    password_hash text NOT NULL,
    full_name text NOT NULL,
    role text NOT NULL CHECK (role IN ('student', 'teacher', 'moderator')),
    timezone text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    must_change_password boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_login_normalized CHECK (login = lower(btrim(login))),
    CONSTRAINT users_login_length CHECK (char_length(login) BETWEEN 1 AND 100),
    CONSTRAINT users_full_name_length CHECK (char_length(full_name) BETWEEN 1 AND 200)
);

CREATE UNIQUE INDEX users_login_unique ON users (lower(login));
CREATE INDEX users_role_active_idx ON users (role, active);

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE availability_windows (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id uuid NOT NULL REFERENCES users(id),
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    allowed_durations smallint[] NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id),
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT availability_time_order CHECK (ends_at > starts_at),
    CONSTRAINT availability_max_length CHECK (ends_at <= starts_at + interval '24 hours'),
    CONSTRAINT availability_durations_not_empty CHECK (cardinality(allowed_durations) BETWEEN 1 AND 4),
    CONSTRAINT availability_durations_allowed CHECK (
        allowed_durations <@ ARRAY[15, 30, 45, 60]::smallint[]
    ),
    CONSTRAINT availability_no_overlap EXCLUDE USING gist (
        teacher_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    ) WHERE (deleted_at IS NULL)
);

CREATE INDEX availability_teacher_start_idx ON availability_windows (teacher_id, starts_at)
    WHERE deleted_at IS NULL;

CREATE TABLE bookings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    availability_id uuid NOT NULL REFERENCES availability_windows(id),
    teacher_id uuid NOT NULL REFERENCES users(id),
    student_id uuid NOT NULL REFERENCES users(id),
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    duration_minutes smallint NOT NULL CHECK (duration_minutes IN (15, 30, 45, 60)),
    status text NOT NULL CHECK (
        status IN ('pending', 'confirmed', 'rejected', 'cancelled_by_student', 'cancelled_by_staff')
    ),
    created_by uuid NOT NULL REFERENCES users(id),
    idempotency_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT bookings_time_order CHECK (ends_at > starts_at),
    CONSTRAINT bookings_duration_matches CHECK (
        ends_at = starts_at + duration_minutes * interval '1 minute'
    ),
    CONSTRAINT bookings_idempotency_key_length CHECK (char_length(idempotency_key) BETWEEN 1 AND 100),
    CONSTRAINT bookings_creator_idempotency_unique UNIQUE (created_by, idempotency_key),
    CONSTRAINT bookings_teacher_no_overlap EXCLUDE USING gist (
        teacher_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    ) WHERE (status IN ('pending', 'confirmed')),
    CONSTRAINT bookings_student_no_overlap EXCLUDE USING gist (
        student_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    ) WHERE (status IN ('pending', 'confirmed'))
);

CREATE INDEX bookings_teacher_start_idx ON bookings (teacher_id, starts_at);
CREATE INDEX bookings_student_start_idx ON bookings (student_id, starts_at);
CREATE INDEX bookings_status_start_idx ON bookings (status, starts_at);

CREATE TABLE teacher_conference_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (provider IN ('teams', 'zoom', 'telemost', 'google_meet', 'sber_jazz', 'custom')),
    title text NOT NULL,
    url text NOT NULL DEFAULT '',
    active boolean NOT NULL DEFAULT false,
    display_order integer NOT NULL DEFAULT 0 CHECK (display_order >= 0),
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT conference_link_title_length CHECK (char_length(title) BETWEEN 1 AND 100),
    CONSTRAINT conference_link_active_url CHECK (NOT active OR char_length(url) > 0)
);

CREATE UNIQUE INDEX teacher_provider_link_unique
    ON teacher_conference_links (teacher_id, provider)
    WHERE provider <> 'custom' AND deleted_at IS NULL;
CREATE INDEX teacher_conference_links_active_idx
    ON teacher_conference_links (teacher_id, active, display_order)
    WHERE deleted_at IS NULL;

CREATE TABLE conference_link_versions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    link_id uuid NOT NULL REFERENCES teacher_conference_links(id) ON DELETE CASCADE,
    title text NOT NULL,
    url text NOT NULL,
    active boolean NOT NULL,
    display_order integer NOT NULL,
    valid_from timestamptz NOT NULL,
    valid_to timestamptz,
    CONSTRAINT conference_link_version_time_order CHECK (valid_to IS NULL OR valid_to > valid_from),
    CONSTRAINT conference_link_versions_no_overlap EXCLUDE USING gist (
        link_id WITH =,
        tstzrange(valid_from, valid_to, '[)') WITH &&
    )
);

CREATE INDEX conference_link_versions_lookup_idx
    ON conference_link_versions (link_id, valid_from, valid_to);

CREATE TABLE audit_log (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
    action text NOT NULL,
    entity_type text NOT NULL,
    entity_id text NOT NULL,
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_actor_created_idx ON audit_log (actor_id, created_at DESC);
CREATE INDEX audit_log_entity_idx ON audit_log (entity_type, entity_id, created_at DESC);
