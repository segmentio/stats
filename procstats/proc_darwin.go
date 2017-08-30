package procstats

/*
#include <mach/mach_init.h>
#include <mach/mach_host.h>
#include <mach/mach_port.h>
#include <mach/mach_traps.h>
#include <mach/task_info.h>
#include <mach/task.h>
#include <sys/proc_info.h>
#include <sys/types.h>
#include <sys/sysctl.h>

// Defined in kern/proc_info.c
extern int proc_pidinfo(int pid, int flavor, uint64_t arg, user_addr_t buffer, uint32_t buffersize, register_t * retval);

// CGO fails to call a macro that just references a global variable...
#ifdef mach_task_self
#undef mach_task_self
static mach_port_t mach_task_self() { return mach_task_self_; }
#endif
*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

func collectProcInfo(pid int) (info ProcInfo, err error) {
	defer func() { err = convertPanicToError(recover()) }()

	if pid != os.Getpid() {
		panic(errors.New("on darwin systems only metrics of the current process can be collected"))
	}

	self := C.mach_port_name_t(C.mach_task_self())
	task := C.mach_port_name_t(0)
	checkKern(C.task_for_pid(self, C.int(pid), &task))

	rusage := syscall.Rusage{}
	check(syscall.Getrusage(syscall.RUSAGE_SELF, &rusage))

	nofile := syscall.Rlimit{}
	check(syscall.Getrlimit(syscall.RLIMIT_NOFILE, &nofile))

	info.CPU.User = time.Duration(rusage.Utime.Nano())
	info.CPU.Sys = time.Duration(rusage.Stime.Nano())

	_, resident, suspend := taskInfo(task)
	info.Memory.Available = memoryAvailable()
	info.Memory.Size = resident
	info.Memory.Resident = resident
	info.Memory.MajorPageFaults = uint64(rusage.Majflt)
	info.Memory.MinorPageFaults = uint64(rusage.Minflt)

	info.Files.Max = nofile.Cur
	info.Files.Open = fdCount(pid)

	info.Threads.Num = threadCount(task)
	info.Threads.InvoluntaryContextSwitches = suspend
	return
}

func memoryAvailable() uint64 {
	mib := [2]C.int{C.CTL_HW, C.HW_MEMSIZE}
	mem := C.int64_t(0)
	len := C.size_t(8) // sizeof(int64_t)

	_, err := C.sysctl(&mib[0], 2, unsafe.Pointer(&mem), &len, nil, 0)
	check(err)

	return uint64(mem)
}

func taskInfo(task C.mach_port_name_t) (virtual uint64, resident uint64, suspend uint64) {
	info := C.mach_task_basic_info_data_t{}
	count := C.mach_msg_type_number_t(C.MACH_TASK_BASIC_INFO_COUNT)

	checkKern(C.task_info(
		C.task_name_t(task),
		C.MACH_TASK_BASIC_INFO, (*C.integer_t)(unsafe.Pointer(&info)),
		&count,
	))

	return uint64(info.virtual_size), uint64(info.resident_size), uint64(info.suspend_count)
}

func fdCount(pid int) uint64 {
	return uint64(C.proc_pidinfo(C.int(pid), C.PROC_PIDLISTFDS, 0, 0, 0, nil)) / uint64(C.PROC_PIDLISTFD_SIZE)
}

func threadCount(task C.mach_port_name_t) uint64 {
	threads := make([]C.thread_act_t, 100)
	count := C.mach_msg_type_number_t(0)

	for {
		checkKern(C.task_threads(
			C.task_inspect_t(task),
			(*C.thread_act_array_t)(unsafe.Pointer(&threads[0])),
			&count,
		))

		if uint64(count) < uint64(len(threads)) {
			break
		}

		threads = make([]C.thread_act_t, 10*len(threads))
		count = 0
	}

	return uint64(count)
}

func checkKern(rc C.kern_return_t) {
	if rc != C.KERN_SUCCESS {
		panic(fmt.Errorf("mach trap kernel error code: %d", rc))
	}
}
