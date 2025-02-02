package dockerCommand

import (
	"MiniDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// StopContainer 停止容器运行
func StopContainer(containerName string) {
	// 得到容器主进程pid
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		logrus.Errorf("get container %v fails: %v", containerName, err)
		return
	}
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		logrus.Errorf("conver pid fails: %v", err)
		return
	}
	// 通过kill系统调用发送SIGTERM型号给容器主进程，使其优雅退出，从而停止容器
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		logrus.Warnf("stop container %v fails: %v", containerName, err)
		// 有时候进程因为别的什么原因已经被kill了
		//return
	}
	// 至此容器进程已被kill掉，接下来修改容器Info信息
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		logrus.Errorf("get container %v fails: %v", containerName, err)
		return
	}
	containerInfo.Status = container.STOP
	containerInfo.Pid = " "
	// 将修改后的信息覆盖至配置文件中
	newInfoBytes, err := json.Marshal(containerInfo)
	if err != nil {
		logrus.Errorf("Json marshal %s fails: %v", containerName, err)
		return
	}
	infoDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	infoDir = filepath.Join(infoDir, container.ConfigName)
	if err := os.WriteFile(infoDir, newInfoBytes, 0622); err != nil {
		logrus.Errorf("write file %s fails: %v", infoDir, err)
	}
}
