*[Русская версия](noise-reduction.ru.md)*

# Noise Reduction

> This document explains how the dephealth alerting stack minimizes alert noise
> and helps operators quickly identify the root cause of a problem.
> For individual rule details, see [Alert Rules](alert-rules.md).
> For Alertmanager configuration, see [Alertmanager](alertmanager.md).

---

## The Problem: Alert Fatigue

When a dependency fails, a naive alerting setup fires **multiple alerts simultaneously**:

- "Dependency is down" (complete failure)
- "Dependency is degraded" (partial failure)
- "Dependency has high latency" (performance drop)
- "Dependency is flapping" (unstable)

Receiving 4 alerts for a single incident is counterproductive:

- **Information overload**: the operator spends time correlating alerts instead of fixing the problem.
- **Alert fatigue**: after seeing too many alerts, operators start ignoring them.
- **Unclear root cause**: which alert is the cause and which is the symptom?

The dephealth alerting stack solves this with four mechanisms: **inhibit rules**, **`for` duration tuning**, **alert grouping**, and **severity-based routing**.

---

## Real-World Scenarios

Each scenario shows what happens when a specific problem occurs — which alerts fire, which are suppressed, and what the operator should do.

<a id="scenario-1-database-master-down"></a>

### Scenario 1: Database Master Down

**Situation**: PostgreSQL master becomes unreachable. The service `order-api` has a single dependency `user-db` of type `postgres` with one endpoint.

**Metric state**:

```text
app_dependency_health{job="order-api", dependency="user-db", type="postgres", host="pg-master", port="5432"} = 0
```

**What happens without noise reduction** (4 alerts):

| Alert | Why It Would Fire |
| --- | --- |
| DependencyDown | `min(...) == 0` — all endpoints down |
| DependencyHighLatency | Connection timeout causes P99 spike (timeout = latency) |
| DependencyFlapping | If the check retries cause brief 0→1→0 transitions |
| DependencyAbsent | Does NOT fire — metrics still exist, value is just 0 |

**What happens with noise reduction** (1 alert):

| Alert | Status | Reason |
| --- | --- | --- |
| **DependencyDown** | **FIRES** | `min == 0`, `for: 1m` elapsed |
| DependencyHighLatency | **suppressed** | Inhibit: DependencyDown (critical) → all warning, equal: `[namespace, dependency]` |
| DependencyFlapping | **suppressed** | Inhibit: DependencyDown → DependencyFlapping, equal: `[job, namespace, dependency]` |

**Result**: The operator receives **1 alert** — `DependencyDown (critical)` for `user-db (postgres)` in `order-api`. The cause is immediately clear: PostgreSQL master is down. No need to correlate multiple alerts.

**Operator action**: Check PostgreSQL master — is the pod running? Is the PVC healthy? Is there a network partition?

---

<a id="scenario-2-one-replica-out-of-three-down"></a>

### Scenario 2: One Redis Replica Out of Three Down

**Situation**: The service `session-api` depends on a Redis cluster with 3 endpoints. One replica becomes unavailable.

**Metric state**:

```text
app_dependency_health{..., dependency="cache", host="redis-0", ...} = 1  (healthy)
app_dependency_health{..., dependency="cache", host="redis-1", ...} = 0  (down)
app_dependency_health{..., dependency="cache", host="redis-2", ...} = 1  (healthy)
```

**What happens with noise reduction** (1 alert):

| Alert | Status | Reason |
| --- | --- | --- |
| DependencyDown | does not fire | `min == 0` but there are healthy endpoints, so the expression evaluates per-dependency: `min(0, 1, 1) = 0`... |

Wait — let's re-examine. The `min by (job, namespace, dependency, type)` takes the minimum across all three endpoints. Since one is 0, `min = 0`, so DependencyDown **would fire**. But DependencyDegraded also fires because there's a mix of 0s and 1s.

Actually, this is where the **inhibit rules prevent the wrong alert from being noisy**. Both rules trigger on the PromQL level, but Alertmanager's inhibit rules ensure the operator sees the right one:

| Alert | PromQL | Alertmanager | Final |
| --- | --- | --- | --- |
| **DependencyDown** | fires (`min == 0`) | **FIRES** (critical, not inhibited) | **operator sees this** |
| **DependencyDegraded** | fires (mix of 0 and 1) | **suppressed** (inhibited by DependencyDown, equal: `[job, namespace, dependency]`) | suppressed |
| DependencyFlapping | may fire if transitions occur | **suppressed** (inhibited by DependencyDown) | suppressed |

