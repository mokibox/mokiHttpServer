package src

import (
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	dir         = os.Getenv("FILE_PATH")               // 文件根路径
	title       = os.Getenv("TITLE")                   // 标题
	isUpload    = os.Getenv("IS_UPLOAD") == "true"     // 是否开启上传
	isDelete    = os.Getenv("IS_DELETE") == "true"     // 是否开启删除
	isMkdir     = os.Getenv("IS_MKDIR") == "true"      // 是否开启创建文件夹
	showHidden  = os.Getenv("SHOW_HIDDEN") == ""       // 是否默认显示隐藏文件
	showDirSize = os.Getenv("SHOW_DIR_SIZE") == "true" // 是否默认计算文件夹大小
)

// HandleBaseRequest 处理基本查询
func HandleBaseRequest(w http.ResponseWriter, r *http.Request) {
	// 验证
	cookie, err := r.Cookie("session_id")
	err = authToCookie(cookie)
	if err != nil {
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("基本查询发生异常，异常为: ", err)
		}
		return
	}

	// 组织返回数据
	resultData := make(map[string]any)
	if title == "" {
		title = "HTTP文件服务器"
	}
	resultData["title"] = title
	resultData["isUpload"] = isUpload
	resultData["isDelete"] = isDelete
	resultData["isMkdir"] = isMkdir
	resultData["showHidden"] = showHidden
	resultData["showDirSize"] = showDirSize

	// 返回基础设置数据
	_, err = w.Write(resultSuccess(resultData))
	if err != nil {
		logrus.Error("基本查询发生异常，异常为: ", err)
		return
	}
	logrus.Info("基本查询方法执行成功!")
}

// HandleLoginRequest 处理登录
func HandleLoginRequest(w http.ResponseWriter, r *http.Request) {
	// 获取参数
	query := r.URL.Query()
	password := query.Get("password")
	now := time.Now()

	// 验证
	cookie, err := authToCode(password, now)
	if err != nil {
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("登录发生异常，异常为: ", err)
		}
		return
	}

	// 写入cookie
	http.SetCookie(w, cookie)
	_, err = w.Write(resultSuccess(now.Unix()))
	if err != nil {
		logrus.Error("登录发生异常，异常为: ", err)
		return
	}
	logrus.Info("登录方法执行成功!")
}

// HandleQueryRequest 查询文件
func HandleQueryRequest(w http.ResponseWriter, r *http.Request) {
	// 验证
	cookie, err := r.Cookie("session_id")
	err = authToCookie(cookie)
	if err != nil {
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("文件查询发生异常, 异常为: ", err)
		}
		return
	}

	// 获取参数, 初始化目录
	localDir := strings.ReplaceAll(dir, "\\", "/")
	query := r.URL.Query()
	filePath := query.Get("filePath")
	localDir = filepath.Join(localDir, filePath)
	showHidden, _ = strconv.ParseBool(query.Get("showHidden"))   // 是否显示隐藏文件
	showDirSize, _ = strconv.ParseBool(query.Get("showDirSize")) // 是否计算文件夹大小

	// 打开目录，读入对象
	d, err := os.Open(localDir)
	if err != nil {
		logrus.Error("文件查询发生异常, 异常为: ", err)
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("文件查询发生异常, 异常为: ", err)
		}
		return
	}
	defer func(d *os.File) {
		err = d.Close()
		if err != nil {
			logrus.Error("文件查询发生异常, 异常为: ", err)
		}
	}(d)
	logrus.Info("文件查询, 目录读取成功!")

	// 读取当前目录下的文件和子目录
	files, err := d.ReadDir(0)
	if err != nil {
		logrus.Error("文件查询发生异常, 异常为: ", err)
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("文件查询发生异常, 异常为: ", err)
		}
		return
	}
	logrus.Info("文件查询, 文件及子目录读取成功!")

	// 组织返回的json
	var filesInfos []map[string]string
	for _, file := range files {
		var info os.FileInfo
		info, err = file.Info()
		if err != nil {
			logrus.Error("文件查询发生异常, 异常为: ", err)
			return
		}
		if !showHidden && len(info.Name()) > 0 && info.Name()[0] == '.' {
			continue
		}
		fileInfo := make(map[string]string)
		fileInfo["path"] = filePath
		fileInfo["name"] = info.Name()
		fileInfo["fileType"] = ""
		fileNameSplit := strings.Split(info.Name(), ".")
		if len(fileNameSplit) > 1 {
			fileInfo["fileType"] = fileNameSplit[len(fileNameSplit)-1]
		}
		fileInfo["size"] = formatSize(float64(info.Size()), 0)
		fileInfo["modified"] = info.ModTime().Format(time.DateTime)
		fileInfo["type"] = "file"
		if info.IsDir() {
			fileInfo["fileType"] = "folder"
			dirPath := filepath.Join(localDir, file.Name())
			fileInfo["type"] = "dir"
			fileInfo["size"] = ""
			if showDirSize {
				var size int64
				size = dirSize(dirPath)
				fileInfo["size"] = formatSize(float64(size), 0)
			}
		}
		filesInfos = append(filesInfos, fileInfo)
	}
	sort.Slice(filesInfos, func(i, j int) bool {
		return filesInfos[j]["type"] == "file"
	})
	logrus.Info("文件查询, 文件信息组织成功!")

	// 设置 Content-Type 为 application/json 并返回文件信息数据
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(resultSuccess(filesInfos))
	if err != nil {
		logrus.Error("文件查询发生异常, 异常为: ", err)
		return
	}
	logrus.Info("文件查询方法执行成功！")
}

