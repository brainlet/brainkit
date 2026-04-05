# Scheduling Test Map

**Purpose:** Verifies schedule creation, firing (recurring and one-time), cancellation, drain behavior, invalid expressions, and transport-level schedule delivery.
**Tests:** 10 functions across 2 files
**Entry point:** `scheduling_test.go` → `Run(t, env)`
**Campaigns:** transport (amqp, redis, postgres, nats, sqlite)

## Files

### fire.go — Schedule firing, cancellation, and edge cases

| Function | Purpose |
|----------|---------|
| testEveryFiresRepeatedly | Creates "every 200ms" schedule, subscribes to the topic, waits 700ms, verifies at least 3 fires |
| testInFiresOnce | Creates "in 100ms" one-time schedule, waits 500ms, verifies exactly 1 fire and schedule removed from ListSchedules |
| testUnschedule | Creates "every 100ms" schedule, waits for some fires, unschedules by ID, waits more, verifies fire count stopped increasing |
| testInvalidExpression | Attempts "cron 0 9 * * *" expression (unsupported), verifies error containing "unsupported schedule expression" |
| testTeardownCancelsSchedules | Deploys .ts that calls bus.schedule, tears down the deployment, verifies all schedules from that source are removed |
| testDrainSkipsFiring | Creates schedule, enables drain mode, waits, verifies no fires occur during drain |
| testE2EScheduleFires | Deploys a handler .ts, creates a "in 200ms" schedule targeting the handler's mailbox topic, verifies the handler receives the scheduled message |
| testInputAbuseScheduleInvalidExpression | Attempts "bananas at midnight" expression, verifies error returned |
| testInputAbuseScheduleEmptyTopic | Creates schedule with empty topic string, verifies either succeeds or errors cleanly (no panic) |

### backend_advanced.go — Transport-level schedule tests

| Function | Purpose |
|----------|---------|
| testScheduleFireOnTransport | Creates "in 200ms" schedule, subscribes to the topic, verifies the payload arrives containing the expected data |

## Cross-references

- **Campaigns:** transport/{amqp,redis,postgres,nats,sqlite}_test.go
- **Related domains:** persistence (schedule persistence across restart), stress (schedule storm)
- **Fixtures:** scheduling-related TS fixtures
