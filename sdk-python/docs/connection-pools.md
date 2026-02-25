*[Русская версия](connection-pools.ru.md)*

# Connection Pools

This guide covers how to integrate dephealth with your application's
existing connection pools for more realistic health monitoring.

## Standalone vs Pool Mode

| | Standalone Mode | Pool Mode |
| --- | --- | --- |
| **Connection** | Creates a new connection per check | Borrows from existing pool |
| **Setup** | Just provide URL | Pass pool/client object |
| **Overhead** | Connection establishment each time | Minimal (reuses pool) |
| **What it tests** | Network reachability + auth | Actual pool health + query |
| **Pool issues** | Not detected | Detected (exhaustion, leaks) |
| **Recommended for** | Simple setups, dev/test | Production |

## PostgreSQL (asyncpg)

```python
import asyncpg
from dephealth.api import DependencyHealth, postgres_check

# Create the pool as your application normally would
pg_pool = await asyncpg.create_pool(
    "postgresql://user:pass@pg.svc:5432/mydb",
    min_size=5,
    max_size=20,
)

dh = DependencyHealth("my-service", "my-team",
    postgres_check("postgres-main",
        pool=pg_pool,
        critical=True,
    ),
)
```

When `pool` is provided, the checker borrows a connection from the pool,
executes `SELECT 1` (or custom query), and returns the connection.

## MySQL (aiomysql)

```python
import aiomysql
from dephealth.api import DependencyHealth, mysql_check

mysql_pool = await aiomysql.create_pool(
    host="mysql.svc",
    port=3306,
    user="user",
    password="pass",
    db="mydb",
    minsize=5,
    maxsize=20,
)

dh = DependencyHealth("my-service", "my-team",
    mysql_check("mysql-main",
        pool=mysql_pool,
        critical=True,
    ),
)
```

## Redis (redis-py)

```python
from redis.asyncio import Redis
from dephealth.api import DependencyHealth, redis_check

redis_client = Redis.from_url(
    "redis://redis.svc:6379",
    max_connections=20,
)

dh = DependencyHealth("my-service", "my-team",
    redis_check("redis-cache",
        client=redis_client,
        critical=False,
    ),
)
```

When `client` is provided, the checker calls `PING` on the existing client
instead of creating a new connection.

## LDAP (ldap3)

```python
from ldap3 import Server, Connection
from dephealth.api import DependencyHealth, ldap_check

ldap_server = Server("ldap://ldap.svc:389")
ldap_conn = Connection(ldap_server, auto_bind=True)

dh = DependencyHealth("my-service", "my-team",
    ldap_check("ldap-server",
        client=ldap_conn,
        check_method="ROOT_DSE",
        critical=False,
    ),
)
```

## Mixed Modes

You can use different modes for different dependencies:

```python
import asyncpg
from redis.asyncio import Redis
from dephealth.api import (
    DependencyHealth,
    postgres_check,
    redis_check,
    http_check,
    kafka_check,
)

pg_pool = await asyncpg.create_pool("postgresql://user:pass@pg.svc:5432/mydb")
redis_client = Redis.from_url("redis://redis.svc:6379")

dh = DependencyHealth("my-service", "my-team",
    # Pool mode -- tests actual pool health
    postgres_check("postgres", pool=pg_pool, critical=True),
    redis_check("redis", client=redis_client, critical=False),

    # Standalone mode -- tests network reachability
    http_check("payment-api",
        url="http://payment.svc:8080",
        critical=True,
    ),
    kafka_check("kafka",
        url="kafka://kafka-1.svc:9092,kafka-2.svc:9092",
        critical=True,
    ),
)
```

## When to Use Which Mode

### Use Pool Mode When

- You want to detect pool exhaustion and connection leaks
- You want to verify the application's actual ability to query the dependency
- You are running in production
- The dependency driver supports async pools

### Use Standalone Mode When

- You don't have a connection pool (e.g., external HTTP/gRPC services)
- You want to test basic network reachability
- You are in development or testing
- The dependency type doesn't support pools (HTTP, gRPC, TCP, AMQP, Kafka)

## Supported Pool Types

| Checker | Pool Parameter | Pool Type |
| --- | --- | --- |
| PostgreSQL | `pool=` | `asyncpg.Pool` |
| MySQL | `pool=` | `aiomysql.Pool` |
| Redis | `client=` | `redis.asyncio.Redis` |
| LDAP | `client=` | `ldap3.Connection` |

HTTP, gRPC, TCP, AMQP, and Kafka checkers always use standalone mode.

## FastAPI Integration with Pools

```python
import os
import asyncpg
from redis.asyncio import Redis
from contextlib import asynccontextmanager
from fastapi import FastAPI

from dephealth.api import DependencyHealth, postgres_check, redis_check
from dephealth_fastapi import DepHealthMiddleware, dependencies_router

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Create pools
    pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
    redis_client = Redis.from_url(os.environ["REDIS_URL"])

    # Create and start dephealth
    dh = DependencyHealth("my-service", "my-team",
        postgres_check("postgres", pool=pg_pool, critical=True),
        redis_check("redis", client=redis_client, critical=False),
    )
    await dh.start()
    app.state.dephealth = dh
    app.state.pg_pool = pg_pool
    app.state.redis = redis_client

    yield

    # Cleanup
    await dh.stop()
    redis_client.close()
    await pg_pool.close()

app = FastAPI(lifespan=lifespan)
app.add_middleware(DepHealthMiddleware)
app.include_router(dependencies_router)
```

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Checkers](checkers.md) — all 9 built-in checkers
- [Configuration](configuration.md) — all options and defaults
- [FastAPI Integration](fastapi.md) — lifespan and middleware
- [Troubleshooting](troubleshooting.md) — common issues and solutions
