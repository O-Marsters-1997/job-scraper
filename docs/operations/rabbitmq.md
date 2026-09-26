# RabbitMQ source queues: operations

## Cutover

1. Stop the old API and worker publishers. There is no real Valkey task data to import for this first deployment.
2. Set `RABBITMQ_USER` and `RABBITMQ_PASSWORD`, apply migration `20260924000000`, and start the pinned RabbitMQ 4.3.6 service with its named `rabbitmq_data` volume. Keep its `hostname: rabbitmq` stable so a recreated container uses the same node data directory.
3. Start the new API and worker together. Startup must declare one quorum `source.<name>` queue per registered Source, direct `source.work` bindings, and quorum `source.dead` bound to `source.dead.exchange`. A declaration mismatch fails startup.
4. Check `rabbitmq-diagnostics -q ping`, the management UI on localhost port 15672, and `just queue-list`. Verify each source queue has priority and delayed-retry arguments, `delivery-limit=5`, an at-least-once DLX, and `reject-publish` overflow.
5. Publish a synthetic known-Source detail through the API or a test client; confirm it reaches `/ingest` and is acknowledged. Force a failing detail and verify its reject retries then shared DLQ entry; inspect it with `just queue-dead inspect TASK_ID` and replay it after fixing the cause. Force a `queued` Source Target start without publishing, then verify the worker's minute reconciliation republishes it. The DB run ID must stay the same.
6. Monitor Source queue depth, oldest task age, and shared DLQ depth. A nonempty DLQ or a growing Source queue needs operator attention. The worker logs task ID, Source, kind, run identity, delivery count, and error.

The API and worker need separate AMQP connections. `RABBITMQ_URL` supplies application credentials; `cmd/queue` uses operator credentials with management API access. Compose binds AMQP and management ports to localhost. A single RabbitMQ container survives process and container restarts, but a host or volume loss requires restoring a backup.

## Back up and restore

Stop the API and worker, then stop RabbitMQ to capture a consistent copy of the quorum log. Mount the Compose volume into a one-off container and archive `/var/lib/rabbitmq` to storage outside the host. For example:

```sh
docker compose stop api worker rabbitmq
mkdir -p backups
docker compose run --no-deps --rm -v "$PWD/backups:/backup" --entrypoint sh rabbitmq -c 'tar -C /var/lib/rabbitmq -czf /backup/rabbitmq.tar.gz .'
```

Restore the archive into an **empty** `rabbitmq_data` volume with the same RabbitMQ version, hostname, and credentials, before starting RabbitMQ. Then start RabbitMQ, check its health and queue bindings, and start the API and worker. Restore the matching PostgreSQL backup as needed. A lost broker volume cannot be repaired by Source Target reconciliation alone: that process only recreates queued or stale starts and due Boards, not accepted detail tasks. Replay any separately exported tasks after restore.

## Replay and rollback

`./queue count` reports DLQ depth. `./queue list [limit]` and `./queue inspect TASK_ID` use the management API's requeue mode, leaving messages in place. `./queue replay TASK_ID` publishes the original typed task with a fresh task ID, checks mandatory routing and a publisher confirm, then acknowledges its DLQ delivery. A publish failure requeues the original DLQ delivery.

Keep the previous API and worker image/configuration until cutover checks pass. If rolling back after RabbitMQ accepted real tasks, stop publishers and workers, drain the broker or export and replay its outstanding tasks into the rollback system before switching binaries. Rolling back without handling those tasks abandons them even though they remain in the RabbitMQ volume.

## Alerts

"Dead letters" and "Queue not draining" (`ops/grafana/alerts/rules.yaml`, folder "Pipeline health")
fire on `source.dead` depth and on a stalled `source.<name>` queue. Inspect and act on either with
`just queue-list` and `just queue-dead inspect|replay` (see "Replay and rollback" above).
