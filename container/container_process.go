package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// NewProcess 创建新容器进程并设置好隔离, 使用管道来传递多个命令行参数,read端传给容器进程，write端保留在父进程
func NewProcess(tty bool, volume string, containerName string, imageName string, envSlice []string) (*exec.Cmd, *os.File) {
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

	// 判断是否要新建终端，否则将日志重定向至'/var/run/minidocker/${containerName}/container.log'
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// 后台容器需将日志重定向
		logdir := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(logdir, 0622); err != nil {
			logrus.Errorf("mkdir log dir: %v fails: %v", logdir, err)
			return nil, nil
		}
		stdLogFilePath := filepath.Join(logdir, ContainerLogFile)
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			logrus.Errorf("create file %v fails: %v", stdLogFilePath, err)
			return nil, nil
		}
		cmd.Stdout = stdLogFile
	}

	// 传递Pipe
	cmd.ExtraFiles = []*os.File{readPipe}
	// 传递环境变量
	cmd.Env = append(os.Environ(), envSlice...)

	NewWorkSpace(imageName, containerName, volume)
	cmd.Dir = fmt.Sprintf(MntUrl, containerName)

	return cmd, writePipe
}
