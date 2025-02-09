package dockerCommand

import (
	"MiniDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ENV_PATH = "/proc/%s/environ"

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

// 通过pid得到对应进程的环境变量
func getEnvByPid(pid string) []string {
	// 进程环境变量存放位置 /proc/{PID}/environ
	path := fmt.Sprintf(ENV_PATH, pid)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		logrus.Errorf("read file: %v fails: %v", path, err)
		return nil
	}
	// 多个环境变量的分隔符是 \u0000
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}

// 生成容器的唯一ID标识
func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(uint64(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
