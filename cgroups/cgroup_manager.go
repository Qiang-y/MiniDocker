package cgroups

import (
	"MiniDocker/cgroups/subsystem"
	"github.com/sirupsen/logrus"
)

/*
CgroupManager 管理cgroup,配置资源限制,以及将进程移动到cgroup中操作交给各个subsystem
工作流程：CgroupManager 在配置容器资源限制时，首先会初始化Subsystem的实例，然后遍历Subsystem实例中的Set方法，

	创建和配置不同的Subsystem挂载的hierarchy中的cgroup，最后通过调用Subsystem实例将容器分别加入哪些cgroup中，实现容器的资源限制
*/
type CgroupManager struct {
	// Path 是相对于hierarchy的root的相对路径，因此一个CgroupManager可以表示多个cgroups
	Path string

	// 限制条件
	Resource *subsystem.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

// Set 设置subsystem到cgroup中，如果cgroup路径不存在会新建
// 这可能会创还能多个cgroups，如果他们不在同一个hierarchy中
func (cm *CgroupManager) Set(res *subsystem.ResourceConfig) error {
	for _, subs := range subsystem.SubsystemsInstance {
		if err := subs.Set(cm.Path, res); err != nil {
			logrus.Warnf("set resource fail: %v", err)
		}
	}
	return nil
}

func (cm *CgroupManager) AddProcess(pid int) error {
	for _, subs := range subsystem.SubsystemsInstance {
		if err := subs.AddProcess(cm.Path, pid); err != nil {
			logrus.Warnf("add process fail: %v", err)
		}
	}
	return nil
}

func (cm *CgroupManager) Remove() error {
	for _, subs := range subsystem.SubsystemsInstance {
		if err := subs.RemoveCgroup(cm.Path); err != nil {
			return err
		}
	}
	return nil
}
