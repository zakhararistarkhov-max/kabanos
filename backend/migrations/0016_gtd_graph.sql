-- +goose Up
-- Mind-map / graph view for GTD. Each "board" is the canvas of one project
-- (board_project_id), or the root board of all projects when board_project_id
-- IS NULL. A node references a task (gtd_items), a sub-project (gtd_projects,
-- enabling "a project of projects" — double-click drills into its own board),
-- or is a free-form note. Edges connect nodes on the same board.

CREATE TABLE gtd_graph_nodes (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    board_project_id UUID REFERENCES gtd_projects(id) ON DELETE CASCADE, -- NULL = root board
    kind             TEXT NOT NULL CHECK (kind IN ('task','project','note')),
    item_id          UUID REFERENCES gtd_items(id) ON DELETE CASCADE,
    ref_project_id   UUID REFERENCES gtd_projects(id) ON DELETE CASCADE,
    label            TEXT NOT NULL DEFAULT '',
    x                DOUBLE PRECISION NOT NULL DEFAULT 0,
    y                DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX gtd_graph_nodes_board_idx ON gtd_graph_nodes (user_id, board_project_id);

CREATE TABLE gtd_graph_edges (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    board_project_id UUID REFERENCES gtd_projects(id) ON DELETE CASCADE,
    source_id        UUID NOT NULL REFERENCES gtd_graph_nodes(id) ON DELETE CASCADE,
    target_id        UUID NOT NULL REFERENCES gtd_graph_nodes(id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_id, target_id)
);
CREATE INDEX gtd_graph_edges_board_idx ON gtd_graph_edges (user_id, board_project_id);

-- +goose Down
DROP TABLE gtd_graph_edges;
DROP TABLE gtd_graph_nodes;
