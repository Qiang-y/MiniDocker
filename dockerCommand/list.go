package dockerCommand

import (
	"MiniDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"text/tabwriter"
)

// ListContainers 打印所有容器信息
func ListContainers() {
	// 容器信息存储路径'/var/run/minidocker'
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, "")
	dirUrl = dirUrl[:len(dirUrl)-1]
	// 读取该文件夹下所有文件
	files, err := os.ReadDir(dirUrl)
	if err != nil {
		logrus.Errorf("read dir %v error: %v", dirUrl, err)
		return
	}

	var containers []*container.ContainerInfo
	for _, file := range files {
		// 根据容器配置文件获取对应信息，并转化为ContainerInfo对象
		fileInfo, _ := file.Info()
		tmpContainer, err := getContainerInfo(fileInfo)
		if err != nil {
			logrus.Errorf("get container: %v info fails: %v", file.Name(), err)
			continue
		}
		containers = append(containers, tmpContainer)
	}

	// tabwriter 是引用的text/tabwriter类库，用于在控制台打印对其的表格
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	// 控制台输出的信息列
	_, _ = fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreatedTime)
	}
	// 刷新标志输出缓冲区，将容器列表打印出来
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush fails: %v", err)
		return
	}
}

// 从文件中的到容器信息描述符
func getContainerInfo(file os.FileInfo) (*container.ContainerInfo, error) {
	// get file name
	containerName := file.Name()
	// 根据文件名生成文件绝对路径
	configFileDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFileDir = filepath.Join(configFileDir, container.ConfigName)
	// 读取config.json内容器信息
	content, err := os.ReadFile(configFileDir)
	if err != nil {
		logrus.Errorf("read file: %v error: %v", configFileDir, err)
		return nil, err
	}
	// 解析json信息
	var containerInfo = &container.ContainerInfo{}
	if err := json.Unmarshal(content, containerInfo); err != nil {
		logrus.Errorf("Json unmarshal fails: %v", err)
		return nil, err
	}
	return containerInfo, nil
}
