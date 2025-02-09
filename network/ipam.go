package network

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net"
	"os"
	"path"
	"strings"
)

const ipamDefaultAllocatorPath = "/var/run/minidocker/network/ipam/subnet.json"

// IPAM 网络ip地址分配结构体，使用位图算法，为方便实现使用string中的一个字符表示一个状态位
type IPAM struct {
	SubnetAllocatorPath string             // 分配文件存放位置
	Subnets             *map[string]string // 网段和位图算法的数组map，key是网段，value是分配位图数组
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// 加载网段地址分配信息
func (ipam *IPAM) load() error {
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	// 打开读取网段配置信息文件
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	if err != nil {
		return nil
	}
	defer subnetConfigFile.Close()
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}
	// unmarshal
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		logrus.Errorf("load allocation info fails: %v", err)
		return err
	}
	return nil
}

// 存储网段地址分配信息
func (ipam *IPAM) dump() error {
	ipamConfigDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(ipamConfigDir, 0644)
		} else {
			return err
		}
	}
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("openfile %s fails: %v", ipam.SubnetAllocatorPath, err)
		return err
	}
	defer subnetConfigFile.Close()
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return err
	}
	return nil
}

// Allocate 从网段中分配一个可用ip地址,并保存到文件中
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	ipam.Subnets = &map[string]string{}
	if err = ipam.load(); err != nil {
		logrus.Errorf("load ipam fails: %v", err)
		//return
	}
	logrus.Infof("subnet: %s", subnet.String())
	one, size := subnet.Mask.Size()

	// 若之前未分配过该网段，则初始化网段的分配配置
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		// 用0填满网段的配置，1<<uint8(size-one)表示这个网段中有多少个可用地址
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}

	// 遍历网段的位图数组
	for c := range (*ipam.Subnets)[subnet.String()] {
		// 找到为了"0"的项和数组序号，即可以分配ip
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			// 将该项设'1'，即分配该ip
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			// 设置初始ip，即主机号全0
			ip = subnet.IP

			// 通过位图的项偏移算出分配的IP地址
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			ip[3] += 1 // 跳过子网地址
			break
		}
	}
	// 将分配结果存储到文件
	if err := ipam.dump(); err != nil {
		logrus.Errorf("dump ipam fails: %v", err)
	}
	return
}

// Release 释放ip地址，并保存回配置文件
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}
	_, subnet, _ = net.ParseCIDR(subnet.String())
	err := ipam.load()
	if err != nil {
		logrus.Errorf("load allocation info %s fails: %v", ipam.SubnetAllocatorPath, err)
	}

	// 计算ip在网段位图数组中的索引位置
	c := 0
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}
	// 将分配的位图数组索引处置0
	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	logrus.Infof("c: %d, len(ipalloc)=%d", c, len(ipalloc))
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)
	// 将分配结果存储到文件
	if err := ipam.dump(); err != nil {
		logrus.Errorf("dump ipam fails: %v", err)
	}
	return nil
}
