package subsystem

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

// CPUSetSubSystem 限制CPU核心数的subsystem
type CPUSetSubSystem struct {
}

func (c *CPUSetSubSystem) Name() string {
	return "CPUSet"
}

// Set 对cgroup设置CPU核心数限制
func (c *CPUSetSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, true); err != nil {
		return err
	} else {
		if res.CPUSet != "" {
			//	设置cgroup内存限制即将限制条件写入cgroupPath对应虚拟文件系统目录中的“cpuset.cpus”文件
			if err = os.WriteFile(path.Join(subsystemCgroupPath, "cpuset.cpus"), []byte(res.MemoryLimit), 0644); err != nil {
				return fmt.Errorf("set cgroup CPUSet fail: %v", err)
			}
		}
		return nil
	}
}

// AddProcess 添加进程到该subsystem
func (c *CPUSetSubSystem) AddProcess(cgroupPath string, pid int) error {
	if subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, false); err != nil {
		return err
	} else {
		// 同样操作，将进程的 pid 写入对应目录中的 'tasks' 文件
		if err = os.WriteFile(path.Join(subsystemCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("cgroup add process fail: %v", err)
		}
	}
	return nil
}

// RemoveCgroup 使用os.Remove移除整个cgroup文件夹，相当于删除group
func (c *CPUSetSubSystem) RemoveCgroup(cgroupPath string) error {
	if subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, false); err != nil {
		return err
	} else {
		return os.Remove(subsystemCgroupPath)
	}
}
