package src

import (
	"encoding/json"
	"errors"
	"fmt"
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
	isUpload    = os.Getenv("IS_UPDATE") == "true"     // 是否开启上传
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
			logrus.Error("基本查询发生异常，异常为: ", err.Error())
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
		logrus.Error("基本查询发生异常，异常为: ", err.Error())
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
			logrus.Error("登录发生异常，异常为: ", err.Error())
		}
		return
	}

	// 写入cookie
	http.SetCookie(w, cookie)
	_, err = w.Write(resultSuccess(now.Unix()))
	if err != nil {
		logrus.Error("登录发生异常，异常为: ", err.Error())
	}
	logrus.Error("登录方法执行成功!")
}

// HandleQueryRequest 查询文件
func HandleQueryRequest(w http.ResponseWriter, r *http.Request) {
	// 验证
	cookie, err := r.Cookie("session_id")
	err = authToCookie(cookie)
	if err != nil {
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("文件查询发生异常, 异常为: ", err.Error())
		}
		return
	}

	// 获取参数, 初始化目录
	localDir := dir
	query := r.URL.Query()
	filePath := query.Get("filePath")
	localDir = filepath.Join(localDir, filePath)
	showHidden, _ = strconv.ParseBool(query.Get("showHidden"))   // 是否显示隐藏文件
	showDirSize, _ = strconv.ParseBool(query.Get("showDirSize")) // 是否计算文件夹大小

	// 打开目录
	d, err := os.Open(localDir)
	if err != nil {
		logrus.Error("文件查询发生异常, 异常为: ", err.Error())
		_, err = w.Write(resultError(-3, err))
		if err != nil {
			logrus.Error("文件查询发生异常, 异常为: ", err.Error())
		}
		return
	}
	defer func(d *os.File) {
		err := d.Close()
		if err != nil {

		}
	}(d)

	// 读取当前目录下的文件和子目录
	files, fileErr := d.ReadDir(0)
	if fileErr != nil {
		fmt.Println("读取文件异常:", fileErr)
		_, err := w.Write(resultError(-3, fileErr))
		if err != nil {
			return
		}
		return
	}

	// 组织返回的json
	var filesList []map[string]string
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return
		}
		if !showHidden && len(info.Name()) > 0 && info.Name()[0] == '.' {
			continue
		}
		files := make(map[string]string)
		files["path"] = filePath
		files["name"] = info.Name()
		files["fileType"] = ""
		fileNameSplit := strings.Split(info.Name(), ".")
		if len(fileNameSplit) > 1 {
			files["fileType"] = fileNameSplit[len(fileNameSplit)-1]
		}
		files["size"] = formatSize(float64(info.Size()), 0)
		files["modified"] = info.ModTime().Format("2006-01-02 15:04:05")
		files["type"] = "file"
		if info.IsDir() {
			files["fileType"] = "folder"
			dirPath := filepath.Join(dir, file.Name())
			files["type"] = "dir"
			files["size"] = ""
			if showDirSize {
				size, err := dirSize(dirPath)
				if err != nil {
					fmt.Println("计算目录大小异常:", err)
				}
				files["size"] = formatSize(float64(size), 0)
			}
		}
		filesList = append(filesList, files)
		sort.Slice(filesList, func(i, j int) bool {
			return filesList[j]["type"] == "file"
		})
	}

	// 设置 Content-Type 为 application/json
	w.Header().Set("Content-Type", "application/json")

	// 返回 JSON 数据
	_, err = w.Write(resultSuccess(filesList))
	if err != nil {
		return
	}
}

// HandleDownloadRequest 下载文件
func HandleDownloadRequest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Path
	reqPthSplit := strings.Split(strings.TrimLeft(query, "/"), "/")
	fileName := reqPthSplit[len(reqPthSplit)-1]
	reName := strings.ReplaceAll(fileName, "pkgDir_", "")
	if reName == ".zip" {
		reName = "root.zip"
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+reName)

	// 认证
	authCode := reqPthSplit[1]
	var pathList []string
	tag, msg := authToPath(authCode)
	// 第一个元素是url,最后一个是文件名
	pathList = append(reqPthSplit[2 : len(reqPthSplit)-1])
	if msg != "" {
		fmt.Println(msg)
		if !tag {
			http.Error(w, msg, 500)
			return
		}
		// 第二个元素是auth
		pathList = append(reqPthSplit[2 : len(reqPthSplit)-1])
	}

	filePath := path.Join(pathList...)
	// 处理文件夹的情况
	if strings.HasPrefix(fileName, "pkgDir_") {
		fileName = strings.ReplaceAll(fileName, "pkgDir_", "")
		dirPath := path.Join(dir, filePath, strings.ReplaceAll(fileName, ".zip", ""))
		if fileName == ".zip" {
			fileName = "root" + fileName
		}
		createZipDir(dirPath, fileName)
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {

			}
		}(fileName)

		// 打开 zip 文件
		file, err := os.Open(fileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {

			}
		}(file)

		// 设置响应头
		w.Header().Set("Content-Type", "application/zip")

		// 将 zip 文件内容复制到响应体中
		_, err = io.Copy(w, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	// 加载文件
	downloadFile := filepath.Join(dir, filePath, fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, downloadFile)
}

