package container

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	RUNNING             = "running"
	STOP                = "stopped"
	EXIT                = "exited"
	DefaultInfoLocation = "/var/run/minidocker/%s/"
	ConfigName          = "config.json"
	ContainerLogFile    = "container.log"
	RootUrl             = "/root/"
	MntUrl              = "/root/mnt/%s"
	WriteLayerUrl       = "/root/writeLayer/%s"
	WorkLayerUrl        = "/root/.tmpWork/%s"
)

// ContainerInfo 容器的基本信息, 默认存储在'/var/run/minidocker/${containerName}/config.json'
type ContainerInfo struct {
	Pid         string `json:"pid"`         // 容器的init进程在主机上的pid
	Id          string `json:"id"`          // 容器ID
	Name        string `json:"name"`        // 容器名
	Command     string `json:"command"`     // 容器内init进程运行的命令
	CreatedTime string `json:"createdTime"` // 创建时间
	Status      string `json:"status"`      // 容器状态
	Volume      string `json:"volume"`      // 挂载数据卷
}

// RecordContainerInfo 记录容器信息
// @return 容器名 或 错误信息
func RecordContainerInfo(containerPID int, containerCmd []string, containerName string, volume string) (string, error) {
	// 生成10位数字的容器ID
	id := randStringBytes(10)
	// 记录当前容器创建时间和初始命令
	createTime := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(containerCmd, "")
	// 若未指定容器名则以容器ID作为容器名
	if containerName == "" {
		containerName = id
	}
	// 生成容器信息结构体实例
	containerInfo := &ContainerInfo{
		Pid:         strconv.Itoa(containerPID),
		Id:          id,
		Name:        containerName,
		Command:     command,
		CreatedTime: createTime,
		Status:      RUNNING,
		Volume:      volume,
	}
	// 将容器信息转为json字符串
	jsonByte, err := json.Marshal(containerInfo)
	if err != nil {
		logrus.Errorf("Record container info fails: %v", err)
		return "", err
	}
	jsonStr := string(jsonByte)

	// 拼凑存储容器信息的路径，并确保存在
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		logrus.Errorf("Mkdir dir: %v fails: %v", dirUrl, err)
		return "", err
	}
	fileName := filepath.Join(dirUrl, ConfigName)
	logrus.Infof("containerInfo file name: %v", fileName)
	// 创建存储文件
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		logrus.Errorf("create file: %v fails: %v", fileName, err)
		return "", err
	}
	// 将json序列化的数据写入文件
	if _, err := file.WriteString(jsonStr); err != nil {
		logrus.Errorf("write file: %v fails: %v", fileName, err)
		return "", err
	}
	return containerName, nil
}

// DeleteContainerInfo 删除容器信息
func DeleteContainerInfo(containerName string) {
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirUrl); err != nil {
		logrus.Errorf("remove dir: %v fails: %v", dirUrl, err)
	}
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
