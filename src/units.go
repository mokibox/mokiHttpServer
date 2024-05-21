package src

import (
	"archive/zip"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 格式化文件大小
func formatSize(size float64, unit int) string {
	// 递归得出size的单位
	if int(size)/1024 != 0 {
		unit := unit + 1
		return formatSize(size/1024, unit)
	}
	// 将数字进行处理
	num := handleNum(size)
	// 添加单位并返回
	if unit == 0 {
		return num + "B"
	} else if unit == 1 {
		return num + "KB"
	} else if unit == 2 {
		return num + "MB"
	} else if unit == 3 {
		return num + "GB"
	} else if unit == 4 {
		return num + "TB"
	}
	return num + "?B"
}

// 处理数字
func handleNum(num float64) string {
	// 如果没有小数位 直接返回整数位即可
	if int(num*100)%100 == 0 {
		return strconv.Itoa(int(num))
	}
	return fmt.Sprintf("%.2f", num)
}

// 计算文件大小
func dirSize(dir string) int64 {
	var size int64
	// 遍历文件夹下的所有文件和子文件夹
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Error("计算文件夹大小发生异常, 异常为: ", err)
		}
		// 如果是文件，则累加文件大小
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		logrus.Error("计算文件夹大小发生异常, 异常为: ", err)
		return 0
	}
	return size
}

// 生成hash
func generateHash(str string, genLen int) string {
	data := []byte(str)
	var hash []byte
	h := md5.New()
	h.Write(data)
	hash = h.Sum(nil)
	if genLen != -1 {
		return hex.EncodeToString(hash)[:genLen]
	}
	return hex.EncodeToString(hash)
}

// 生成sessionId
func generateSessionID() string {
	// 创建一个32字节的随机字节数组
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		logrus.Error("sessionId生成发生异常, 异常为: ", err)
		return ""
	}
	// 将随机字节数组编码为URL安全的Base64字符串
	return base64.URLEncoding.EncodeToString(b)
}

// 创建目录压缩
func createZipDir(dir, zipFile string) {
	// 创建一个新的 zip 文件
	newZipFile, err := os.Create(zipFile)
	if err != nil {
		logrus.Error("压缩文件创建发生异常, 异常为: ", err)
		return
	}
	defer func(newZipFile *os.File) {
		err = newZipFile.Close()
		if err != nil {
			logrus.Error("压缩文件创建发生异常, 异常为: ", err)
			return
		}
	}(newZipFile)

	// 创建一个 zip.Writer
	zipWriter := zip.NewWriter(newZipFile)
	defer func(zipWriter *zip.Writer) {
		err = zipWriter.Close()
		if err != nil {
			logrus.Error("压缩文件创建发生异常, 异常为: ", err)
			return
		}
	}(zipWriter)

	// 遍历目录中的所有文件和子目录
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 将文件或目录添加到 zip 文件中
		if !info.IsDir() {
			var (
				file *os.File
				f    io.Writer
			)
			file, err = os.Open(path)
			if err != nil {
				return err
			}
			defer func(file *os.File) {
				err = file.Close()
				if err != nil {
					logrus.Warning("压缩文件创建发生异常, 异常为: ", err)
				}
			}(file)

			// 创建 zip 文件中的文件
			zipPath := strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), dir+"/")
			if zipPath == "" {
				return nil
			}
			zipPath = filepath.ToSlash(zipPath)
			f, err = zipWriter.Create(zipPath)
			if err != nil {
				return err
			}

			// 将文件内容复制到 zip 文件中
			_, err = io.Copy(f, file)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logrus.Warning("压缩文件创建发生异常, 异常为: ", err)
	}
}

// 返回数据
func resultError(code int, err error) []byte {
	return resultAll("操作失败!", code, nil, err)
}

func resultSuccess(data any) []byte {
	return resultAll("操作成功!", 1, data, nil)
}

func resultAll(msg string, code int, data any, err error) []byte {
	result := make(map[string]any)
	result["code"] = code
	result["msg"] = msg
	if data != nil {
		result["data"] = data
	}
	if err != nil {
		result["err"] = err.Error()
	}
	marshal, err := json.Marshal(result)
	if err != nil {
		fmt.Println("json转换错误！")
	}
	return marshal
}
