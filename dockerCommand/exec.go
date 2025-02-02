package dockerCommand

import (
	_ "MiniDocker/nsenter" // 必须引用该包C程序才能运行
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

const ENV_EXEC_PID = "minidocker_pid"
const ENV_EXEC_CMD = "minidocker_cmd"

// ExecContainer 进入容器并执行命令
func ExecContainer(containerName string, containerCmds []string) {
	// 获取容器主进程pid
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
	// 获取容器主进程的环境变量
	containerEnvs := getEnvByPid(pid)
	// 将主机环境变量和容器环境变量一起放入执行exec的进程中
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err := cmd.Run(); err != nil {
		logrus.Errorf("exec container %v fails: %v", containerName, err)
	}
}