**Important nuance**: with a single dependency that has multiple endpoints, `min == 0` when **any** endpoint is down. This means DependencyDown fires even for partial failure. This is by design — the `min` aggregation treats a dependency as a whole. If you need to distinguish "1 of 3 down" from "all 3 down", see [Custom Rules](custom-rules.md) for per-endpoint alerting.

**Result**: **1 alert** — `DependencyDown (critical)`. The annotations say "all endpoints of dependency `cache`... are unavailable," but in practice the operator investigates and finds only 1 of 3 is down. The description is conservative (assumes worst case).

**Alternative approach**: If distinguishing partial from complete failure is important, add a custom rule that uses `count` instead of `min`:

```yaml
# Fires only when ALL endpoints are truly down
- alert: DependencyCompletelyDown
  expr: |
    count by (job, namespace, dependency, type) (
      app_dependency_health == 1
    ) == 0
```

See [Custom Rules](custom-rules.md) for more examples.

**Operator action**: Check Redis replica `redis-1` — pod status, network connectivity, memory usage.

---

<a id="scenario-3-http-service-responding-slowly"></a>

### Scenario 3: HTTP Service Responding Slowly

**Situation**: The `payment-api` dependency (type `http`) is reachable but responds slowly. Health checks pass (return 2xx) but take 2-3 seconds instead of the usual 50ms.

**Metric state**:

```text
app_dependency_health{..., dependency="payment-api", type="http"} = 1  (healthy — 2xx response)
app_dependency_latency_seconds P99 = 2.5s  (above 1s threshold)
```

**What happens with noise reduction** (1 alert):

| Alert | Status | Reason |
| --- | --- | --- |
| DependencyDown | does not fire | `health = 1` — dependency is available |
| DependencyDegraded | does not fire | No endpoints with `health = 0` |
| **DependencyHighLatency** | **FIRES** | `P99 > 1s` for more than 5 minutes |
| DependencyFlapping | does not fire | No state changes — consistently healthy |

**Result**: **1 alert** — `DependencyHighLatency (warning)` for `payment-api (http)`. No other alerts fire because the dependency is available — just slow.

**Why this is the right signal**: The operator knows the problem is **performance**, not **availability**. This narrows the investigation: check response times, look for upstream bottlenecks, examine resource usage on the payment service — not network connectivity or DNS.

**Operator action**: Investigate `payment-api` performance — CPU/memory, database queries, upstream dependencies. The dependency is alive, so the problem is capacity or load, not infrastructure.

---

<a id="scenario-4-unstable-network-flapping"></a>

### Scenario 4: Unstable Network (Flapping)

**Situation**: A Redis dependency intermittently becomes unreachable due to network instability. The health status alternates: 1→0→1→0→1→0 every 2-3 minutes.

**Metric state over 15 minutes**:

```text
app_dependency_health = 1, 0, 1, 0, 1, 0, 1  (7 changes in 15 minutes)
```

**What happens with noise reduction**:

| Alert | Status | Reason |
| --- | --- | --- |
| DependencyDown | depends on timing | If the check happens during a "down" window, `min == 0`. But with `for: 1m`, the dependency might recover before 1 minute passes → likely does NOT fire. |
| DependencyDegraded | does not apply | Single endpoint — no partial failure concept. |
| DependencyHighLatency | unlikely | Timeouts during "down" periods may spike P99, but the `for: 5m` filter usually prevents firing. |
| **DependencyFlapping** | **FIRES** | `changes(health[15m]) = 7 > 4` — fires immediately (`for: 0m`). |

**Result**: **1 alert** — `DependencyFlapping (info)`. The `$value` in the description shows "7 times" — the operator knows this is instability, not a hard failure.

**Why `info` severity matters here**: Flapping alerts go to the null receiver by default (Alertmanager UI only, no push notifications). This is intentional:

- Flapping is a **symptom**, not a root cause. The operator needs to investigate *why* the dependency is unstable.
- Sending push notifications for flapping would add noise — the operator already knows something is wrong if DependencyDown fires during a "down" window.

**Operator action**: Investigate network stability — check for packet loss, DNS resolution issues, firewall rules, or resource exhaustion on the Redis node.

---

<a id="scenario-5-service-restart-or-deploy"></a>

### Scenario 5: Service Restart or Deploy

**Situation**: A rolling deployment replaces all pods of the `catalog-api` service. During the rollout, there's a 3-minute window where no pod exports dephealth metrics.

**Metric state**:

```text
# Before deploy: metrics present
app_dependency_health{job="catalog-api", ...} = 1

# During deploy (3 minutes): no metrics at all
# (old pod terminated, new pod not ready yet)

# After deploy: metrics return
app_dependency_health{job="catalog-api", ...} = 1
```

**What happens with noise reduction**:

