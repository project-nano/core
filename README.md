# Nano Core

[[版本历史/ChangeLog](CHANGELOG.md)]

[English Version](#introduce)

### 简介

Core模块是Nano集群的主控节点，将Cell节点的计算资源组成虚拟化资源池，在节点之间调度管理云主机实例。

Core模块所有功能均可以通过API方式进行调用，便于用户集成到内部系统中。

由于涉及网络配置，建议使用专用Installer进行部署，项目最新版本请访问[此地址](https://github.com/project-nano/releases)

[项目官网](https://nanos.cloud/)

[项目全部源代码](https://github.com/project-nano)

### 编译

环境要求

- CentOS 7 x86
- Golang 1.20

```
准备依赖的framework
$git clone https://github.com/project-nano/framework.git

准备编译源代码
$git clone https://github.com/project-nano/core.git

编译
$cd core
$go build
```

编译成功在当前目录生成二进制文件core

### 使用

环境要求

- CentOS 7 x86

```
执行以下指令，启动Core模块
$./core start

也可以使用绝对地址调用或者写入开机启动脚本，比如
$/opt/nano/core/core start

```

模块运行日志输出在log/core.log文件中，用于查错和调试

**由于Cell节点依赖Core模块进行自动网络识别，所以集群工作时必须最先启动Core模块，再启动其他Cell节点。**

此外，除了模块启动功能，Core还支持以下命令参数启动

| 命令名 | 说明                               |
| ------ | ---------------------------------- |
| start  | 启动服务                           |
| stop   | 停止服务                           |
| status | 检查当前服务状态                   |
| halt   | 强行中止服务（用于服务异常时重启） |



### 配置

Core模块配置信息存放在config路径文件中，修改后需要重启模块生效

#### 域通讯配置

文件config/domain.cfg管理Core模块的域通讯信息

| 参数               | 值类型 | 默认值      | 必填 | 说明                                  |
| ------------------ | ------ | ----------- | ---- | ------------------------------------- |
| **domain**         | 字符串 | nano        | 是   | 通讯域名称，用于节点间识别            |
| **group_address**  | 字符串 | 224.0.0.226 | 是   | 通讯域组播地址，用于服务发现          |
| **group_port**     | 整数   | 5599        | 是   | 通讯域组播端口，用于服务发现          |
| **listen_address** | 字符串 |             | 是   | Core模块的主机监听地址，提供API等服务 |
| **timeout**        | 整数   | 10          |      | 交易处理超时时间，单位：秒            |

假设Core模块工作地址为192.168.1.31，示例配置文件如下

```json
{
 "domain": "nano",
 "group_address": "224.0.0.226",
 "group_port": 5599,
 "listen_address": "192.168.1.31"
}
```



#### API配置

文件config/api.cfg管理Core模块的应用接口

| 参数                | 值类型   | 默认值 | 必填 | 说明                                       |
| ------------------- | -------- | ------ | ---- | ------------------------------------------ |
| **port**            | 整数     | 5850   | 是   | API监听端口                                |
| **credentials**     | 对象数组 |        | 是   | 允许调用的API身份                          |
| **credentials.id**  | 字符串   |        | 是   | 身份校验的请求标识ID，对应FrontEnd的api_id |
| **credentials.key** | 字符串   |        | 是   | 身份校验的秘钥内容，对应FrontEnd的api_key  |

示例配置文件如下

```json
{
 "port": 5850,
 "credentials": [
  {
   "id": "dummyID",
   "key": "ThisIsAKeyPlaceHolder_ChangeToYourContent"
  }
 ]
}
```



#### 镜像服务配置

文件config/image.cfg管理Core模块的镜像服务

| 参数          | 值类型 | 默认值 | 必填 | 说明                    |
| ------------- | ------ | ------ | ---- | ----------------------- |
| **cert_file** | 字符串 |        | 是   | 镜像服务TLS传输证书文件 |
| **key_file**  | 字符串 |        | 是   | 镜像服务TLS传输秘钥文件 |

示例配置文件如下

```json
{
 "cert_file": "/opt/nano/core/cert/nano_image.crt.pem",
 "key_file": "/opt/nano/core/cert/nano_image.key.pem"
}
```



### 目录结构

模块主要目录和文件如下

| 目录/文件 | 说明                 |
| --------- | -------------------- |
| core      | 模块二进制执行文件   |
| cert/     | TLS证书存储目录      |
| config/   | 配置文件存储目录     |
| data/     | 模块运行数据存储目录 |
| log/      | 运行日志存储目录     |



# README

### Introduce

Core is control center of Nano cluster, which groups the computing resources of Cell nodes into a virtualized resource pool, and schedules and manages instances among the nodes.

A application could call all functions of Core via API.

It is recommended to use a dedicated Installer for deployment. For the latest project version, please visit [this address](https://github.com/project-nano/releases).

[Official Project Website](https://us.nanos.cloud/en/)

[Full Source Code of the Project](https://github.com/project-nano)

### Compilation

Environment requirements

- CentOS 7 x86
- Golang 1.20

```bash
Prepare the framework dependencies
$git clone https://github.com/project-nano/framework.git

Prepare the source code for compilation
$git clone https://github.com/project-nano/core.git

Compile
$cd core
$go build
    
```

The binary file "core" will be generated in the current directory when success

### Usage

Environment

- CentOS 7 x86

```bash
start module
$./core start

Alternatively, you can use an absolute address or write it into a startup script, such as:
$/opt/nano/core/core start
    
```

The log file core.log is output on the log/core.log file.

**Since the Cell nodes depend on the Core module for automatic network recognition, you must start the Core module first, and then start other Cell nodes.**

Core also supports the following command

| Command name | Explanation                               |
| ------------ | ----------------------------------------- |
| start        | Start service                             |
| stop         | Stop service                              |
| status       | Check current service status              |
| halt         | Force abort service when exception occurs |

## Configure

All configure files stores in the path: config. The module needs to be restarted before the changes take effect.

### Domain Configuration

Filename: domain.cfg

See more detail about [Domain](<https://nanocloud.readthedocs.io/projects/guide/en/latest/concept.html#communicate-domain>)

| Parameter          | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| **domain**         | The name of a communication domain, like 'nano' in default, only allows characters. |
| **group_port**     | Multicast port, 5599 in default                              |
| **group_address**  | Multicast address, '224.0.0.226' in default.                 |
| **listen_address** | Listening Address of the core service，string in the IPv4 format |

### Configuration

Core module configuration information is stored in files under the config path, and modifications require a restart of the module to take effect.

#### Domain Communication Configuration

The file `config/domain.cfg` manages the domain communication information for the Core module.

| Parameter          | Value Type | Default Value | Required | Explanation                                                  |
| ------------------ | ---------- | ------------- | -------- | ------------------------------------------------------------ |
| **domain**         | String     | nano          | Yes      | The name of the communication domain, used for cluster identification |
| **group_address**  | String     | 224.0.0.226   | Yes      | Multicast address of the communication domain, used for service discovery |
| **group_port**     | Integer    | 5599          | Yes      | Multicast port of the communication domain, used for service discovery |
| **listen_address** | String     |               | Yes      | Listening Address of the core service，string in the IPv4 format |
| **timeout**        | Integer    | 10            |          | Transaction timeout in seconds                               |

Assuming the working address of the Core module is 192.168.1.31, an example configuration file is as follows:

```json
{
 "domain": "nano",
 "group_address": "224.0.0.226",
 "group_port": 5599,
 "listen_address": "192.168.1.31"
}
```

### API Configuration

The file `config/api.cfg` manages the API service

| Parameter           | Value Type   | Default Value | Required | Explanation                                                  |
| ------------------- | ------------ | ------------- | -------- | ------------------------------------------------------------ |
| **port**            | Integer      | 5850          | Yes      | The API listening port                                       |
| **credentials**     | Array Object |               | Yes      | Allowed API identities                                       |
| **credentials.id**  | String       |               | Yes      | The request ID for identity verification, corresponding to FrontEnd's api_id |
| **credentials.key** | String       |               | Yes      | The secret key content for identity verification, corresponding to FrontEnd's api_key |

An example configuration file is as follows:

```json
{
 "port": 5850,
 "credentials": [
  {
   "id": "dummyID",
   "key": "ThisIsAKeyPlaceHolder_ChangeToYourContent"
  }
 ]
}
```

### Image Service Configuration

The file `config/image.cfg` manages the image service of the Core module.

| Parameter     | Value Type | Default Value | Required | Explanation                                |
| ------------- | ---------- | ------------- | -------- | ------------------------------------------ |
| **cert_file** | String     |               | Yes      | TLS certificate file for the image service |
| **key_file**  | String     |               | Yes      | TLS secret key file for the image service  |

An example configuration file is as follows:

```json
{
 "cert_file": "/opt/nano/core/cert/nano_image.crt.pem",
 "key_file": "/opt/nano/core/cert/nano_image.key.pem"
}
```



### Directory Structure

| Directory/File | Explanation                                |
| -------------- | ------------------------------------------ |
| core           | The binary execution file of the module    |
| cert/          | The storage directory for TLS certificates |
| config/        | The storage directory for configurations   |
| data/          | The storage directory for operation data   |
| log/           | The storage directory for logs             |
