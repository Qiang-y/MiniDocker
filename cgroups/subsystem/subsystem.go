package subsystem

// ResourceConfig 传递资源限制
type ResourceConfig struct {
	MemoryLimit string // 内存限制
	CPUShare    string // CPU时间片权重
	CPUSet      string // CPU核心数
}

// Subsystem 接口，每个subsystem都要实现
// cgroup抽象成path，因为cgroup在hierarchy的路径便是虚拟文件系统中的虚拟路径
type Subsystem interface {
	// Name 返回subsystem的名字，如cpu memory
	Name() string

	// Set 设置某个cgroup在该Subsystem中的资源限制
	Set(path string, res *ResourceConfig) error

	// AddProcess 将进程添加入某cgroup
	AddProcess(path string, pid int) error

	// RemoveCgroup 移除某个cgroup
	RemoveCgroup(path string) error
}

var SubsystemsInstance = []Subsystem{
	&MemorySubSystem{},
	&CPUSubSystem{},
	&CPUSetSubSystem{},
}
