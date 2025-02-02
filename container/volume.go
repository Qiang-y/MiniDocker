package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NewWorkSpace 创建容器文件系统
func NewWorkSpace(imageName, containerName, volume string) {
	// 创建只读、读写层并挂载到/root/mnt
	CreateReadOnlyLayer(imageName)
	CreateWriteLayer(containerName)
	CreateMountPoint(containerName, imageName)

	// 判断volume是否要挂载数据卷
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(volumeURLs, containerName)
			logrus.Infof("%q", volumeURLs)
		} else {
			logrus.Infof("Volume parameter input is not correct.")
		}
	}
}

// CreateReadOnlyLayer 新建 busybox 文件夹，将 busybox.tar 解压到 busybox 目录下，作为容器的只读层
// 解压tar格式的镜像文件作为只读层
func CreateReadOnlyLayer(imageName string) {
	unTarFolderUrl := filepath.Join(RootUrl, imageName)
	imageUrl := filepath.Join(RootUrl, imageName) + ".tar"
	logrus.Infof("unTarfolder url: %v", unTarFolderUrl)
	logrus.Infof("image url: %v", imageUrl)
	exist, err := PathExists(unTarFolderUrl)
	if err != nil {
		logrus.Infof("Fail to judge whether dir %v exists: %v", exist, err)
	}
	if exist == false {
		// 新建 镜像只读 目录
		if err := os.MkdirAll(unTarFolderUrl, 0777); err != nil {
			logrus.Errorf("Mkdir dir %v fails: %v", unTarFolderUrl, err)
		}
		// 解压 [镜像].tar
		if _, err := exec.Command("tar", "-xvf", imageUrl, "-C", unTarFolderUrl).CombinedOutput(); err != nil {
			logrus.Errorf("unTar dit %v fails: %v", unTarFolderUrl, err)
		}
	}
}

// CreateWriteLayer 为容器创建 writeLayer 文件夹作为容器 唯一 可写层
func CreateWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.MkdirAll(writeURL, 0777); err != nil {
		logrus.Errorf("Mkdir dir %v fails: %v", writeURL, err)
	}
}

// CreateMountPoint 新建 mnt 文件夹作为挂载点，并将 writeLayer 目录和 busybox(image) 目录 mount 到 mnt 目录下
// ubuntu22.04 内核不支持AUFS，使用OverLay代替
func CreateMountPoint(containerName, imageName string) {
	// 创建mnt文件夹作为挂载点
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	if err := os.MkdirAll(mntUrl, 0777); err != nil {
		logrus.Errorf("Mkdir dir %v fails: %v", mntUrl, err)
	}

	// 创建临时工作文件夹
	workURL := fmt.Sprintf(WorkLayerUrl, containerName)
	if err := os.MkdirAll(workURL, 0777); err != nil {
		logrus.Errorf("Mkdir dir %v fails: %v", workURL, err)
	}

	// 将writeLayer目录和image目录mount到mnt目录下
	// 改用OverLay
	tmpWriteLayer := fmt.Sprintf(WriteLayerUrl, containerName)
	tmpImageLocation := filepath.Join(RootUrl, imageName)
	dirs := fmt.Sprintf(
		"lowerdir=%s,upperdir=%s,workdir=%s",
		tmpImageLocation,
		tmpWriteLayer,
		workURL,
	)
	cmd := exec.Command("mount", "-t", "overlay", "-o", dirs, "overlay", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// 启动命令并阻塞等待
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
}

// DeleteWorkSpace Docker 删除容器时将容器对应的writeLayer和Container-initLayer删除，
// 从而保留镜像所有内容，
// 简化操作，在容器退出时便删除writeLayer和work
func DeleteWorkSpace(volume, containerName string) {
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			DeleteMountPointWithVolume(volumeURLs, containerName)
		} else {
			DeleteMountPoint(containerName)
		}
	} else {
		DeleteMountPoint(containerName)
	}
	DeleteWriteLayer(containerName)
}