// HandleDownloadRequest 下载文件
func HandleDownloadRequest(w http.ResponseWriter, r *http.Request) {
	// 获取参数
	query := r.URL.Path
	reqPthSplit := strings.Split(strings.TrimLeft(query, "/"), "/")

	// 切割出文件名并写入返回头里
	fileName := reqPthSplit[len(reqPthSplit)-1]
	reName := strings.ReplaceAll(fileName, "pkgDir_", "")
	if reName == ".zip" {
		reName = "root.zip"
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+reName)

	// 认证并处理路径
	pathCode := reqPthSplit[1]
	var pathList []string
	tag, msg := authToPath(pathCode)
	pathList = append(reqPthSplit[2 : len(reqPthSplit)-1]) // 第一个元素是url,最后一个是文件名
	if msg != "" {
		if !tag {
			logrus.Error("下载文件发生异常, 异常为: ", msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		pathList = append(reqPthSplit[2 : len(reqPthSplit)-1]) // 第二个元素是auth
	}
	filePath := path.Join(pathList...)
	localDir := strings.ReplaceAll(dir, "\\", "/")
	logrus.Info("下载文件, 路径处理完成!")

	// 处理文件夹的情况
	if strings.HasPrefix(fileName, "pkgDir_") {
		fileName = strings.ReplaceAll(fileName, "pkgDir_", "")
		dirPath := path.Join(localDir, filePath, strings.ReplaceAll(fileName, ".zip", ""))
		if fileName == ".zip" {
			fileName = "root" + fileName
		}
		createZipDir(dirPath, fileName)
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				logrus.Error("下载文件发生异常, 异常为: ", err)
			}
		}(fileName)

		// 打开 zip 文件
		file, err := os.Open(fileName)
		if err != nil {
			logrus.Error("下载文件发生异常, 异常为: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func(file *os.File) {
			err = file.Close()
			if err != nil {
				logrus.Error("下载文件发生异常, 异常为: ", err)
			}
		}(file)
		logrus.Info("下载文件, 文件夹ZIP生成成功!")

		// 设置响应头并将 zip 文件内容复制到响应体中
		w.Header().Set("Content-Type", "application/zip")
		_, err = io.Copy(w, file)
		if err != nil {
			logrus.Error("下载文件发生异常, 异常为: ", err)
		}
		logrus.Info("下载文件方法执行成功!")
		return
	}

	// 处理正常的文件
	downloadFile := filepath.Join(localDir, filePath, fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, downloadFile)
	logrus.Info("下载文件方法执行成功!")
}

// HandleRemoveRequest 删除文件
func HandleRemoveRequest(w http.ResponseWriter, r *http.Request) {
	// 认证
	cookie, err := r.Cookie("session_id")
	err = authToCookie(cookie)
	if err != nil {
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("删除文件发生异常, 异常为: ", err)
		}
		return
	}

	// 功能是否开启
	if !isDelete {
		logrus.Error("删除文件非法访问, 功能未开启!")
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(resultError(-2, errors.New("功能未开启! ")))
		if err != nil {
			logrus.Error("删除文件发生异常, 异常为: ", err)
		}
		return
	}

	// 获取参数并初始化目录
	localDir := strings.ReplaceAll(dir, "\\", "/")
	query := r.URL.Query()
	filePath := query.Get("filePath")
	fileName := query.Get("fileName")

	// 尝试删除文件
	err = os.Remove(path.Join(localDir, filePath, fileName))
	if err != nil {
		logrus.Error("删除文件发生异常, 异常为: ", err)
		_, err = w.Write(resultError(-1, errors.New(strings.ReplaceAll(err.Error(), dir, ""))))
		if err != nil {
			logrus.Error("删除文件发生异常, 异常为: ", err)
		}
		return
	}
	logrus.Info("删除文件, 文件删除成功!")

	// 设置 Content-Type 为 application/json 并返回成功的消息
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(resultSuccess(nil))
	if err != nil {
		logrus.Error("删除文件发生异常, 异常为: ", err)
		return
	}
	logrus.Info("删除文件方法执行成功!")
}

