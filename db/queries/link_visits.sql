-- name: CreateLinkVisit :one
INSERT INTO link_visits (link_id, ip, user_agent, referer, status)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListLinkVisitsPaged :many
SELECT * FROM link_visits ORDER BY id LIMIT $1 OFFSET $2;

-- name: CountLinkVisits :one
SELECT count(*) FROM link_visits;
