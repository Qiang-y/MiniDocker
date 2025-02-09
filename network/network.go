package network

import (
	"MiniDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/run/minidocker/network/network/" // 默认网络配置信息存储位置
	drivers            = map[string]NetworkDriver{}             // 驱动字典，存储驱动信息
	networks           = map[string]*Network{}                  // 网络字段，存储网络信息
)

// Network 抽象的网络数据结构
type Network struct {
	Name    string     // 网络名
	IpRange *net.IPNet // 地址段
	Driver  string     // 网络驱动名
}

// Endpoint 网络端点，用于链接容器与网络，保证容器内部与网络的通信，包括地址，veth设备、端口映射、链接的容器和网络等信息
type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	PortMapping []string         `json:"port_mapping"`
	Network     *Network
}

// NetworkDriver 网络驱动接口
type NetworkDriver interface {
	// Name 驱动名
	Name() string
	// Create 创建网络
	Create(subnet string, name string) (*Network, error)
	// Delete 删除网络
	Delete(network Network) error
	// Connect 链接容器网络端点到网络
	Connect(network *Network, endpoint *Endpoint) error
	// Disconnect 从网络上移除容器网络端点
	Disconnect(network Network, endpoint *Endpoint) error
}

// CreateNetwork 创建网络
func CreateNetwork(driver, subnet, name string) error {
	// 将网段解释成net.IPNet对象
	_, cidr, _ := net.ParseCIDR(subnet)
	// 通过IPAM分配网关ip，获取网段中的第一个ip作为网关的ip
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIp

	// 调用指定的网络驱动创建网络，divers中存有各个网络驱动的实例，通过调用网络驱动的Create方法
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return nil
	}
	// 保存网络信息值文件系统，方便查询和在网络上链接网络端点
	return nw.dump(defaultNetworkPath)
}

// 将网络的配置信息保存在文件系统中
func (nw *Network) dump(dumpPath string) error {
	// check the file path and create if not exit
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}
	// 保存文件的名字是网络名
	nwPath := filepath.Join(dumpPath, nw.Name)
	// 打开文件，参数为：存在内容则清空，值写入，不存在则创建
	newFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("open file %v fails: %v", nwPath, err)
		return err
	}
	defer newFile.Close()

	// 以json格式保存
	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("json marshal nw fails: %v", err)
		return err
	}
	if _, err := newFile.Write(nwJson); err != nil {
		logrus.Errorf("write file %v fails: %v", nwPath, err)
		return err
	}
	return nil
}

// 从文件中读取网络配置
func (nw *Network) load(loadPath string) error {
	// open the config file
	nwConfigFile, err := os.Open(loadPath)
	if err != nil {
		logrus.Errorf("open config file %v fails: %v", loadPath, err)
		return err
	}
	defer nwConfigFile.Close()
	// get json from config file
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}
	// unmarshal json to get config
	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		logrus.Errorf("load nw info fails: %v", err)
		return err
	}
	return nil
}

// 从网络配置目录中删除网络的配置文件
func (nw *Network) remove(dumpPath string) error {
	nwPath := path.Join(dumpPath, nw.Name)
	if _, err := os.Stat(nwPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	return os.Remove(nwPath)
}

// Init 初始化，从网络配置目录中加载所有网络配置信息到networks字典
func Init() error {
	// 加载网络驱动(bridge)
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver
	// 验证网络配置目录
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}

	// 遍历检查网络配置目录中的所有文件
	filepath.Walk(defaultNetworkPath, func(nwPath string, info fs.FileInfo, err error) error {
		// 如果是目录则跳过
		if info.IsDir() {
			return nil
		}

		// 加载文件名作为网络
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}
		// 加载网络配置信息
		if err := nw.load(nwPath); err != nil {
			logrus.Errorf("load network %s fails: %v", nwName, err)
		}
		logrus.Infof("load NW: %v", nw)
		networks[nwName] = nw
		return nil
	})

	return nil
}

// ListNetwork 显示网络列表
func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, _ = fmt.Fprintf(w, "Name\tIpRange\tDriver\n")
	// 遍历网络信息
	for _, nw := range networks {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", nw.Name, nw.IpRange.String(), nw.Driver)
	}
	if err := w.Flush(); err != nil {
		logrus.Errorf("flush error: %v", err)
		return
	}
}

