package container

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

// 检查 root 是否为挂载点
func isMountPoint(path string) (bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	absPath = filepath.Clean(absPath)

	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}
		mountPoint := fields[4]

		// 处理转义字符（如空格转义为\040）
		mountPoint = strings.ReplaceAll(mountPoint, "\\040", " ")
		mountPoint = strings.ReplaceAll(mountPoint, "\\011", "\t")
		mountPoint = strings.ReplaceAll(mountPoint, "\\012", "\n")
		mountPoint = strings.ReplaceAll(mountPoint, "\\134", "\\")

		if mountPoint == absPath {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// 验证挂载命名空间是否独立
func verifyMountNamespace() error {
	// 获取当前进程的挂载命名空间 ID
	selfMountNs, err := os.Readlink("/proc/self/ns/mnt")
	if err != nil {
		return fmt.Errorf("获取挂载命名空间失败: %v", err)
	}

	// 获取父进程的挂载命名空间 ID
	parentMountNs, err := os.Readlink("/proc/1/ns/mnt")
	if err != nil {
		return fmt.Errorf("获取父进程挂载命名空间失败: %v", err)
	}

	// 如果相同，说明未隔离
	if selfMountNs == parentMountNs {
		return fmt.Errorf("挂载命名空间未隔离！")
	}

	logrus.Printf("挂载命名空间已隔离: %s (子进程) vs %s (父进程)", selfMountNs, parentMountNs)
	return nil
}
