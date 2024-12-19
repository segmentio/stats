package procstats

import (
	"errors"

	"github.com/mdlayher/taskstats"
	"golang.org/x/sys/unix"
)

func collectDelayInfo(pid int) (DelayInfo, error) {
	client, err := taskstats.New()
	switch {
	case errors.Is(err, unix.ENOENT):
		return DelayInfo{}, errors.New("failed to communicate with taskstats Netlink family, ensure this program is not running in a network namespace")
	case err != nil:
		return DelayInfo{}, err
	default:
		defer client.Close()
	}

	stats, err := client.TGID(pid)
	switch {
	case errors.Is(err, unix.EPERM):
		return DelayInfo{}, errors.New("failed to open Netlink socket: permission denied, ensure CAP_NET_RAW is enabled for this process, or run it with root privileges")
	case err != nil:
		return DelayInfo{}, err
	default:
		return DelayInfo{
			BlockIODelay:   stats.BlockIODelay,
			CPUDelay:       stats.CPUDelay,
			FreePagesDelay: stats.FreePagesDelay,
			SwapInDelay:    stats.SwapInDelay,
		}, nil
	}
}
