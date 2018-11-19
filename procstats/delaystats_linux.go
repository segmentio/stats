package procstats

import (
	"errors"
	"syscall"

	"github.com/segmentio/taskstats"
)

func collectDelayInfo(pid int) DelayInfo {
	client, err := taskstats.New()
	if err == syscall.ENOENT {
		err = errors.New("Failed to communicate with taskstats Netlink family.  Ensure this program is not running in a network namespace.")
	}
	check(err)

	stats, err := client.PID(pid)
	if err == syscall.EPERM {
		err = errors.New("Failed to open Netlink socket: permission denied.  Ensure CAP_NET_RAW is enabled for this process, or run it with root privileges.")
	}
	check(err)

	return DelayInfo{
		BlockIODelay:   stats.BlockIODelay,
		CPUDelay:       stats.CPUDelay,
		FreePagesDelay: stats.FreePagesDelay,
		SwapInDelay:    stats.SwapInDelay,
	}
}
