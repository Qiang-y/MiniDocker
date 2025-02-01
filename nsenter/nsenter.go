package nsenter

/*
#define _GNU_SOURCE
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <unistd.h>

// 该函数类似于包的构造函数
__attribute__((constructor)) void enter_namespace(void) {
	char *minidocker_pid;
	// 从环境变量中获取需要进入的PID
	minidocker_pid = getenv("minidocker_pid");
	if (minidocker_pid) {
		fprintf(stdout, "c- got mydocker_pid=%s\n", minidocker_pid);
	} else {
		fprintf(stdout, "c- missing mydocker_pid env skip nsenter");
		// 对于没有指定pid的直接退出，即run等命令不会接着执行
		return;
	}
	char *minidocker_cmd;
	// 从环境变量中获取需要执行的命令
	minidocker_cmd = getenv("minidocker_cmd");
	if (minidocker_cmd) {
		fprintf(stdout, "c- got mydocker_cmd=%s\n", minidocker_pid);
	} else {
		fprintf(stdout, "c- missing mydocker_cmd env skip nsenter");
		return;
	}
	int i;
	char nspath[1024];
	// 需要进入的5中namespace
	char *namespaces[] = { "ipc", "uts", "net", "pid", "mnt" };

	for (i=0; i<5; i++) {
		// 拼接对应路径 '/proc/${pid}/ns/${namespace}'
		sprintf(nspath, "/proc/%s/ns/%s", minidocker_pid, namespaces[i]);
		int fd = open(nspath, O_RDONLY);

		// 通过setns系统调用进入命名空间
		if (setns(fd, 0) == -1) {
			fprintf(stderr, "c- setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
		} else {
			fprintf(stdout, "c- setns on %s namespace succeeded\n", namespaces[i]);
		}
		close(fd);
	}
	// 在进入的namespace中执行指定的命令
	int res = system(minidocker_cmd);
	exit(0);
	return;
}
*/
import "C"
