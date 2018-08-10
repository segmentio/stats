package procstats

import (
	"syscall"

	"github.com/mdlayher/taskstats"
)

func collectDelayInfo(pid int) (info DelayInfo, err error) {
	client, err := taskstats.New()
	check(err)

	stats, err := client.PID(pid)
	if err != nil {
		if err1, ok := err.(syscall.Errno); ok && err1 == syscall.EPERM {
			panic("Failed to open Netlink socket: permission denied.  Ensure CAP_NET_RAW is enabled for this process, or run as root.")
		}
		check(err)
	}

	info.BlockIODelay = stats.BlockIODelay
	info.CPUDelay = stats.CPUDelay
	info.FreePagesDelay = stats.FreePagesDelay
	info.SwapInDelay = stats.SwapInDelay

	return info, nil
}