// HandleUploadRequest 上传文件
func HandleUploadRequest(w http.ResponseWriter, r *http.Request) {
	// 认证
	cookie, err := r.Cookie("session_id")
	err = authToCookie(cookie)
	if err != nil {
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("上传文件发生异常, 异常为: ", err)
		}
		return
	}
	// 功能是否开启
	if !isUpload {
		logrus.Error("上传文件非法访问, 功能未开启!")
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(resultError(-2, errors.New("功能未开启! ")))
		if err != nil {
			logrus.Error("上传文件发生异常，异常为: ", err)
		}
		return
	}

	if r.Method == "POST" {
		// 获取文件及参数
		filePath := r.FormValue("filePath")
		var (
			file    multipart.File
			handler *multipart.FileHeader
		)
		file, handler, err = r.FormFile("file")
		if err != nil {
			logrus.Error("上传文件发生异常, 异常为: ", err)
			return
		}
		defer func(file multipart.File) {
			err = file.Close()
			if err != nil {
				logrus.Error("上传文件发生异常, 异常为: ", err)
			}
		}(file)
		logrus.Info("上传文件, 文件获取成功!")

		// 创建目标文件
		uploadFileName := filepath.Join(dir, filePath, handler.Filename)
		var destFile *os.File
		destFile, err = os.Create(uploadFileName)
		if err != nil {
			logrus.Error("上传文件发生异常, 异常为: ", err)
			return
		}
		defer func(destFile *os.File) {
			err = destFile.Close()
			if err != nil {
				logrus.Error("上传文件发生异常, 异常为: ", err)
			}
		}(destFile)
		logrus.Info("上传文件, 文件创建成功!")

		//将文件内容拷贝到目标文件
		_, err = io.Copy(destFile, file)
		if err != nil {
			logrus.Error("上传文件发生异常, 异常为: ", err)
			return
		}
	}
	logrus.Info("上传文件方法执行成功!")
}

// HandleMkdirRequest 创建文件夹
func HandleMkdirRequest(w http.ResponseWriter, r *http.Request) {
	// 认证
	cookie, err := r.Cookie("session_id")
	err = authToCookie(cookie)
	if err != nil {
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("创建文件夹发生异常, 异常为: ", err)
		}
		return
	}

	// 判断功能是否开启
	if !isMkdir {
		logrus.Error("上传文件非法访问, 功能未开启!")
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(resultError(-2, errors.New("功能未开启! ")))
		if err != nil {
			logrus.Error("创建文件夹发生异常, 异常为: ", err)
		}
		return
	}

	// 获取参数并组织返回参数
	query := r.URL.Query()
	dirPath := query.Get("dirPath")

	// 创建目录
	mkdir := path.Join(dir, dirPath)
	err = os.Mkdir(mkdir, 0755)
	if err != nil {
		logrus.Error("创建文件夹发生异常, 异常为: ", err)
		_, err = w.Write(resultError(-2, err))
		if err != nil {
			logrus.Error("创建文件夹发生异常, 异常为: ", err)
		}
		return
	}

	// 设置 Content-Type 为 application/json
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(resultSuccess(nil))
	if err != nil {
		logrus.Error("创建文件夹发生异常, 异常为: ", err)
		return
	}
	logrus.Info("创建文件夹方法执行成功!")
}
