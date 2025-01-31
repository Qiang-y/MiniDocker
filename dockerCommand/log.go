package dockerCommand

import (
	"MiniDocker/container"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
)

// LogContainer 打印容器日志
func LogContainer(containerName string) {
	// find the log file
	logDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logDir = filepath.Join(logDir, container.ContainerLogFile)
	// open log file
	file, err := os.Open(logDir)
	defer file.Close()
	if err != nil {
		logrus.Errorf("log contanier open file: %v fails: %v", logDir, err)
		return
	}
	// read log file
	content, err := io.ReadAll(file)
	if err != nil {
		logrus.Errorf("log container read file: %v fails: %v", logDir, err)
		return
	}
	// 将文件内容输出到终端
	_, _ = fmt.Fprintf(os.Stdout, string(content))
}
