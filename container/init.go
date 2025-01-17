package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

// NewProcess 创建新容器进程并设置好隔离
func NewProcess(tty bool, containerCmd string) *exec.Cmd {
	args := []string{"init", containerCmd}
	cmd := exec.Command("/proc/self/exe", args...)

	// 隔离namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID,
	}

	// 判断是否要新建终端
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd
}

/*
InitProcess 是在容器内部执行的，执行到此容器所在进程已经被创建，这是该容器进程执行的第一个函数
使用 mount 关在proc文件系统，以便后面通过 ps 等系统 命令取查看当前进程资源
需要mount / 要指定为 private ，否则容器内proc会使用外面的proc，即使是在不同的namespace
*/
func InitProcess(containerCmd string, args []string) error {
	/*
		mount 命令的 flags参数可以设置选项以控制挂载行为
		MS_NOEXEC：禁止在该文件系统上执行程序
		MS_NOSUID: 禁止在该文件系统上运行 setuid 或 setgid 程序
		MS_NODEV: 禁止在该文件系统上访问设备文件
	*/
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	// mount, 将默认文件系统类型的空文件系统挂载到根目录
	if err := syscall.Mount("proc", "/proc", "proc", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		logrus.Errorf("mount / fails: %v", err)
		return err
	}
	// mount proc, 将proc文件系统类型的proc文件系统挂载到/proc目录
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	argv := []string{containerCmd}

	/*
		黑魔法！！！
		正常容器运行后发现 用户进程即containerCmd进程并不是Pid=1，因为initProcess是第一个执行的进程
		syscall.Exec 会调用 kernel 内部的 execve 系统函数, 它会覆盖当前进程的镜像、数据和堆栈等信息，Pid不变，将运行的程序替换成另一个。这样就能将用户进程替换init进程称为Pid=1的前台进程、
		用户进程作为Pid=1前台进程，当该进程退出后容器会因为没有前台进程而自动退出，这是docker的特性
	*/
	if err := syscall.Exec(containerCmd, argv, os.Environ()); err != nil {
		logrus.Errorf("mount /proc fails: %v", err)
	}

	return nil
}
