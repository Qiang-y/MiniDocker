package dockerCommand

import (
	"MiniDocker/container"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

// RemoveContainer 删除容器
func RemoveContainer(containerName string) {
	// get the information of container
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		logrus.Errorf("get container %s information fails: %v", containerName, err)
		return
	}
	// only remove the stop container
	if containerInfo.Status != container.STOP {
		logrus.Errorf("only can remove the stop container")
		return
	}
	infoDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	// remove all path
	if err := os.RemoveAll(infoDir); err != nil {
		logrus.Errorf("remove file %s fails: %v", infoDir, err)
	}
}
