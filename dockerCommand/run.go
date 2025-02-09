package dockerCommand

import (
	"MiniDocker/cgroups"
	"MiniDocker/cgroups/subsystem"
	"MiniDocker/container"
	"MiniDocker/network"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

// Run `docker run` 时真正调用的函数
func Run(tty bool, containerCmd []string, res *subsystem.ResourceConfig, volume string, containerName string, imageName string, envSlice []string, nw string, portmapping []string) {
	// 生成10位数字的容器ID
	containerID := randStringBytes(10)
	// 若未指定容器名则以容器ID作为容器名
	if containerName == "" {
		containerName = containerID
	}

	// `docker init <containerCmd>` 创建隔离了namespace的新进程, 返回的写通道口用于传容器命令
	initProcess, writePipe := container.NewProcess(tty, volume, containerName, imageName, envSlice)
	logrus.Infof("parent pid: %v", os.Getpid())
	// start the init process
	if err := initProcess.Start(); err != nil {
		logrus.Error(err)
	}

	// 记录容器信息
	containerName, err := container.RecordContainerInfo(initProcess.Process.Pid, containerCmd, containerName, containerID, volume)
	if err != nil {
		logrus.Errorf("record container info fails: %v", err)
		return
	}

	// 创建 cgroupManager 控制所有 hierarchies层级 的资源配置
	cm := cgroups.NewCgroupManager("simple-docker")
	defer cm.Remove()
	cm.Set(res)
	cm.AddProcess(initProcess.Process.Pid)

	if nw != "" {
		// 配置容器网络
		if err := network.Init(); err != nil {
			logrus.Errorf("init network fails: %v", err)
			return
		}
		containerInfo := &container.ContainerInfo{
			Id:          containerID,
			Pid:         strconv.Itoa(initProcess.Process.Pid),
			Name:        containerName,
			PortMapping: portmapping,
		}
		if err := network.Connect(nw, containerInfo); err != nil {
			logrus.Errorf("Error Connect Network %v", err)
			return
		}
	}

	// 发生容器起始命令
	sendInitCommand(containerCmd, writePipe)

	// 等待进程运行完毕(-it)
	if tty {
		initProcess.Wait()

		// 容器结束运行后清理资源
		//mntURl := "/root/mnt/"
		//rootURL := "/root/"
		container.DeleteContainerInfo(containerName)
		container.DeleteWorkSpace(volume, containerName)
	}

	os.Exit(0)
}

// 通过管道发送容器的起始命令，并关闭通道
func sendInitCommand(containerCmd []string, writePipe *os.File) {
	cmdString := strings.Join(containerCmd, " ")
	logrus.Infof("init command is: %v", cmdString)
	writePipe.WriteString(cmdString)
	writePipe.Close()
}
