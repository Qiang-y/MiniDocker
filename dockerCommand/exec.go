package dockerCommand

import (
	"MiniDocker/container"
	_ "MiniDocker/nsenter" // 必须引用该包C程序才能运行
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const ENV_EXEC_PID = "minidocker_pid"
const ENV_EXEC_CMD = "minidocker_cmd"

// ExecContainer 进入容器并执行命令
func ExecContainer(containerName string, containerCmds []string) {
	// 获取容器pid
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		logrus.Errorf("exec container get pid by name %v fails: %v", containerName, err)
		return
	}
	// 将命令以空格为分隔符拼接成一个字符串，方便传递
	cmdStr := strings.Join(containerCmds, " ")
	logrus.Infof("container pid: %s, command: %s ", pid, cmdStr)

	// 在设定好环境变量后重新执行exec，进入nsenter分支执行setns系统调用
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)

	if err := cmd.Run(); err != nil {
		logrus.Errorf("exec container %v fails: %v", containerName, err)
	}
}

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
