package serve

import "syscall"

func syscallUmaskImpl(mask int) int { return syscall.Umask(mask) }
