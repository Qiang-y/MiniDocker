package dockerCommand

import (
	"MiniDocker/cgroups"
	"MiniDocker/cgroups/subsystem"
	"MiniDocker/container"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

// Run `docker run` 时真正调用的函数
func Run(tty bool, containerCmd []string, res *subsystem.ResourceConfig, volume string) {
	// `docker init <containerCmd>` 创建隔离了namespace的新进程, 返回的写通道口用于传容器命令
	initProcess, writePipe := container.NewProcess(tty, volume)
	logrus.Infof("parent pid: %v", os.Getpid())
	// start the init process
	if err := initProcess.Start(); err != nil {
		logrus.Error(err)
	}

	// 创建 cgroupManager 控制所有 hierarchies层级 的资源配置
	cm := cgroups.NewCgroupManager("simple-docker")
	defer cm.Remove()
	cm.Set(res)
	cm.AddProcess(initProcess.Process.Pid)

	// 发生容器起始命令
	sendInitCommand(containerCmd, writePipe)

	// 等待进程运行完毕
	if err := initProcess.Wait(); err != nil {
		//return
	}

	// 容器结束运行后清理资源
	mntURl := "/root/mnt/"
	rootURL := "/root/"
	container.DeleteWorkSpace(rootURL, mntURl, volume)

	os.Exit(0)
}

// 通过管道发送容器的起始命令，并关闭通道
func sendInitCommand(containerCmd []string, writePipe *os.File) {
	cmdString := strings.Join(containerCmd, " ")
	logrus.Infof("init command is: %v", cmdString)
	writePipe.WriteString(cmdString)
	writePipe.Close()
}
