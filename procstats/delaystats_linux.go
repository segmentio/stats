package procstats

import (
	"github.com/mdlayher/taskstats"
)

func collectDelayInfo(pid int) (info DelayInfo, err error) {
	client, err := taskstats.New()
	check(err)

	stats, err := client.PID(pid)
	check(err)

	info.BlockIODelay = stats.BlockIODelay
	info.CPUDelay = stats.CPUDelay
	info.FreePagesDelay = stats.FreePagesDelay
	info.SwapInDelay = stats.SwapInDelay

	return info, nil
}
