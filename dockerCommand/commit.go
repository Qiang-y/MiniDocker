package dockerCommand

import (
	"github.com/sirupsen/logrus"
	"os/exec"
)

// CommitContainer 将容器文件系统打包成${imageName}.tar
func CommitContainer(imageName string) {
	mntURL := "/root/mnt"
	imageTar := "/root/" + imageName + ".tar"
	logrus.Infof("imageTar: %v", imageTar)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		logrus.Errorf("Tar folder: %v fails: %v", mntURL, err)
	}
}
