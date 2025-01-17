package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
)

// 挂载了 memory subsystem 的 hierarchy 的根目录位置
const cgroupMemoryHierarchyMount = "/sys/fs/cgroup/memory"

func main() {
	fmt.Println(os.Args[0])
	// 第二次运行该程序, 容器进程
	if os.Args[0] == "/proc/self/exe" {
		// 容器进程
		fmt.Printf("current pid %d\n", syscall.Getpid())
		cmd := exec.Command("sh", "-c", `stress --vm-bytes 200m --vm-keep -m 1`)
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}

	// 第一次运行程序
	// 重新运行一遍当前程序
	cmd := exec.Command("/proc/self/exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start是非阻塞的, 可以接着运行else
	if err := cmd.Start(); err != nil {
		fmt.Println("ERROR!!", err)
		os.Exit(1)
	} else {
		// 得到fork出来的进程在外面namespace的Pid
		fmt.Printf("%v", cmd.Process.Pid)

		// 在系统默认创建挂载了 memory subsystem 的 Hierarchy 上创建 cgroup
		// 0755 表示目录所有者拥有读、写、执行权限，同组用户和其他人拥有读和执行权限。
		os.Mkdir(path.Join(cgroupMemoryHierarchyMount, "testmemorylimit"), 0755)
		// 将容器进程添加进cgroup, 0644 : 所有者读写权限，同组和其他用户只读
		os.WriteFile(path.Join(cgroupMemoryHierarchyMount, "testmemorylimit", "tasks"), []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
		// 限制 cgroup 进程使用内存上限
		os.WriteFile(path.Join(cgroupMemoryHierarchyMount, "testmemorylimit", "memory.linit_in_bytes"), []byte("100m"), 0644)

		cmd.Process.Wait() // 等待子进程完成
	}

}
