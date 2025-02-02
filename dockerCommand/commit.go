package dockerCommand

import (
	"MiniDocker/container"
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
)

// CommitContainer 将容器文件系统打包成${imageName}.tar
func CommitContainer(containerName, imageName string) {
	mntURL := fmt.Sprintf(container.MntUrl, containerName)
	mntURL += "/"
	imageTar := container.RootUrl + "/" + imageName + ".tar"
	logrus.Infof("containerURL: %v", mntURL)
	logrus.Infof("imageTar: %v", imageTar)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		logrus.Errorf("Tar folder: %v fails: %v", mntURL, err)
	}
}
