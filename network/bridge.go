package network

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
	"time"
)

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

// Create 创建bridge网络
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip

	// 初始化网络对象
	n := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  d.Name(),
	}
	// 配置linux bridge
	err := d.initBridge(n)
	if err != nil {
		logrus.Errorf("init bridge fails: %v", err)
	}
	// 返回配置好的网络
	return n, err
}

// 初始化Linux Bridge
func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	// 创建Bridge网络
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("add bridge fails: %v", err)
	}

	// 设置Bridge设备的IP地址和路由
	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP
	logrus.Infof("gatewayIP: %v", gatewayIP)
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("assing ip address: %s on bridge: %s fails: %v", gatewayIP, bridgeName, err)
	}

	// 启动Bridge设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("set bridge: %s up fails: %v", bridgeName, err)
	}

	// 设置 iptables 的 SNAT 规则
	if err := setIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("set iptables for: %v falis: %s", bridgeName, err)
	}
	return nil
}

// Delete 删除网络
func (d *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name // 网络名即设备名
	// 得到网络对应的设备
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	// 删除对应的Bridge设备
	return netlink.LinkDel(br)
}

// Connect 创建Veth，并将其中一端挂载到Bridge设备上
func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	// 获取接口名和接口对象
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	// 1. 创建Veth接口并配置
	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]         // 因为Linux接口名有限制，只取endpoint ID前5位
	la.MasterIndex = br.Attrs().Index // 通过设置Veth的master属性，使其一端挂载到网络对应的Bridge上

	// 2. 创建Veth对象，通过PeerName设置另一端的名字为cif-{endpoint id 前五位}
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

	// 3. 通过LinkAdd方法创建该Veth接口
	if err := netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("add endpoint device fails: %v", err)
	}

	// 4. 设置启动该Veth
	if err := netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("set endpint device up fails: %v", err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

// 创建Linux Bridge 设备
func createBridgeInterface(bridgeName string) error {
	// 检查是否存在同名的Bridge设备， 若存在或报错则返回错误
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// 初始化一个netlink的Link基础对象，Lin名字即为Bridge设备名
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

	// 使用创建的Link属性创建netlink的bridge对象
	br := &netlink.Bridge{LinkAttrs: la}
	// 调用netlink的Linkadd方法创还能bridge设备，相当于 ip link add xxx
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("create Bridge: %v fails: %v", bridgeName, err)
	}
	return nil
}

// 设置网络接口的IP地址，例如setInterfaceIp("testBridge", "192.168.0.1/24")
func setInterfaceIP(name string, rawIP string) error {
	retries := 2
	var iface netlink.Link
	var err error
	// 设置两次重置
	for i := 0; i < retries; i++ {
		// 找到需要设置的网络接口
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		logrus.Infof("error retrieving new bridge netlink link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot the error: %v", err)
	}
	fmt.Printf("rawip: %s", rawIP)
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		logrus.Errorf("ParseIPNet ip: %d fails: %s", rawIP, err)
		return err
	}
	addr := &netlink.Addr{IPNet: ipNet, Peer: ipNet, Label: "", Flags: 0, Scope: 0, Broadcast: nil}
	return netlink.AddrAdd(iface, addr)
}

// 启动网络接口设备
func setInterfaceUP(interfaceName string) error {
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("get interface %s fails: %v", interfaceName, err)
	}

	// 通过netlink的LinkSetUp方法设置接口状态为UP，相当于 ip link set xxx up 命令
	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("enabling bridge interface %s fails: %v", interfaceName, err)
	}
	return nil
}

// 设置Linux iptables 对应bridge的 MASQUERADE 规则,使从该网桥出来的网络包都能进行源ip地址的转换
func setIPTables(bridgeName string, subnet *net.IPNet) error {
	// Golang中没有直接操控iptables的库，需要通过命令的方式来配置
	// iptables -t nat -A POSTROUTING -s <bridgeName> ! -o <bridgeName> -j MASQUERADE
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	output, err := cmd.Output()
	if err != nil {
		logrus.Errorf("iptables Output, %v", output)
	}
	return err
}
