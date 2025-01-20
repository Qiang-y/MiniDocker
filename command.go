package main

import (
	"MiniDocker/cgroups/subsystem"
	"MiniDocker/container"
	"MiniDocker/dockerCommand"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var runCommand = cli.Command{
	Name:  "run",
	Usage: "Create a container | miniDocker run -it [command]",
	Flags: []cli.Flag{
		// 整合i和t
		&cli.BoolFlag{
			Name:  "it",
			Usage: "open an interactive tty(pseudo terminal)", // 打开交互式tty
		},
		&cli.StringFlag{
			Name:  "m",
			Usage: "limit the memory",
		},
		&cli.StringFlag{
			Name:  "cpu",
			Usage: "limit the cpu amount",
		},
		&cli.StringFlag{
			Name:  "cpushare",
			Usage: "limit the cpu share",
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

		// check "-it"
		tty := context.Bool("it")

		// 得到资源配置
		resourceConfig := subsystem.ResourceConfig{
			MemoryLimit: context.String("m"),
			CPUShare:    context.String("cpushare"),
			CPUSet:      context.String("cpu"),
		}

		// 启动函数
		dockerCommand.Run(tty, containerCmd, &resourceConfig)

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