// DeleteMountPoint 删除为挂载数据卷的容器文件系统，先unmount mnt目录，后删除mnt目录
func DeleteMountPoint(containerName string) {
	mntURL := fmt.Sprintf(MntUrl, containerName)
	logrus.Infof("mntURL: %v", mntURL)
	cmd := exec.Command("umount", mntURL)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	if _, err := cmd.CombinedOutput(); err != nil {
		logrus.Errorf("umount mnt fails: %v", err)
	}
	// delete mnt/
	if err := os.RemoveAll(mntURL); err != nil {
		logrus.Errorf("remove mnt dir: %v fails: %v", mntURL, err)
	}
}

// DeleteMountPointWithVolume 删除有挂载 volume 数据卷的容器文件系统
func DeleteMountPointWithVolume(volumeURLs []string, containerName string) {
	// 卸载容器里volume挂载点的文件系统
	mntURL := fmt.Sprintf(MntUrl, containerName)
	containerUrl := filepath.Join(mntURL, volumeURLs[1])
	cmd := exec.Command("umount", containerUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("umount volume dir: %v fails: %v", containerUrl, err)
	}
	// 卸载整个容器文件系统的挂载点
	cmd = exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("remove mnt dir: %v fails: %v", mntURL, err)
	}
	// 删除容器文件系统的挂载点
	if err := os.RemoveAll(mntURL); err != nil {
		logrus.Errorf("remove mnt dir: %v fails: %v", mntURL, err)
	}
	// 删除 volume 工作的临时目录
	tmpWorkDir := filepath.Join(volumeURLs[0], "..", ".volumeWork")
	if err := os.RemoveAll(tmpWorkDir); err != nil {
		logrus.Infof("remove volume tmpwork dir: %v fails: %v", mntURL, err)
	}
}

// DeleteWriteLayer 删除writeLayer目录和临时work目录，即抹去容器对文件系统的更改
func DeleteWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.RemoveAll(writeURL); err != nil {
		logrus.Errorf("remove writeLayer dir: %v fails: %v", writeURL, err)
	}
	workURL := fmt.Sprintf(WorkLayerUrl, containerName)
	if err := os.RemoveAll(workURL); err != nil {
		logrus.Errorf("remove tmpWork dir: %v fails: %v", workURL, err)
	}
}

// 解析 volume 字符串
func volumeUrlExtract(volume string) []string {
	return strings.Split(volume, ":")
}

/*
MountVolume 挂载数据卷进容器
1.读取宿主机文件目录 URL，创建宿主机文件目录 (/root/${parentURL})
2.读取容器挂载点 URL，在容器文件系统里创建挂载点 (/root/mnt/${containerURL})
3.把宿主机文件目录挂载到容器挂载点
*/
func MountVolume(volumeURLs []string, containerName string) {
	// create host file catalog
	parentUrl := volumeURLs[0]
	if err := os.MkdirAll(parentUrl, 0777); err != nil {
		logrus.Infof("Mkdir parent dir: %v error: %v", parentUrl, err)
	}
	// create mount point in container file system
	containerUrl := volumeURLs[1]
	mntURL := fmt.Sprintf(MntUrl, containerName)
	containerVolumeURL := filepath.Join(mntURL, containerUrl)
	if err := os.MkdirAll(containerVolumeURL, 0777); err != nil {
		logrus.Infof("Mkdir container dir: %v error: %v", containerVolumeURL, err)
	}

	// 为overlay挂载创建必须的lower和work目录，确保work目录为空
	tmpWorkDir := filepath.Join(parentUrl, "..", ".volumeWork")
	lowerDir := filepath.Join(tmpWorkDir, ".emptyLower")
	logrus.Infof("lowerDir: %v", lowerDir)
	workDir := filepath.Join(tmpWorkDir, ".work")
	if err := os.MkdirAll(lowerDir, 0777); err != nil {
		logrus.Errorf("Mkdir .lower dir: %v fails: %v", lowerDir, err)
	}
	// 确保work目录是空的
	if err := os.RemoveAll(workDir); err == nil {
		if err := os.MkdirAll(workDir, 0777); err != nil {
			logrus.Errorf("Mkdir .work dir: %v fails :%v", workDir, err)
		}
	}

	// mount host file catalog to mount point in container
	options := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, parentUrl, workDir)
	cmd := exec.Command("mount", "-t", "overlay", "-o", options, "overlay", containerVolumeURL)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	if _, err := cmd.CombinedOutput(); err != nil {
		logrus.Errorf("Mount volune fails: %v", err)
	}
}
