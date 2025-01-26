package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

/*
InitProcess 是在容器内部执行的，执行到此容器所在进程已经被创建，这是该容器进程执行的第一个函数
使用 mount 关在proc文件系统，以便后面通过 ps 等系统 命令取查看当前进程资源
需要mount / 要指定为 private ，否则容器内proc会使用外面的proc，即使是在不同的namespace
*/
func InitProcess() error {
	// 获取命令参数
	containerCmd := readCommand()
	if containerCmd == nil || len(containerCmd) == 0 {
		return fmt.Errorf("init process fails, containerCmd is nil")
	}

	/*
		mount 命令的 flags参数可以设置选项以控制挂载行为
		MS_NOEXEC：禁止在该文件系统上执行程序
		MS_NOSUID: 禁止在该文件系统上运行 setuid 或 setgid 程序
		MS_NODEV: 禁止在该文件系统上访问设备文件
	*/
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	// mount, 将默认文件系统类型的空文件系统挂载到根目录
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		logrus.Errorf("mount / fails: %v", err)
		return err
	}
	// mount proc, 将proc文件系统类型的proc文件系统挂载到/proc目录
	err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logrus.Errorf("mount /proc fails: %v", err)
	}
	//argv := []string{containerCmd}

	// LookPath 查到参数命令的绝对路径
	path, err := exec.LookPath(containerCmd[0])
	if err != nil {
		logrus.Errorf("initProcess look path fails: %v", err)
		return nil
	}
	logrus.Infof("Find path: %v", path)

	/*
		黑魔法！！！
		正常容器运行后发现 用户进程即containerCmd进程并不是Pid=1，因为initProcess是第一个执行的进程
		syscall.Exec 会调用 kernel 内部的 execve 系统函数, 它会覆盖当前进程的镜像、数据和堆栈等信息，Pid不变，将运行的程序替换成另一个。这样就能将用户进程替换init进程称为Pid=1的前台进程、
		用户进程作为Pid=1前台进程，当该进程退出后容器会因为没有前台进程而自动退出，这是docker的特性
	*/
	if err := syscall.Exec(path, containerCmd, os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}

	return nil
}

func readCommand() []string {
	// 在新建进程时除了3个标准io操作，将管道作为额外的第四个文件传入，因此管道的fd为3\
	// 如果父进程没有传入数据则会阻塞等待
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := io.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("read pipe fails: %v", err)
		return nil
	}
	return strings.Split(string(msg), " ")
}
