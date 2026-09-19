-- +goose Up
-- Unify tasks: a graph task node is now just a placement of a gtd_item, so its
-- title/note/deadline come from the item (not the node). Backfill the item from
-- any data that lived on task nodes so nothing is lost, then those node columns
-- are simply ignored for task nodes (they stay for free-form "note" nodes).

UPDATE gtd_items i SET due_on = n.deadline
  FROM gtd_graph_nodes n
  WHERE n.item_id = i.id AND n.kind = 'task' AND n.deadline IS NOT NULL AND i.due_on IS NULL;

UPDATE gtd_items i SET notes = n.note
  FROM gtd_graph_nodes n
  WHERE n.item_id = i.id AND n.kind = 'task' AND n.note <> '' AND COALESCE(i.notes, '') = '';

UPDATE gtd_items i SET title = n.label
  FROM gtd_graph_nodes n
  WHERE n.item_id = i.id AND n.kind = 'task' AND n.label <> '' AND n.label <> i.title;

-- +goose Down
-- No-op: the backfill is not reversible (item fields may have been edited since).
SELECT 1;
