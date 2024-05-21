## HTTP文件服务器

本项目是使用go语言和vue框架实现的http文件服务器，为的是内网快速分享文件。为了实现需求，自然是越简单易用越好！本项目可使用`Docker`
进行部署！<br/>
目前实现功能：**文件夹打包下载**、**文件下载**、**文件上传**、**目录浏览**、**验证功能**。<br/>
待实现的功能：**文件预览**、**手机端适配**、**二维码生成**、**文件搜索**。

### 安装依赖

```shell
go mod download
```

### 运行程序

linux :

```shell
export "FILE_PATH=you_file_path"
go run main.go
```

windows :

```shell
set "FILE_PATH=you_file_path"
go run main.go
```

### 使用Docker

拉取镜像 :

```shell
docker pull mokibox/moki-http-server
```

启动程序 :

```shell
docker run -itd -p 8800:8800 -name http-server \
-e FILE_PATH=/data \
-e AUTH_CODE=you_auth_code \
-e TITLE=you_server_title \
-e IS_UPLOAD=false \
-e IS_DELETE=false \
-e IS_MKDIR=false \
-e SHOW_HIDDEN=true \
-e SHOW_DIR_SIZE \
-v you_file_path:/data \
--restart always \
mokibox/moki-http-server
```

### 使用Docker-compose

新建`docker-compose.yml`文件，复制如下内容 :

```markdown
version: '3.8'

services:
http-server:
image: mokibox/moki-http-server
container_name: http-server
ports:

- "8800:8800"
  environment:
  FILE_PATH: /data
  AUTH_CODE: you_auth_code
  TITLE: you_server_title
  IS_UPLOAD: "false"
  IS_DELETE: "false"
  IS_MKDIR: "false"
  SHOW_HIDDEN: "true"
  SHOW_DIR_SIZE: "true"
  volumes:
- you_file_path:/data
  restart: always
```

运行如下命令 :

```shell
docker compose up -itd
```

### 参数介绍

程序中环境变量参数及其含义:

|      参数名      |    含义     |  默认值  | 是否必填 |
|:-------------:|:---------:|:-----:|:----:|
|   FILE_PATH   |   文件路径    |   无   |  是   |
|   AUTH_CODE   |   验证密码    |   无   |  否   |
|     TITLE     |   服务标题    |   无   |  否   |
|   IS_UPLOAD   |  是否开启上传   | false |  否   |
|   IS_DELETE   |  是否开启删除   | false |  否   |
|   IS_MKDIR    | 是否开启创建文件夹 | false |  否   |
|  SHOW_HIDDEN  | 是否显示隐藏文件  | true  |  否   |
| SHOW_DIR_SIZE | 是否计算文件夹大小 | false |  否   |

### 接口介绍

如果你不满意前端,可以使用自己的前端，以下是接口列表及返回的错误码 :

| 错误码  |  含义   |
|:----:|:-----:|
| `-1` | 常规错误  |
| `-2` | 功能性错误 |
| `-3` | 验证性错误 |

|      接口      |    含义    |
|:------------:|:--------:|
|   `/query`   |  查询文件信息  |
|  `/delete`   |  删除文件接口  |
|  `/upload`   |  上传文件接口  |
| `/createDir` | 创建文件夹接口  |
|   `/base`    | 查询基本信息接口 |

### 程序截图

![image-20240521161449364](https://pic.mokibox.cn/pic/2024/05/21/664c57fe2311e.png)
![image-20240521160951514](https://pic.mokibox.cn/pic/2024/05/21/664c56d6e12d7.png)
![image-20240521161409529](https://pic.mokibox.cn/pic/2024/05/21/664c57d77520b.png)
![image-20240521161529060](https://pic.mokibox.cn/pic/2024/05/21/664c582689c53.png)

