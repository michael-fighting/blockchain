package apiserver
//
//import (
//	"fmt"
//	"github.com/aws/aws-sdk-go/aws"
//	"github.com/aws/aws-sdk-go/service/s3/s3manager"
//	"go.uber.org/zap"
//	"io"
//	"mime/multipart"
//	"os"
//	"path"
//	"portal-backend/server/global"
//	"time"
//)
//
//func (srv *ApiServer) UploadFile(file *multipart.FileHeader) (string, string, error) {
//	// 读取文件后缀
//	ext := path.Ext(file.Filename)
//	// 拼接新文件名
//	filename := utils.GenRandomHex32() + "_" + time.Now().Format("20060102150405") + ext
//	// 尝试创建此路径
//	mkdirErr := os.MkdirAll(global.GVA_CONFIG.Local.Path, os.ModePerm)
//	if mkdirErr != nil {
//		global.GVA_LOG.Error("function os.MkdirAll() Filed", zap.Any("err", mkdirErr.Error()))
//		return "", "", errors.New("function os.MkdirAll() Filed, err:" + mkdirErr.Error())
//	}
//	// 拼接路径和文件名
//	p := global.GVA_CONFIG.Local.Path + "/" + filename
//
//	f, openError := file.Open() // 读取文件
//	if openError != nil {
//		global.GVA_LOG.Error("function file.Open() Filed", zap.Any("err", openError.Error()))
//		return "", "", errors.New("function file.Open() Filed, err:" + openError.Error())
//	}
//	defer f.Close() // 创建文件 defer 关闭
//
//	out, createErr := os.Create(p)
//	if createErr != nil {
//		global.GVA_LOG.Error("function os.Create() Filed", zap.Any("err", createErr.Error()))
//
//		return "", "", errors.New("function os.Create() Filed, err:" + createErr.Error())
//	}
//	defer out.Close() // 创建文件 defer 关闭
//
//	_, copyErr := io.Copy(out, f) // 传输（拷贝）文件
//	if copyErr != nil {
//		global.GVA_LOG.Error("function io.Copy() Filed", zap.Any("err", copyErr.Error()))
//		return "", "", errors.New("function io.Copy() Filed, err:" + copyErr.Error())
//	}
//	return p, filename, nil
//}
