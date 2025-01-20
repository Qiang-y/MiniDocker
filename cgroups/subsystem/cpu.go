package subsystem

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

// CPUSubSystem 对CPU时间片进行限制的subsystem
type CPUSubSystem struct {
}

func (c *CPUSubSystem) Name() string {
	return "CPUShare"
}

// Set 对cgroup设置cpu时间片
func (c *CPUSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, true); err != nil {
		return nil
	} else {
		if res.CPUShare != "" {
			//	设置cgroup的CPU限制即将限制条件写入cgroupPath对应虚拟文件系统目录中的"cpu.shares"文件
			if err = os.WriteFile(path.Join(subsystemCgroupPath, "cpu.shares"), []byte(res.MemoryLimit), 0644); err != nil {
				return fmt.Errorf("set cgroup CPU share fail: %v", err)
			}
		}
		return nil
	}
}

// AddProcess 添加进程到该subsystem
func (c *CPUSubSystem) AddProcess(cgroupPath string, pid int) error {
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
func (c *CPUSubSystem) RemoveCgroup(cgroupPath string) error {
	if subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, false); err != nil {
		return err
	} else {
		return os.Remove(subsystemCgroupPath)
	}
}
