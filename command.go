package main

import (
	"MiniDocker/cgroups/subsystem"
	"MiniDocker/container"
	"MiniDocker/dockerCommand"
	"MiniDocker/network"
	"errors"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var runCommand = cli.Command{
	Name:  "run",
	Usage: "Create a container | miniDocker run [args] [image] [command]",
	Flags: []cli.Flag{
		// 整合i和t, 交互式运行
		&cli.BoolFlag{
			Name:  "it",
			Usage: "open an interactive tty(pseudo terminal)", // 打开交互式tty
		},
		// 后台运行
		&cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		// 限制内存占用
		&cli.StringFlag{
			Name:  "m",
			Usage: "limit the memory",
		},
		// 限制CPU核心数
		&cli.StringFlag{
			Name:  "cpu",
			Usage: "limit the cpu amount",
		},
		// 限制CPU时间片权重
		&cli.StringFlag{
			Name:  "cpushare",
			Usage: "limit the cpu share",
		},
		// 挂载数据卷
		&cli.StringFlag{
			Name:  "v",
			Usage: "set volume, user: -v [volumeDir]:[containerVolumeDir]",
		},
		// 指定容器名字
		&cli.StringFlag{
			Name:  "name",
			Usage: "set container name",
		},
		// 指定环境变量, 可指定多个
		&cli.StringSliceFlag{
			Name:  "e",
			Usage: "set environments",
		},
		// 设置网络
		&cli.StringFlag{
			Name:  "net",
			Usage: "set container network",
		},
		// 设置端口映射
		&cli.StringSliceFlag{
			Name:  "p",
			Usage: "set port mapping",
		},
	},
	/*
		run 命令执行的函数
		判断参数是否包含command	获取用户指定的command 调用Run function去准备容器
	*/
	Action: func(context *cli.Context) error {
		args := context.Args()
		if args.Len() == -1 {
			return errors.New("missing container command")
		}

		// 得到容器起始命令
		containerCmd := make([]string, args.Len())
		for index, cmd := range args.Slice() {
			containerCmd[index] = cmd
		}

		// check "-it" or "-d"
		createTTY := context.Bool("it")
		detach := context.Bool("d")
		// detach和createTTY不能共存
		if createTTY && detach {
			return fmt.Errorf("it and d paramter can not both provided")
		}
		logrus.Infof("createTTY %v", createTTY)

		// 得到资源配置
		resourceConfig := subsystem.ResourceConfig{
			MemoryLimit: context.String("m"),
			CPUShare:    context.String("cpushare"),
			CPUSet:      context.String("cpu"),
		}

		// 传递volume
		volume := context.String("v")
		// 容器名
		containerName := context.String("name")

		// get image name
		imageName := containerCmd[0]
		containerCmd = containerCmd[1:]

		// get environments
		envSlice := context.StringSlice("e")

		// get network and portMapping
		network := context.String("net")
		portmapping := context.StringSlice("p")

		// 启动函数
		dockerCommand.Run(createTTY, containerCmd, &resourceConfig, volume, containerName, imageName, envSlice, network, portmapping)

		return nil
	},
}

// 该command不面向用户，只协助runCommand
// docker init
var initCommand = cli.Command{
	Name:  "init",
	Usage: "init a container process run user's process in container. Do not call in outside",
	/*
		获取传递来的command参数
		执行容器的初始化操作
	*/
	Action: func(context *cli.Context) error {
		logrus.Infof("Start initating...")
		return container.InitProcess()
	},
}

// 打包容器形成镜像命令
var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image; commit [containerName] [imageName]",
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 1 {
			return fmt.Errorf("missing container name, use: commit [imageName]")
		}
		containerName := context.Args().Get(0)
		imageName := context.Args().Get(1)
		dockerCommand.CommitContainer(containerName, imageName)
		return nil
	},
}

// 查看所有容器信息命令
var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(context *cli.Context) error {
		dockerCommand.ListContainers()
		return nil
	},
}

// 查看指定容器的日志命令
var logCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs of container",
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 1 {
			return fmt.Errorf("please input container name")
		}
		containerName := context.Args().Get(0)
		dockerCommand.LogContainer(containerName)
		return nil
	},
}

// exec，进入容器命令, mydocker exec {containerName} {containerCmd}
var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {
		// for callback, 若环境变量不为空即第二次执行，直接返回以免重复调用
		if os.Getenv(dockerCommand.ENV_EXEC_PID) != "" {
			logrus.Infof("pid callback pid %v", os.Getpid())
			return nil
		}
		if context.Args().Len() < 2 {
			return fmt.Errorf("missing container name or command")
		}
		containerName := context.Args().Get(0)
		var containerCmds []string
		for _, arg := range context.Args().Tail() {
			containerCmds = append(containerCmds, arg)
		}

		dockerCommand.ExecContainer(containerName, containerCmds)
		return nil
	},
}

// 停止容器命令
var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		dockerCommand.StopContainer(containerName)
		return nil
	},
}

// 删除容器命令
var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove a container",
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		dockerCommand.RemoveContainer(containerName)
		return nil
	},
}

// 容器网络相关命令
var networkCommand = cli.Command{
	Name:  "network",
	Usage: "container network commands",
	Subcommands: []*cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "driver",
					Usage: "network driver",
				},
				// 子网网段
				&cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr",
				},
			},
			Action: func(context *cli.Context) error {
				if context.Args().Len() < 1 {
					return fmt.Errorf("missing network name")
				}
				network.Init()
				driver, subnet, networkName := context.String("driver"), context.String("subnet"), context.Args().Get(0)
				err := network.CreateNetwork(driver, subnet, networkName)
				if err != nil {
					return fmt.Errorf("create network %s fails: %v", networkName, err)
				}
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "list container networks",
			Action: func(context *cli.Context) error {
				_ = network.Init()
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "rm",
			Usage: "remove container network",
			Action: func(context *cli.Context) error {
				if context.Args().Len() < 1 {
					return fmt.Errorf("missing the container network name")
				}
				networkName := context.Args().Get(0)
				network.Init()
				err := network.DeleteNetwork(networkName)
				if err != nil {
					return fmt.Errorf("remove network %s fails: %v", networkName, err)
				}
				return nil
			},
		},
	},
}
