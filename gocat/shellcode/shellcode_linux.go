package shellcode

/*
#include <stdio.h>
#include <sys/mman.h>
#include <string.h>
#include <unistd.h>

void run(char *shellcode, size_t length){
	unsigned char *ptr;
	ptr = (unsigned char*) mmap(0, length, PROT_READ|PROT_WRITE|PROT_EXEC, MAP_ANONYMOUS | MAP_PRIVATE, -1, 0);
	if (ptr == MAP_FAILED) {
		perror("mmap");
		return;
	}
	memcpy(ptr, shellcode, length);
	int (*)
	(*(void(*)()) ptr)();
}
*/
import (
	"fmt"
)

// Runner runner
func Runner(shellcode []byte) bool {
	fmt.Println("[!] Shellcode executor for linux not available")
	return false
}

// IsAvailable does a shellocode runner exist
func IsAvailable() bool {
	return false
}
