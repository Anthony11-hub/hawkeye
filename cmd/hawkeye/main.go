package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"text/tabwriter"
	"time"

	"github.com/Anthony11-hub/hawkeye/internal/collector"
	"github.com/Anthony11-hub/hawkeye/internal/collector/resolver"
)

func main() {
	// os.Args[0] is the program name, real args start at index 1
	cmd := "snapshot"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "snapshot":
		runSnapshot()
	case "daemon":
		runDaemon()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\nusage: hawkeye [snapshot]|daemon]\n", cmd)
		os.Exit(2)
	}
}

// runSnapshot prints the machine memory line, then the top processes by RSS
func runSnapshot() {
	sys, err := collector.FetchMemory()
	if err != nil {
		log.Fatal(err)
	}
	procs, err := collector.FetchProcesses()
	if err != nil {
		log.Fatal(err)
	}

	// Processes owned by other users are only readable as root.
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "warning: not running as root, other users' processes may be missing")
	}

	apps := resolver.Group(procs)

	// Total memory, and how much of it is pm2 God Daemons (one per site).
	var total, daemons uint64
	var daemonCount int
	for _, a := range apps {
		total += a.RSS
		daemons += a.DaemonRSS
		if a.DaemonRSS > 0 {
			daemonCount++
		}
	}

	fmt.Printf("RAM  total %s | used %s | available %s | counted by hawkeye %s\n",
		formatBytes(sys.Total), formatBytes(sys.Used), formatBytes(sys.Available), formatBytes(total))
	fmt.Printf("pm2  %d daemons using %s in total (already included in the site rows)\n\n",
		daemonCount, formatBytes(daemons))

	// tabwriter lines columns up for us. Fields are tab-separated and Flush() does the padding
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "APP\tTYPE\tRSS\t%RAM\tPROCS\tPM2 DAEMON")
	for _, a := range apps {
		daemon := "-"
		if a.DaemonRSS > 0 {
			daemon = formatBytes(a.DaemonRSS)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%.1f%%\t%d\t%s\n",
			a.Name, a.Kind, formatBytes(a.RSS), float64(a.RSS)/float64(sys.Total)*100,
			a.Procs, daemon)
	}
	w.Flush()
}

// runProcs prints the raw top processes by RSS (the step 1 view)
func runProcs() {
	sys, err := collector.FetchMemory()
	if err != nil {
		log.Fatal(err)
	}
	procs, err := collector.FetchProcesses()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("RAM  total %s | used %s | available %s\n\n",
		formatBytes(sys.Total), formatBytes(sys.Used), formatBytes(sys.Available))

	// tabwriter lines columns up for us. Fields are tab-separated and
	// Flush() does the padding.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PID\tRSS\tNAME\tCOMMAND")
	for _, p := range collector.TopByRSS(procs, 20) {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.PID, formatBytes(p.RSS), p.Name, truncate(p.Cmdline, 60))
	}
	w.Flush()

}

// runDaemon streams whole system metrics until ctrl+c
// Has a slower tick and a shutdown-safe send
func runDaemon() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	metrics := make(chan collector.SystemMetrics)
	go runCollector(ctx, metrics, 2*time.Second)

	for m := range metrics {
		fmt.Printf("%+v\n", m)
	}
}

func runCollector(ctx context.Context, out chan<- collector.SystemMetrics, every time.Duration) {
	defer close(out)

	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m, err := collector.FetchMetrics()
			if err != nil {
				log.Println("collect:", err)
				continue
			}
			select {
			case out <- m:
			case <-ctx.Done():
				return
			}
		}
	}
}

// formatBytes turns 1536 into "1.5 KiB"
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "..."
}