// Connect 创建容器并连接网络
func Connect(networkName string, cinfo *container.ContainerInfo) error {
	// 获取网络信息
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network: %s", networkName)
	}
	logrus.Infof("ip: %s", network.IpRange.String())
	_, cidr, _ := net.ParseCIDR(network.IpRange.String())
	logrus.Infof("cidr: %v", cidr)
	// 通过IPAM从网络的网段中获取可用的ip作为容器ip
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	logrus.Infof("allocate ip: %v", ip)
	// 创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, network.Name),
		IPAddress:   ip,
		Network:     network,
		PortMapping: cinfo.PortMapping,
	}
	logrus.Infof(" networK name: %v, ip: %v, mask: %v", network.Name, network.IpRange.String(), network.IpRange.Mask)
	// 连接网络
	if err := drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}
	// 进入到容器网络namespace内配置网络容器设备的ip和路由
	if err := configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}
	// 配置容器到宿主机的端口映射
	return configPortmapping(ep, cinfo)
}

// DeleteNetwork 删除网络
func DeleteNetwork(networkName string) error {
	// 查找网络
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network: %s", networkName)
	}
	// 释放ip
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("remove network gateway ip fails: %s", err)
	}
	// 调用网络驱动删除网络创建的设备与配置
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("remove network driver fails: %s", err)
	}
	// 删除对应的网络配置文件
	return nw.remove(defaultNetworkPath)
}

// 进入容器对应的net namespace配置Veth另一端的IP和路由
func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	// 找到 Veth 的另一端
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("get Veth endpoint: %v fails: %v", ep.Device.PeerName, err)
	}

	// 将Veth的该端点加入该net namespace
	// defer延迟执行的是enterContainerNetns返回的函数，而enter函数此处就执行了，因此该语句以下操作都是在容器网络空间进行
	defer enterContainerNetns(&peerLink, cinfo)()

	// 此时已进入到容器的网络空间中
	// 找到容器的IP地址及网段，若ip为192.128.0.1，网段为192.128.0.0/24，则产出192.128.0.1/24
	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress
	logrus.Infof("interfaceIP: %d , mask: %d", interfaceIP.IP, interfaceIP.Mask)
	// 设置容器NS内的Veth端点的IP
	logrus.Infof("intetfaceIP: %v", interfaceIP.String())
	if err := setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v, %v", ep.Network, err)
	}

	// 启动容器内的Veth端点
	if err := setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}
	// net namespace 中的默认回环地址127.0.0.1是关闭的，设置开启
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	// 0.0.0.0/0 网段表示所有ip地址段
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	// 创建路由规则，表示所有的网段（外部请求）都通过容器内的Veth端点访问
	// 相当于命令： route add -net 0.0.0.0/0 gw <Bridge 网桥地址> dev <容器内Veth端点设备>
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}
	// 调用RouteAdd将路由规则添加到容器的netns
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

// 进入容器的Net Namespace，将容器的网络端点加入该网络空间
// 锁定当前程序执行的线程，是当前线程进入到容器的网络空间
// @return 返回一个函数指针，执行该返回函数才会退出容器的网络空间，回到宿主机的网络空间
func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	// 找到容器的Net Namespace
	// 通过/proc/[pid]/ns/net 文件来操作Net Namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("open container net file fails: %v", err)
	}
	nsFD := f.Fd() // 得到文件描述符

	// 锁定当前程序执行的线程，如果不锁定操作系统线程的话，golang的goroutine可能会被调度到别的现场上，就不能保证一直在所需要的网络空间中了
	runtime.LockOSThread()

	// 修改网络端点Veth的另一端，将其移动到容器的Net Namespace
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("set Veth link Netns fails: %v", err)
	}

	// 获得当前进程（其实精确到线程）网络的Net Namespace，以便后面从容器的网络空间中退出回到原本的网络空间
	origins, err := netns.Get()

	// 将当前进程加入到容器的网络空间, 底层是调用了linux的setns系统调用
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("set netns fails: %v", err)
	}

	// 通过回调返回原本的Net Namespace，在容器的网络空间中执行完容器配置后会调用该回调函数返回原本的网络空间
	return func() {
		netns.Set(origins)
		origins.Close()
		runtime.UnlockOSThread() // 取消goroutine对当前执行线程的锁定
		f.Close()
	}
}

// 通过iptables的DNAT规则配置主机到容器的端口映射
func configPortmapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	// 遍历容器端口映射表
	for _, pm := range ep.PortMapping {
		// 分割成主机端口和容器端口
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port mapping format eror: %v", pm)
			continue
		}

		// 调用iptables命令
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return nil
}

func Disconnect(networkName string, cinfo *container.ContainerInfo) error {
	//todo: implement here
	return nil
}
