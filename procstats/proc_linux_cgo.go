package procstats

// #include <unistd.h>
import "C"

func getclktck() uint64 { return uint64(C.sysconf(C._SC_CLK_TCK)) }
