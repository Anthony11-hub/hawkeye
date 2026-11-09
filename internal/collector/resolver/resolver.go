package resolver

import (
	"sort"
	"strings"

	"github.com/Anthony11-hub/hawkeye/internal/collector"
)

// minSiteUID is where regular (non-system) Linux users start.
// Cloudpanel creates one such user per site, so every process owned by a user at or above
// this UID is treated as part of that user's app
const minSiteUID = 1000

// Kind says how a row was grouped
type Kind string

const (
	KindSIte   Kind = "site"   // grouped by owner: one Cloudpanel site =  one user
	KindSystem Kind = "system" // grouped by process name: postgres, redis
)

// App is one row in ranking
type App struct {
	Name      string
	Kind      Kind
	Procs     int    // how many os processes belong to it
	RSS       uint64 // summed accross all of them
	DaemonRSS uint64 // part of RSS that is the site's own pm2 God Daemon (overhead)
}

// isPM2Daemon reports whether a process is a pm2 God Daemon. pm2 renames its
// process title to something like "PM2 v5.4.2: God Daemon (/home/site/.pm2)",
// which shows up in the command line. Requiring the "PM2 " prefix avoids
// false matches like `runuser ... "God Daemon"` or `grep "God Daemon"`.
func isPM2Daemon(p collector.ProcSample) bool {
	return strings.HasPrefix(p.Cmdline, "PM2") && strings.Contains(p.Cmdline, "God Daemon")
}

// Group assigns every process to a row and returns the rows ranked by RSS.
//
//   - Owned by a site user (UID >= minSiteUID): grouped by username.
//   - Anything else (root, postgres, redis, www-data...): grouped by process
//     name, so all postgres processes become one "postgres" row.
func Group(procs []collector.ProcSample) []App {
	// Two maps, because a username and a process name could collide.
	sites := make(map[string]*App)
	system := make(map[string]*App)

	for _, p := range procs {
		var a *App
		if p.UID >= minSiteUID {
			a = lookup(sites, p.User, KindSIte)
		} else {
			a = lookup(system, p.Name, KindSystem)
		}
		a.Procs++
		a.RSS += p.RSS
		if isPM2Daemon(p) {
			a.DaemonRSS += p.RSS
		}
	}

	result := make([]App, 0, len(sites)+len(system))
	for _, a := range sites {
		result = append(result, *a)
	}
	for _, a := range system {
		result = append(result, *a)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].RSS > result[j].RSS })
	return result
}

// lookup returns the entry for name, creating it on first use.
func lookup(m map[string]*App, name string, kind Kind) *App {
	a, ok := m[name]
	if !ok {
		a = &App{Name: name, Kind: kind}
		m[name] = a
	}
	return a
}