| Alert | Status | Reason |
| --- | --- | --- |
| DependencyDown | does not fire | No data → no `min == 0` evaluation. `absent` ≠ `== 0`. |
| DependencyDegraded | does not fire | No data to evaluate. |
| DependencyHighLatency | does not fire | No data to evaluate. |
| DependencyFlapping | does not fire | No data to evaluate. |
| DependencyAbsent | depends on duration | `for: 5m` — if the deploy takes less than 5 minutes, the alert does NOT fire. If longer than 5 minutes → **FIRES**. |

**Result for deploy < 5 minutes**: **0 alerts**. The `for: 5m` filter absorbs the gap. This is the expected behavior for normal deployments.

**Result for deploy > 5 minutes**: **1 alert** — `DependencyAbsent (warning)`. This signals a potential deployment problem — the rollout is taking too long, or the new pods aren't starting.

**Why this works**: The 5-minute threshold is tuned for typical Kubernetes rolling updates. Most deployments complete within 2-3 minutes. If metrics are absent for longer, something is wrong.

**Operator action**: Check the deployment status — `kubectl rollout status`, pod events, image pull status.

---

<a id="inhibit-rules"></a>

## Mechanism 1: Inhibit Rules

Inhibit rules are Alertmanager's way of **suppressing less important alerts when a more severe alert is already firing** for the same dependency.

### Suppression Hierarchy

```text
DependencyDown (critical)
  ├── suppresses → DependencyDegraded    (equal: job, namespace, dependency)
  ├── suppresses → DependencyHighLatency (equal: job, namespace, dependency)
  ├── suppresses → DependencyFlapping    (equal: job, namespace, dependency)
  └── suppresses → all warning severity  (equal: namespace, dependency)

DependencyDegraded (warning)
  └── suppresses → DependencyFlapping    (equal: job, namespace, dependency)
```

### How `equal` Labels Work

The `equal` field specifies which labels must match for suppression to apply. For example:

```yaml
inhibit_rules:
  - source_matchers:
      - alertname = "DependencyDown"
    target_matchers:
      - alertname = "DependencyDegraded"
    equal: ['job', 'namespace', 'dependency']
```

This means: "If DependencyDown fires for `job=order-api, namespace=production, dependency=user-db`, then suppress DependencyDegraded **only if** it has the same `job`, `namespace`, and `dependency` values."

**Why this matters**: If `order-api` has a DependencyDown for `user-db` and a DependencyDegraded for `cache`, only the `user-db` DependencyDegraded is suppressed. The `cache` alert still fires — it's a different dependency.

### The Cascade Rule

The last inhibit rule is broader:

```yaml
- source_matchers:
    - alertname = "DependencyDown"
    - severity = "critical"
  target_matchers:
    - severity = "warning"
  equal: ['namespace', 'dependency']
```

Note: `equal` uses only `namespace` and `dependency` (not `job`). This suppresses warning alerts **across all services** that depend on the same failing dependency. If `user-db` is down, all services that depend on `user-db` get their warning alerts suppressed — only the critical DependencyDown alerts remain.

---

## Mechanism 2: `for` Duration

The `for` field controls how long a condition must be true before an alert fires. This is the first line of defense against transient failures.

| Alert | `for` | Rationale |
| --- | --- | --- |
| DependencyDown | 1m | Critical failure — alert quickly, but filter 1-2 bad scrapes (scrape interval = 15s) |
| DependencyDegraded | 2m | Partial failure is often transient (rolling updates, pod rescheduling) |
| DependencyHighLatency | 5m | Latency fluctuates naturally (GC, pool warmup, load spikes) |
| DependencyFlapping | 0m | `changes()` already uses a 15m window — no additional delay needed |
| DependencyAbsent | 5m | Covers typical deployment gaps (rolling updates take 2-3 minutes) |

### What Happens Without `for`

If all rules had `for: 0m`:

- **DependencyDown**: Every transient network blip triggers a critical page. With 15s scrape interval, a single failed scrape fires the alert.
- **DependencyHighLatency**: A single GC pause or cold connection causes a P99 spike → false alert.
- **DependencyAbsent**: Every rolling deployment triggers the alert → alert fatigue.

### Tuning `for` for Your Environment

- **Faster detection** (shorter `for`): reduce values if your SLO requires faster response. Trade-off: more false positives.
- **Less noise** (longer `for`): increase values if you experience many transient failures. Trade-off: slower detection of real problems.

---

## Mechanism 3: Alert Grouping

Alertmanager groups related alerts into a single notification, reducing the number of messages the operator receives.

### Default Grouping Configuration

```yaml
route:
  group_by: ['alertname', 'namespace', 'job', 'dependency']
  group_wait: 30s
  group_interval: 5m
```

