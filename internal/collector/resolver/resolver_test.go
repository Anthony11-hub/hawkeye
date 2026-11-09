package resolver

import (
	"testing"

	"github.com/Anthony11-hub/hawkeye/internal/collector"
)

func TestGroupBySiteUserAndSystemName(t *testing.T) {
	procs := []collector.ProcSample{
		// site "shop": pm2 daemon + npm + node, all owned by user shop
		{PID: 100, Name: "PM2 v5.4.2: God", Cmdline: "PM2 v5.4.2: God Daemon (/home/shop/.pm2)", RSS: 60, UID: 1001, User: "shop"},
		{PID: 101, Name: "npm", RSS: 50, UID: 1001, User: "shop"},
		{PID: 102, Name: "node", RSS: 300, UID: 1001, User: "shop"},
		// site "blog"
		{PID: 200, Name: "node", RSS: 200, UID: 1002, User: "blog"},
		// system services, grouped by process name
		{PID: 300, Name: "postgres", RSS: 400, UID: 110, User: "postgres"},
		{PID: 301, Name: "postgres", RSS: 500, UID: 110, User: "postgres"},
		{PID: 302, Name: "redis-server", RSS: 30, UID: 111, User: "redis"},
		{PID: 303, Name: "nginx", RSS: 10, UID: 0, User: "root"},
		{PID: 304, Name: "nginx", RSS: 20, UID: 33, User: "www-data"},
	}

	got := Group(procs)

	want := []struct {
		name   string
		kind   Kind
		rss    uint64
		procs  int
		daemon uint64
	}{
		{"postgres", KindSystem, 900, 2, 0},
		{"shop", KindSIte, 410, 3, 60},
		{"blog", KindSIte, 200, 1, 0},
		{"redis-server", KindSystem, 30, 1, 0},
		{"nginx", KindSystem, 30, 2, 0},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Name != w.name || g.Kind != w.kind || g.RSS != w.rss || g.Procs != w.procs || g.DaemonRSS != w.daemon {
			t.Errorf("row %d: got %+v, want %+v", i, g, w)
		}
	}
}

func TestPM2DaemonDetectionIgnoresLookalikes(t *testing.T) {
	daemon := collector.ProcSample{Cmdline: "PM2 v5.4.2: God Daemon (/home/shop/.pm2)"}
	lookalike := collector.ProcSample{Cmdline: `runuser -u shop -- bash -c exec -a "PM2 v5: God Daemon"`}
	if !isPM2Daemon(daemon) {
		t.Error("real daemon not detected")
	}
	if isPM2Daemon(lookalike) {
		t.Error("lookalike command line wrongly detected as a daemon")
	}
}

func TestSiteAndSystemNamesDoNotCollide(t *testing.T) {
	procs := []collector.ProcSample{
		{PID: 1, Name: "redis", RSS: 10, UID: 1001, User: "redis"}, // a site user that happens to be called redis
		{PID: 2, Name: "redis", RSS: 20, UID: 111, User: "redis"},  // the real redis service
	}
	if got := Group(procs); len(got) != 2 {
		t.Fatalf("expected 2 separate rows, got %+v", got)
	}
}
