# hawkeye

A small CLI that shows **which apps use how much RAM on a Linux VPS**.

Tools like `top` or a hosting panel show you per-process or whole-machine numbers. When you run 15+ Node apps, each under its own user with its own pm2, what you actually want to know is "which *app* is eating my memory?". hawkeye groups processes into apps and ranks them.

```
RAM  total 11.7 GiB | used 6.6 GiB | available 5.1 GiB | counted by hawkeye 8.9 GiB
pm2  16 daemons using 890.0 MiB in total (already included in the site rows)

APP          TYPE    RSS        %RAM  PROCS  PM2 DAEMON
mysqld       system  904.5 MiB  7.6%  1      -
shop         site    902.1 MiB  7.5%  9      45.4 MiB
dashboard    site    741.2 MiB  6.2%  9      49.4 MiB
pos          site    519.9 MiB  4.3%  9      64.2 MiB
portal       site    471.7 MiB  3.9%  9      51.6 MiB
nginx        system  453.3 MiB  3.8%  8      -
blog-api     site    334.3 MiB  2.8%  7      56.1 MiB
worker       site    259.3 MiB  2.2%  7      55.0 MiB
...
postgres     system  60.7 MiB   0.5%  6      -
```

## How apps are grouped

- **Sites:** every process owned by a regular user (UID >= 1000) is grouped under that username. This matches setups like CloudPanel, where each site runs as its own system user.
- **System services:** everything else (root, postgres, redis, nginx...) is grouped by process name, so all `postgres` processes become one row.
- **pm2 daemons:** each site's pm2 God Daemon is detected and shown in its own column, so you can see how much memory the process manager itself costs. It is already counted inside the site's RSS.

## Build and run

Requires Go.

```bash
go build -o hawkeye ./cmd/hawkeye
sudo ./hawkeye snapshot
```

Run it as root to see other users' processes. Without root it only sees your own and prints a warning.

| Command | What it does |
|---|---|
| `hawkeye snapshot` | Apps ranked by memory (default) |
| `hawkeye procs` | Top 20 raw processes by memory |
| `hawkeye daemon` | Streams whole-system CPU, RAM, disk and network every 2s |

## Deploy to a server

The build is a single static binary, so deploying is a file copy:

```bash
make deploy SERVER=user@host
ssh -t user@host hawkeye snapshot
```

See the `Makefile` for the exact build flags.

## Caveats

- **RSS double-counts shared memory.** Processes of the same app share pages (the Node binary, shared libraries), so summing RSS overstates real usage. The "counted by hawkeye" figure in the header will be higher than "used". Switching to PSS is on the roadmap.
- Linux only (reads `/proc`).

## Roadmap

- [x] Per-process collector
- [x] Group processes into apps by owner
- [x] pm2 daemon overhead
- [ ] CPU % per app
- [ ] History in SQLite (`daemon` stores samples, `history <app>` queries them)
- [ ] Memory leak detection and a `compare` command for framework comparisons
- [ ] Live `top` view
- [ ] Alerts (webhook / Telegram)
- [ ] JSON output and HTTP API for a dashboard
- [ ] PSS memory and cgroup/Docker support
