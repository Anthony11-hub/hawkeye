package collector

import (
	"fmt"
	osuser "os/user"
	"sort"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

// ProcSample is a snapshot of one OS process at one moment
type ProcSample struct {
	PID     int32
	PPID    int32 // parent PID
	Name    string
	Cmdline string
	RSS     uint64 // resident memory in bytes
	UID     uint32 // owner's user ID
	User    string // owner's username
}

type MemSummary struct {
	Total     uint64
	Used      uint64
	Available uint64 // what the kernel could hand out without swapping; the honest "free" number
}

func FetchMemory() (MemSummary, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return MemSummary{}, fmt.Errorf("mem: %w", err)
	}
	return MemSummary{Total: vm.Total, Used: vm.Used, Available: vm.Available}, nil
}

// FetchProcesses lists every process we are allowed to read
// It returns a slice ([]ProcSample), Go's growable list type
func FetchProcesses() ([]ProcSample, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("list processess: %w", err)
	}

	// make(type, length, capacity) preallocates so append never has to grow
	out := make([]ProcSample, 0, len(procs))

	// looking up a username reads /etc/passwd, so cache per UID
	usernames := make(map[uint32]string)

	for _, p := range procs {
		// A process can exit between being listed and being read, and some are
		// owned by other users. Both cases return an error, so we skip that
		// process rather than failing the whole snapshot.

		mi, err := p.MemoryInfo()
		if err != nil || mi.RSS == 0 {
			continue // RSS of 0 means a kernel thread, which has no user memory
		}
		ppid, _ := p.Ppid()
		name, _ := p.Name()
		cmd, _ := p.Cmdline()

		// Uids returns real, effective, saved and filesystem IDs; [0] is the real one
		var uid uint32
		if uids, err := p.Uids(); err == nil && len(uids) > 0 {
			uid = uids[0]
		}
		username, ok := usernames[uid]
		if !ok {
			if u, err := osuser.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil {
				username = u.Username
			} else {
				username = strconv.FormatUint(uint64(uid), 10) // no passwd entry: show the number
			}
			usernames[uid] = username
		}

		out = append(out, ProcSample{
			PID:     p.Pid,
			PPID:    ppid,
			Name:    name,
			Cmdline: strings.TrimSpace(cmd),
			RSS:     mi.RSS,
			UID:     uid,
			User:    username,
		})
	}
	return out, nil
}

// TopByRSS sorts in place, biggest memory first, and returns the first n
func TopByRSS(procs []ProcSample, n int) []ProcSample {
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].RSS > procs[j].RSS
	})
	if n > len(procs) {
		n = len(procs)
	}
	return procs[:n]
}
