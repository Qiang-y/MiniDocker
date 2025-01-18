package subsystem

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

/*
FindCgroupMountpoint 通过在/proc/self/mountinfo 中获取该 hierarchy 的 cgroup 根节点的路径
*/
func FindCgroupMountpoint(subsystemName string) string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")

		// 查找 subsystemName 是否出现在最后一条记录中，如果是，第五个字段为根路径
		// 对于大多数系统来说，cgroup 的挂载点信息会放在最后
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			if opt == subsystemName {
				return fields[4]
			}
		}
	}
	return ""
}

/*
GetCgroupPath 用于获取某个 subsystem 所挂载的 hierarchy 上的虚拟文件系统(挂载后的文件夹)下的cgroup的路径。
通过对这个目录的改写来改动cgroup
autoCreate: 为true且该路径不存在，则新建一个 cgroup (在 hierarchy 环境下，mkdir会隐式地创建一个cgroup，其中包含许多配置文件)
*/
func GetCgroupPath(subsystemName string, cgroupPath string, autoCreate bool) (string, error) {
	// 找到cgroup的hierarchy挂载的根目录
	cgroupRootPath := FindCgroupMountpoint(subsystemName)
	expectedPath := path.Join(cgroupRootPath, cgroupPath)

	// 用Stat检查路径是否存在
	if _, err := os.Stat(expectedPath); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err := os.Mkdir(expectedPath, 0755); err != nil {
				return "", fmt.Errorf("error when create cgroup: %v", err)
			}
		}
		return expectedPath, nil
	} else {
		return "", fmt.Errorf("cgroup path error: %v", err)
	}
}
