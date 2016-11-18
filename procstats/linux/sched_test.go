package linux

import (
	"reflect"
	"testing"
)

func TestParseProcSched(t *testing.T) {
	text := `cat (5013, #threads: 1)
-------------------------------------------------------------------
se.exec_start                                :      44212723.563800
se.vruntime                                  :             0.553868
se.sum_exec_runtime                          :             8.945468
nr_switches                                  :                   51
nr_voluntary_switches                        :                   45
nr_involuntary_switches                      :                    6
se.load.weight                               :                 1024
se.avg.load_sum                              :              2139396
se.avg.util_sum                              :              2097665
se.avg.load_avg                              :                   40
se.avg.util_avg                              :                   39
se.avg.last_update_time                      :       44212723563800
policy                                       :                    0
prio                                         :                  120
clock-delta                                  :                   41
`

	proc, err := ParseProcSched(text)

	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(proc, ProcSched{
		NRSwitches:            51,
		NRVoluntarySwitches:   45,
		NRInvoluntarySwitches: 6,
		SEAvgLoadSum:          2139396,
		SEAvgUtilSum:          2097665,
		SEAvgLoadAvg:          40,
		SEAvgUtilAvg:          39,
	}) {
		t.Error(proc)
	}

}
