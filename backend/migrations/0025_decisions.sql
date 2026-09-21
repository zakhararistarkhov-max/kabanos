-- +goose Up
-- Decision journal: capture a decision with its context and expected outcome,
-- come back to it on a review date, then record what actually happened, rate how
-- right the decision was, and note lessons for the future.

CREATE TABLE decisions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    context     TEXT NOT NULL DEFAULT '',
    decision    TEXT NOT NULL DEFAULT '',
    expected    TEXT NOT NULL DEFAULT '',                 -- predicted outcome
    confidence  SMALLINT CHECK (confidence BETWEEN 1 AND 5), -- how sure at the time
    decided_on  DATE NOT NULL DEFAULT current_date,
    review_at   DATE,                                     -- when to revisit
    status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','reviewed')),
    result      TEXT NOT NULL DEFAULT '',                 -- what actually happened
    rating      SMALLINT CHECK (rating BETWEEN 1 AND 5),  -- how right it turned out
    lessons     TEXT NOT NULL DEFAULT '',                 -- notes for the future
    reviewed_at TIMESTAMPTZ,
    notified_at TIMESTAMPTZ,                              -- review-nudge push dedupe
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX decisions_user_idx ON decisions (user_id, status, review_at, created_at DESC);
-- Worker scan for due review nudges.
CREATE INDEX decisions_due_idx ON decisions (review_at) WHERE status='open' AND notified_at IS NULL;

-- +goose Down
DROP TABLE decisions;
