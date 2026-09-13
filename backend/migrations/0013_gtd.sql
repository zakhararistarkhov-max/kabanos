-- +goose Up
-- GTD ("Getting Things Done") workspace: a capture inbox that is clarified into
-- buckets and (optionally) grouped under projects. One row per atomic thought or
-- action; the `bucket` column is what the Clarify/Organize steps move items
-- between. Calendar items carry a concrete time so a later CalDAV sync can map
-- them to Yandex.Calendar events one-to-one.

CREATE TABLE gtd_projects (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    outcome      TEXT NOT NULL DEFAULT '',              -- desired successful result
    notes        TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'active'
                 CHECK (status IN ('active','someday','done','dropped')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);
CREATE INDEX gtd_projects_user_idx ON gtd_projects (user_id);

CREATE TABLE gtd_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id    UUID REFERENCES gtd_projects(id) ON DELETE SET NULL,
    title         TEXT NOT NULL,
    notes         TEXT NOT NULL DEFAULT '',
    -- Where the item lives after clarification:
    --   inbox     — captured, not yet processed
    --   next      — a concrete next action (has a context)
    --   waiting   — delegated / awaiting someone (waiting_for)
    --   calendar  — time-specific, must happen at scheduled_at
    --   someday   — someday/maybe
    --   reference — non-actionable info to keep
    bucket        TEXT NOT NULL DEFAULT 'inbox'
                  CHECK (bucket IN ('inbox','next','waiting','calendar','someday','reference')),
    context       TEXT NOT NULL DEFAULT '',             -- e.g. @calls, @computer, @home
    waiting_for   TEXT NOT NULL DEFAULT '',             -- who/what we're waiting on
    scheduled_at  TIMESTAMPTZ,                          -- calendar items: start
    end_at        TIMESTAMPTZ,                          -- calendar items: optional end
    all_day       BOOLEAN NOT NULL DEFAULT false,
    due_on        DATE,                                 -- soft due date for next/waiting
    energy        TEXT NOT NULL DEFAULT ''
                  CHECK (energy IN ('','low','medium','high')),
    time_minutes  INT CHECK (time_minutes IS NULL OR time_minutes > 0),  -- estimated effort
    priority      INT NOT NULL DEFAULT 0 CHECK (priority BETWEEN 0 AND 3),
    done          BOOLEAN NOT NULL DEFAULT false,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX gtd_items_user_bucket_idx ON gtd_items (user_id, bucket) WHERE NOT done;
CREATE INDEX gtd_items_user_done_idx   ON gtd_items (user_id, done);
CREATE INDEX gtd_items_project_idx     ON gtd_items (project_id);
CREATE INDEX gtd_items_scheduled_idx   ON gtd_items (user_id, scheduled_at) WHERE bucket = 'calendar' AND NOT done;

-- +goose Down
DROP TABLE gtd_items;
DROP TABLE gtd_projects;
