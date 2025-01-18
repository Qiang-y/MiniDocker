package subsystem

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

// MemorySubsystem memory大小限制的subsystem实现
type MemorySubsystem struct {
}

func (ms *MemorySubsystem) Name() string {
	return "memory"
}

// Set 对cgroup设置内存大小限制
func (ms *MemorySubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsystemCgroupPath, err := GetCgroupPath(ms.Name(), cgroupPath, true); err != nil {
		return err
	} else {
		if res.MemoryLimit != "" {
			//	设置cgroup内存限制即将限制条件写入cgroupPath对应虚拟文件系统目录中的“memory.limit_in_bytes”文件
			if err = os.WriteFile(path.Join(subsystemCgroupPath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), 0644); err != nil {
				return fmt.Errorf("set cgroup memory fail: %v", err)
			}
		}
		return nil
	}
}

// AddProcess 添加进程到该subsystem
func (ms *MemorySubsystem) AddProcess(cgroupPath string, pid int) error {
	if subsystemCgroupPath, err := GetCgroupPath(ms.Name(), cgroupPath, false); err != nil {
		return err
	} else {
		// 同样操作，将进程的 pid 写入对应目录中的 'task' 文件
		if err = os.WriteFile(path.Join(subsystemCgroupPath, "task"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("cgroup add process fail: %v", err)
		}
	}
	return nil
}

// RemoveCgroup 使用os.Remove移除整个cgroup文件夹，相当于删除group
func (ms *MemorySubsystem) RemoveCgroup(cgroupPath string) error {
	if subsystemCgroupPath, err := GetCgroupPath(ms.Name(), cgroupPath, false); err != nil {
		return err
	} else {
		return os.Remove(subsystemCgroupPath)
	}
}
