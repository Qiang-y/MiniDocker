package dockerCommand

import (
	"MiniDocker/container"
	"github.com/sirupsen/logrus"
	"os"
)

// Run `docker run` 时真正调用的函数
func Run(tty bool, containerCmd string) {
	// `docker init <containerCmd>` 创建隔离了namespace的新进程
	initProcess := container.NewProcess(tty, containerCmd)

	// start the init process
	if err := initProcess.Start(); err != nil {
		logrus.Error(err)
	}

	// 等待进程运行完毕
	if err := initProcess.Wait(); err != nil {
		return
	}
	os.Exit(-1)
}