// HandleRemoveRequest 删除文件
func HandleRemoveRequest(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	// 认证
	authErr := authToCookie(cookie)
	if authErr != nil {
		fmt.Println(authErr)
		_, err := w.Write(resultError(-3, authErr))
		if err != nil {
			return
		}
		return
	}
	// 功能是否开启
	if !isDelete {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(resultError(-2, errors.New("功能未开启! ")))
		if err != nil {
			return
		}
		return
	}

	// 初始化目录，获取参数
	dir := dir
	query := r.URL.Query()

	// 获取文件路径
	filePath := query.Get("filePath")

	// 获取文件名
	fileName := query.Get("fileName")

	// 尝试删除文件
	err = os.Remove(path.Join(dir, filePath, fileName))
	if err != nil {
		fmt.Println("删除文件异常:", err)
		_, err := w.Write(resultError(-1, err))
		if err != nil {
			return
		}
		return
	}
	fmt.Println("删除成功！")

	// 设置 Content-Type 为 application/json
	w.Header().Set("Content-Type", "application/json")

	// 返回 JSON 数据
	_, err = w.Write(resultSuccess(nil))
}

// HandleUploadRequest 上传文件
func HandleUploadRequest(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	// 获取cookie
	if cookie == nil {
		_, err = w.Write(resultError(-3, errors.New("未验证, 请先进行验证! ")))
		if err != nil {
			return
		}
		return
	}
	// 认证
	authErr := authToCookie(cookie)
	if authErr != nil {
		fmt.Println(authErr)
		_, err := w.Write(resultError(-3, authErr))
		if err != nil {
			return
		}
		return
	}
	// 功能是否开启
	if !isUpload {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(resultError(-2, errors.New("功能未开启! ")))
		if err != nil {
			return
		}
		return
	}
	if r.Method == "POST" {
		// 解析表单数据
		err := r.ParseMultipartForm(10 << 20) // 最大10MB
		if err != nil {
			fmt.Println(err)
			return
		}
		filePath := r.FormValue("filePath")

		// 获取文件句柄和文件头信息
		file, handler, err := r.FormFile("file")
		if err != nil {
			fmt.Println("Error Retrieving the File")
			fmt.Println(err)
			return
		}
		defer func(file multipart.File) {
			err := file.Close()
			if err != nil {

			}
		}(file)

		fmt.Printf("Uploaded File: %+v\n", handler.Filename)
		fmt.Printf("File Size: %+v\n", handler.Size)
		fmt.Printf("MIME Header: %+v\n", handler.Header)

		// 创建目标文件
		uploadFileName := filepath.Join(dir, filePath, handler.Filename)
		destFile, err := os.Create(uploadFileName)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer func(destFile *os.File) {
			err := destFile.Close()
			if err != nil {

			}
		}(destFile)

		//将文件内容拷贝到目标文件
		_, err = io.Copy(destFile, file)
		if err != nil {
			fmt.Println(err)
			return
		}

	}
}

// HandleMkdirRequest 创建文件夹
func HandleMkdirRequest(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	// 获取cookie
	if cookie == nil {
		_, err := w.Write(resultError(-3, errors.New("未验证, 请先进行验证! ")))
		if err != nil {
			return
		}
		return
	}
	// 认证
	authErr := authToCookie(cookie)
	if authErr != nil {
		fmt.Println(authErr)
		_, err := w.Write(resultError(-3, authErr))
		if err != nil {
			return
		}
		return
	}
	// 判断功能是否开启
	if !isMkdir {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(resultError(-2, errors.New("功能未开启! ")))
		if err != nil {
			fmt.Println("出现异常: ", err)
		}
		return
	}
	result := make(map[string]string)
	query := r.URL.Query()
	dirPath := query.Get("dirPath")
	// 创建目录
	mkdir := path.Join(dir, dirPath)
	err = os.Mkdir(mkdir, 0755)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}
	result["code"] = "1"
	// 设置 Content-Type 为 application/json
	w.Header().Set("Content-Type", "application/json")

	resultJson, err := json.Marshal(result)

	// 返回 JSON 数据
	_, err = w.Write(resultJson)
}
