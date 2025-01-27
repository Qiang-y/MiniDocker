package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

/*
InitProcess 是在容器内部执行的，执行到此容器所在进程已经被创建，这是该容器进程执行的第一个函数
使用 mount 关在proc文件系统，以便后面通过 ps 等系统 命令取查看当前进程资源
需要mount / 要指定为 private ，否则容器内proc会使用外面的proc，即使是在不同的namespace
*/
func InitProcess() error {
	// 验证是否处于独立的挂载命名空间
	if err := verifyMountNamespace(); err != nil {
		return err
	}

	// 获取命令参数
	containerCmd := readCommand()
	if containerCmd == nil || len(containerCmd) == 0 {
		return fmt.Errorf("init process fails, containerCmd is nil")
	}

	if err := setUpMount(); err != nil {
		logrus.Errorf("initProcess setUpMount fails: %v", err)
		return err
	}
	//argv := []string{containerCmd}

	// LookPath 查到参数命令的绝对路径
	path, err := exec.LookPath(containerCmd[0])
	if err != nil {
		logrus.Errorf("initProcess look path fails: %v", err)
		return nil
	}
	logrus.Infof("Find path: %v", path)

	/*
		黑魔法！！！
		正常容器运行后发现 用户进程即containerCmd进程并不是Pid=1，因为initProcess是第一个执行的进程
		syscall.Exec 会调用 kernel 内部的 execve 系统函数, 它会覆盖当前进程的镜像、数据和堆栈等信息，Pid不变，将运行的程序替换成另一个。这样就能将用户进程替换init进程称为Pid=1的前台进程、
		用户进程作为Pid=1前台进程，当该进程退出后容器会因为没有前台进程而自动退出，这是docker的特性
	*/
	if err := syscall.Exec(path, containerCmd, os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}

	return nil
}

// setUpMount init 挂载点
func setUpMount() error {
	// 获取当前路径
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("get current location error: %v", err)
		return err
	}
	logrus.Infof("current location: %v", pwd)

	/*
		mount 命令的 flags参数可以设置选项以控制挂载行为
		MS_NOEXEC：禁止在该文件系统上执行程序
		MS_NOSUID: 禁止在该文件系统上运行 setuid 或 setgid 程序
		MS_NODEV: 禁止在该文件系统上访问设备文件
	*/
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	// mount, 将挂载命名空间的传播模式设置为 MS_PRIVATE，阻止挂载事件传播到宿主机
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		logrus.Errorf("mount / fails: %v", err)
		return err
	}

	if err := pivotRoot(pwd); err != nil {
		logrus.Errorf("pivot root fails: %v", err)
		return err
	}

	// mount proc, 将proc文件系统类型的proc文件系统挂载到/proc目录
	err = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logrus.Errorf("mount /proc fails: %v", err)
	}

	// tmpfs 是一种基于内存的文件系统，用 RAM 或 swap 分区来存储, 提供临时设备文件存储
	if err := syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755"); err != nil {
		logrus.Errorf("mount /dev failed: %v", err)
		return err
	}
	return nil
}

// pivotRoot 通过pivot_root系统调用切换当前的root文件系统
/*
# 初始状态
宿主机根 (/)
└── 容器root目录 (/var/lib/minidocker/rootfs)

# 执行绑定挂载后
宿主机根 (/)
└── 容器root目录 [独立挂载点] (/var/lib/minidocker/rootfs)

# 执行pivot_root后
新根 (容器root目录)
└── .pivot_root (挂载旧根)

# 清理后
新根完全独立，旧根卸载
*/
func pivotRoot(root string) error {
	// 检查路径是否存在
	if b, _ := PathExists(root); b {
		return fmt.Errorf("路径 %s 不存在", root)
	}

	// bind mount 绑定挂载将相同的内容换一个挂载点，通过 bind mount将 root 重新挂载一次，即创建一个新的挂载点副本，使当前root 的老root和新root不在同一个文件系统下
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount rootfs to itsellf error: %v", err)
	}
	if b, _ := isMountPoint(root); !b {
		return fmt.Errorf("%s 不是一个挂载点", root)
	}

	// create 'rootfs/.pivot_root' to store old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	logrus.Infof("pivotDir: %v", pivotDir)
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return fmt.Errorf("make pivotDir fails: %v", err)
	}

	// pivot_root 改变根文件系统到新的rootfs，老的rootfs现挂载在rootfs/.pivot_root上
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root: %v", err)
	}
	// change current work dir to root dir, make the after operate base on new rootfs
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir: %v", err)
	}

	pivotDir = filepath.Join("/", ".pivot_root")
	// umount rootfs/.rootfs_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir: %v", err)
	}
	// 删除临时文件夹
	return os.Remove(pivotDir)
}

func readCommand() []string {
	// 在新建进程时除了3个标准io操作，将管道作为额外的第四个文件传入，因此管道的fd为3\
	// 如果父进程没有传入数据则会阻塞等待
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := io.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("read pipe fails: %v", err)
		return nil
	}
	return strings.Split(string(msg), " ")
}
