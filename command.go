package main

import (
	"MiniDocker/cgroups/subsystem"
	"MiniDocker/container"
	"MiniDocker/dockerCommand"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var runCommand = cli.Command{
	Name:  "run",
	Usage: "Create a container | miniDocker run -it [command]",
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
		// 启动函数
		dockerCommand.Run(createTTY, containerCmd, &resourceConfig, volume, containerName)

		return nil
	},
}

// 该command不面向用户，值只协助runCommand
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
	Usage: "commit a container into image",
	Action: func(context *cli.Context) error {
		if context.Args().Len() < 1 {
			return fmt.Errorf("missing container name, use: commit [imageName]")
		}
		imageName := context.Args().Get(0)
		dockerCommand.CommitContainer(imageName)
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
