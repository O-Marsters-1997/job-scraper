-- name: UpsertPushSubscription :exec
INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(endpoint)::text, sqlc.arg(p256dh)::text, sqlc.arg(auth)::text)
ON CONFLICT (endpoint) DO UPDATE
SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth;

-- name: ListPushSubscriptions :many
SELECT endpoint, p256dh, auth FROM push_subscriptions
WHERE user_id = $1
ORDER BY created_at, id;

-- name: DeletePushSubscription :exec
DELETE FROM push_subscriptions WHERE endpoint = sqlc.arg(endpoint)::text AND user_id = sqlc.arg(user_id)::uuid;