| Parameter | Value | Effect |
| --- | --- | --- |
| `group_by` | `[alertname, namespace, job, dependency]` | Alerts with the same combination of these labels are grouped together |
| `group_wait` | 30s | After the first alert in a group, wait 30s for more alerts to arrive before sending |
| `group_interval` | 5m | Minimum time between notifications for the same group |

**Why `group_by` does NOT include `host` or `port`**:

If `host` were in `group_by`, each endpoint would generate a separate notification. For a Redis cluster with 3 endpoints, 3 separate messages would be sent instead of 1.

By grouping on `dependency`, all endpoints of the same dependency are combined: "Dependency `cache` has 2 of 3 endpoints down" — one message.

### Critical Alerts: Faster Delivery

```yaml
routes:
  - matchers:
      - severity = "critical"
    group_wait: 10s
    repeat_interval: 1h
```

For critical alerts, `group_wait` is reduced to 10s (from 30s default) for faster delivery. `repeat_interval` is also shorter (1h instead of 4h) to keep the operator aware.

---

## Mechanism 4: Severity-Based Routing

Different severity levels are routed to different receivers:

| Severity | Receiver | Behavior |
| --- | --- | --- |
| `critical` | default (webhook/Telegram) | Push notification, `group_wait: 10s`, `repeat_interval: 1h` |
| `warning` | default (webhook/Telegram) | Push notification, `group_wait: 30s`, `repeat_interval: 4h` |
| `info` | null | Alertmanager UI only, no push notification |

### Why `info` Goes to Null

Info alerts (DependencyFlapping) are diagnostic signals. Sending push notifications for them would:

1. Increase noise — flapping often accompanies other alerts.
2. Not be actionable — "the dependency is unstable" requires investigation, not immediate action.
3. Be suppressed anyway — inhibit rules suppress flapping when DependencyDown or DependencyDegraded fires.

The info alerts are still visible in the Alertmanager UI for operators who actively investigate issues.

---

## Anti-Patterns: What NOT to Do

### 1. `for: 0m` on Critical Alerts

```yaml
# BAD: fires on every transient failure
- alert: DependencyDown
  expr: min by (...) (app_dependency_health) == 0
  for: 0m   # ← notification storm
```

With 15s scrape interval, a single network hiccup fires a critical page. The operator gets woken up at 3 AM for a problem that resolved itself 15 seconds later.

### 2. No Inhibit Rules

Without inhibit rules, a single PostgreSQL failure produces:

- DependencyDown (critical) — correct
- DependencyHighLatency (warning) — noise (timeout = high latency)
- DependencyFlapping (info) — noise (if connection retries cause transitions)

The operator receives 3 alerts and must correlate them manually. With 10 dependencies, this becomes unmanageable.

### 3. `group_by` Including Instance/Host/Port

```yaml
# BAD: each endpoint generates a separate notification
group_by: ['alertname', 'namespace', 'job', 'dependency', 'host', 'port']
```

For a dependency with 5 endpoints, this produces 5 separate messages instead of 1. The operator's Telegram floods with messages that all describe the same problem.

### 4. Too Short `repeat_interval`

```yaml
# BAD: repeats every 5 minutes for unresolved alerts
repeat_interval: 5m
```

If a dependency is down for 1 hour, the operator receives 12 repeated notifications. This is the fastest way to make operators mute the channel.

**Recommended**: `repeat_interval: 4h` for warning, `1h` for critical. The operator knows about the problem — repeating it every 5 minutes doesn't help.

### 5. Single Receiver for All Severities

```yaml
# BAD: info/warning/critical all go to the same Telegram channel
route:
  receiver: ops-telegram
  # no severity-based routes
```

Info alerts (flapping) flood the same channel as critical alerts (DependencyDown). The critical alerts get lost in the noise.

**Solution**: Route critical to a high-priority channel (with sound notifications), warning to a regular channel, and info to null or a low-priority channel.

---

## Summary: How It All Works Together

When a problem occurs, the four mechanisms work in sequence:

```text
1. PromQL expression evaluates to true
   │
2. `for` duration filter: transient problems are filtered out
   │
3. Alert sent to Alertmanager
   │
4. Inhibit rules: redundant alerts are suppressed
   │
5. Grouping: related alerts are combined
   │
6. Routing: directed to appropriate receiver by severity
   │
7. Operator receives: 1 clear, actionable notification
```

The goal: **one problem → one alert → one clear action**.

---

## What's Next

- [Alert Rules](alert-rules.md) — detailed description of each rule's PromQL
- [Alertmanager Configuration](alertmanager.md) — how to configure routing, receivers, and inhibit rules
- [Custom Rules](custom-rules.md) — writing your own rules with proper noise reduction
