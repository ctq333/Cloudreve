package explorer

import (
	"context"
	"fmt"
	model "github.com/HFO4/cloudreve/models"
	"github.com/HFO4/cloudreve/pkg/auth"
	"github.com/HFO4/cloudreve/pkg/cache"
	"github.com/HFO4/cloudreve/pkg/filesystem"
	"github.com/HFO4/cloudreve/pkg/filesystem/fsctx"
	"github.com/HFO4/cloudreve/pkg/serializer"
	"github.com/HFO4/cloudreve/pkg/util"
	"github.com/gin-gonic/gin"
	"net/url"
	"time"
)

// ItemMoveService 处理多文件/目录移动
type ItemMoveService struct {
	SrcDir string      `json:"src_dir" binding:"required,min=1,max=65535"`
	Src    ItemService `json:"src" binding:"exists"`
	Dst    string      `json:"dst" binding:"required,min=1,max=65535"`
}

// ItemRenameService 处理多文件/目录重命名
type ItemRenameService struct {
	Src     ItemService `json:"src" binding:"exists"`
	NewName string      `json:"new_name" binding:"required,min=1,max=255"`
}

// ItemService 处理多文件/目录相关服务
type ItemService struct {
	Items []uint `json:"items" binding:"exists"`
	Dirs  []uint `json:"dirs" binding:"exists"`
}

// Archive 创建归档
func (service *ItemService) Archive(ctx context.Context, c *gin.Context) serializer.Response {
	// 创建文件系统
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}

	// 检查用户组权限
	if !fs.User.Group.OptionsSerialized.ArchiveDownloadEnabled {
		return serializer.Err(serializer.CodeGroupNotAllowed, "当前用户组无法进行此操作", nil)
	}

	// 开始压缩
	ctx = context.WithValue(ctx, fsctx.GinCtx, c)
	zipFile, err := fs.Compress(ctx, service.Dirs, service.Items)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, "无法创建压缩文件", err)
	}

	// 生成一次性压缩文件下载地址
	siteURL, err := url.Parse(model.GetSettingByName("siteURL"))
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, "无法解析站点URL", err)
	}
	zipID := util.RandStringRunes(16)
	signedURI, err := auth.SignURI(
		fmt.Sprintf("/api/v3/file/archive/%s/archive.zip", zipID),
		time.Now().Unix()+30,
	)
	finalURL := siteURL.ResolveReference(signedURI).String()

	// 将压缩文件记录存入缓存
	err = cache.Set("archive_"+zipID, zipFile, 30)
	if err != nil {
		return serializer.Err(serializer.CodeIOFailed, "无法写入缓存", err)
	}

	return serializer.Response{
		Code: 0,
		Data: finalURL,
	}
}

// Delete 删除对象
func (service *ItemService) Delete(ctx context.Context, c *gin.Context) serializer.Response {
	// 创建文件系统
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}

	// 删除对象
	err = fs.Delete(ctx, service.Dirs, service.Items)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
	}

}

// Move 移动对象
func (service *ItemMoveService) Move(ctx context.Context, c *gin.Context) serializer.Response {
	// 创建文件系统
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}

	// 移动对象
	err = fs.Move(ctx, service.Src.Dirs, service.Src.Items, service.SrcDir, service.Dst)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
	}

}

// Copy 复制对象
func (service *ItemMoveService) Copy(ctx context.Context, c *gin.Context) serializer.Response {
	// 复制操作只能对一个目录或文件对象进行操作
	if len(service.Src.Items)+len(service.Src.Dirs) > 1 {
		return serializer.ParamErr("只能复制一个对象", nil)
	}

	// 创建文件系统
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}

	// 复制对象
	err = fs.Copy(ctx, service.Src.Dirs, service.Src.Items, service.SrcDir, service.Dst)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
	}

}

// Rename 重命名对象
func (service *ItemRenameService) Rename(ctx context.Context, c *gin.Context) serializer.Response {
	// 重命名作只能对一个目录或文件对象进行操作
	if len(service.Src.Items)+len(service.Src.Dirs) > 1 {
		return serializer.ParamErr("只能操作一个对象", nil)
	}

	// 创建文件系统
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}

	// 重命名对象
	err = fs.Rename(ctx, service.Src.Dirs, service.Src.Items, service.NewName)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
	}
}