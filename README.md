
- [My Mini Docker](#my-mini-docker)
- [简介](#简介)
- [主要功能](#主要功能)
- [使用](#使用)
    - [演示Demo](#demo)
    - [命令列表](#命令列表)



# My Mini Docker
    开发环境：Ubuntu22.04, Golang1.23.2
## 简介
My Mini Docker 是一个仿制 Docker 的个人 Golang 项目，实现了容器的基本功能，包括镜像管理、容器隔离、资源控制和网络通信。该项目支持大部分常用的 Docker 命令，并兼容 Docker 镜像的导入与运行。
## 主要功能
- 通过`Linux Namespace`实现容器间的基础隔离和资源控制。
- 借助`Linux Cgroups`完成对容器资源的控制。
- 通过`OverlayFS`代替原本的`AUFS`实现容器文件系统的隔离，从而更好地适配高版本 Linux 内核并确保其正常运行。
- 实现容器的镜像打包功能，并支持兼容 Docker 镜像的导入与运行。
- 通过Linux虚拟网络设备`Veth`和`Bridge`构建容器网络系统，实现容器与主机、容器与容器、容器与外界的网络通信。
## 使用
    镜像文件默认存放在/root/下，需运行的镜像同样需存放在/root/，推荐使用Docker导出的镜像文件运行。
### Demo
运行容器:
`MiniDocker run [args] [imageName] [commands]`

查看容器列表：
`MiniDocker ps`

停止容器/删除容器：
`MiniDocker stop [containerName]`/`MiniDocker rm [containerName]`

打包镜像(默认存放于/root/)：
`MiniDocker commit [containerName] [imageName]`

查看后台容器日志：
`MiniDocker logs [containerName]`

创建容器网络：
`MiniDocker network create --driver [驱动名] --subnet [子网网段] [网络名]`

查看网络：
`MiniDocker network list`

删除网络：
`MiniDocker network rm [networkName]`



### 命令列表
总体命令：
```
NAME:
   miniDocker - Usage

USAGE:
   miniDocker [global options] command [command options]

COMMANDS:
   run      Create a container | miniDocker run [args] [image] [command]
   init     init a container process run user's process in container. Do not call in outside
   commit   commit a container into image; commit [containerName] [imageName]
   ps       list all the containers
   logs     print logs of container
   exec     exec a command into container
   stop     stop a container
   rm       remove a container
   network  container network commands
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```

run命令：
```
NAME:
   miniDocker run - Create a container | miniDocker run [args] [image] [command]

USAGE:
   miniDocker run [command options]

OPTIONS:
   --it                   open an interactive tty(pseudo terminal) (default: false)
   -d                     detach container (default: false)
   -m value               limit the memory
   --cpu value            limit the cpu amount
   --cpushare value       limit the cpu share
   -v value               set volume, user: -v [volumeDir]:[containerVolumeDir]
   --name value           set container name
   -e value [ -e value ]  set environments
   --net value            set container network
   -p value [ -p value ]  set port mapping
   --help, -h             show help
```

网络相关命令：
```
NAME:
   miniDocker network - container network commands

USAGE:
   miniDocker network command [command options]

COMMANDS:
   create   create a container network
   list     list container networks
   rm       remove container network
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help
```