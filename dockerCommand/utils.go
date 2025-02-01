package dockerCommand

import (
	"MiniDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

// 根据容器名得到对应容器的PID
func getContainerPidByName(containerName string) (string, error) {
	// 得到对应容器信息
	configDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configDir = filepath.Join(configDir, container.ConfigName)
	// read config file
	contentBytes, err := os.ReadFile(configDir)
	if err != nil {
		logrus.Errorf("read config file %v fails: %v", configDir, err)
		return "", err
	}
	var containerInfo container.ContainerInfo
	// unmarshal
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return "", err
	}
	return containerInfo.Pid, nil
}

// 根据容器名得到对应容器的信息
func getContainerInfoByName(containerName string) (*container.ContainerInfo, error) {
	infoDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	infoDir = filepath.Join(infoDir, container.ConfigName)
	contentBytes, err := os.ReadFile(infoDir)
	if err != nil {
		logrus.Errorf("Read file %v fails: %v", infoDir, err)
		return nil, err
	}
	var containerInfo = &container.ContainerInfo{}
	if err := json.Unmarshal(contentBytes, containerInfo); err != nil {
		logrus.Errorf("Unmarshal file %v fails: %v", infoDir, err)
		return nil, err
	}
	return containerInfo, nil
}
