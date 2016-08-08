package procstats

/*
#include <libproc.h>
#include <unistd.h>
#include <mach/mach.h>

static int procstats_thread_count() {
  thread_array_t thread_list;
  mach_msg_type_number_t thread_count;
  task_t task;

  if (task_for_pid(mach_task_self(), getpid(), &task) != KERN_SUCCESS) {
    return 0;
  }

  if (task_threads(task, &thread_list, &thread_count) != KERN_SUCCESS) {
    return 0;
  }

  vm_deallocate (mach_task_self(), (vm_address_t) thread_list, thread_count * sizeof(thread_act_t));
  return thread_count;
}

static int procstats_fd_count() {
  long size = proc_pidinfo(getpid(), PROC_PIDLISTFDS, 0, 0, 0);

  if (size < 0) {
    return -1;
  }

  return size / sizeof(struct proc_fdinfo);
}
*/
import "C"
import "syscall"

func threadCount() uint64 { return uint64(C.procstats_thread_count()) }

func fdCount() uint64 { return uint64(C.procstats_fd_count()) }

func fdMax() uint64 {
	rlimit := &syscall.Rlimit{}
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, rlimit)
	return rlimit.Cur
}
