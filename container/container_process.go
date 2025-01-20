package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

// NewProcess 创建新容器进程并设置好隔离, 使用管道来传递多个命令行参数,read端传给容器进程，write端保留在父进程
func NewProcess(tty bool) (*exec.Cmd, *os.File) {
	//args := []string{"init", containerCmd}
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		logrus.Errorf("new pipe error: %v", err)
		return nil, nil
	}

	cmd := exec.Command("/proc/self/exe", "init")

	// 隔离namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
	}

	// 判断是否要新建终端
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// 传递Pipe
	cmd.ExtraFiles = []*os.File{readPipe}
	return cmd, writePipe
}
