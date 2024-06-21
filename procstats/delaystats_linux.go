package procstats

import (
	"errors"

	"github.com/mdlayher/taskstats"
	"golang.org/x/sys/unix"
)

func collectDelayInfo(pid int) DelayInfo {
	client, err := taskstats.New()
	if err == unix.ENOENT {
		err = errors.New("failed to communicate with taskstats Netlink family, ensure this program is not running in a network namespace")
	}
	check(err)

	stats, err := client.TGID(pid)
	if err == unix.EPERM {
		err = errors.New("failed to open Netlink socket: permission denied, ensure CAP_NET_RAW is enabled for this process, or run it with root privileges")
	}
	check(err)

	return DelayInfo{
		BlockIODelay:   stats.BlockIODelay,
		CPUDelay:       stats.CPUDelay,
		FreePagesDelay: stats.FreePagesDelay,
		SwapInDelay:    stats.SwapInDelay,
	}
}
