-- name: CreateLink :one
INSERT INTO links (original_url, short_name) VALUES ($1, $2) RETURNING *;

-- name: GetLink :one
SELECT * FROM links WHERE id = $1;

-- name: ListLinks :many
SELECT * FROM links ORDER BY id;

-- name: UpdateLink :one
UPDATE links SET original_url = $2, short_name = $3 WHERE id = $1 RETURNING *;

-- name: DeleteLink :execrows
DELETE FROM links WHERE id = $1;
