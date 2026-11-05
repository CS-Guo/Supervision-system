// Package storage 提供附件二进制文件存储，附件元数据仍保存在 MySQL 中。
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// Local 使用随机存储键而不是用户提供的文件名，将文件保存在 Root 下。
// 后续可以替换为对象存储实现，而不需要修改领域数据结构。
type Local struct{ Root string }

// Save 将上传文件写入新建路径，并返回不可推测的存储键和实际字节数。
func (l Local) Save(file multipart.File) (string, int64, error) {
	b := make([]byte, 20)
	if _, e := rand.Read(b); e != nil {
		return "", 0, e
	}
	key := hex.EncodeToString(b)
	if e := os.MkdirAll(l.Root, 0750); e != nil {
		return "", 0, e
	}
	p := filepath.Join(l.Root, key)
	out, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if e != nil {
		return "", 0, e
	}
	n, e := io.Copy(out, file)
	ce := out.Close()
	if e != nil {
		_ = os.Remove(p)
		return "", 0, e
	}
	if ce != nil {
		return "", 0, ce
	}
	return key, n, nil
}

// Open 只使用 key 的基础文件名解析路径，防止目录穿越。
func (l Local) Open(key string) (*os.File, error) {
	return os.Open(filepath.Join(l.Root, filepath.Base(key)))
}

// Delete 用于后续数据库事务失败时执行附件补偿清理。
func (l Local) Delete(key string) { _ = os.Remove(filepath.Join(l.Root, filepath.Base(key))) }
