package comm

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	logger = InitLog()
}

// exec and wait for command
func ExecAndWait(timeout int, name string, arg ...string) (stdout, stderr string, err error) {
	logger.Debugf("cmd: %s %+v\n", name, arg)
	cmd := exec.Command(name, arg...)
	var bufStdout, bufStderr bytes.Buffer
	cmd.Stdout = &bufStdout
	cmd.Stderr = &bufStderr
	err = cmd.Start()
	if err != nil {
		err = fmt.Errorf("start fail: %w", err)
		return
	}

	// wait for process finished
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err = cmd.Process.Kill(); err != nil {
			err = fmt.Errorf("timeout: %w", err)
			return
		}
		<-done
		err = fmt.Errorf("time out and process was killed")
	case err = <-done:
		stdout = bufStdout.String()
		stderr = bufStderr.String()
		if err != nil {
			err = fmt.Errorf("run: %w", err)
			return
		}
	}
	return
}

// deb config info struct
type DebConfig struct {
	Info struct {
		Appid       string `yaml:"appid"`
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
		Kind        string `yaml:"kind"`
		Arch        string `yaml:"arch"`
	} `yaml:"info"`
	FileElement struct {
		Deb     []DebInfo `yaml:"deb"`
		Package []string  `yaml:"add-package"`
	} `yaml:"file"`
	BuildInfo struct{} `yaml:"build"`
}

type DebInfo struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Ref  string `yaml:"ref"`
	Hash string `yaml:"hash"`
	Path string
}

func (ts *DebInfo) CheckDebHash() bool {
	if ts.Hash == "" {
		return false
	}
	hash, err := GetFileSha256(ts.Path)
	if err != nil {
		logger.Warn(err)
		return false
	}
	if hash == ts.Hash {
		return true
	}

	return false
}

func (ts *DebInfo) FetchDebFile(dirPath string) bool {

	logger.Debugf("FetchDebFile :%v", ts)
	if ts.Type == "repo" {
		// ts.path = fmt.Sprintf("%s/", dirPath)

		_, msg, err := ExecAndWait(1<<20, "wget", "-P", dirPath, ts.Ref)
		if err != nil {

			logger.Errorf("msg: %+v err:%+v", msg, err)
			return false
		}
		debFilePath, err := filepath.Glob(fmt.Sprintf("%s/%s_*.deb", dirPath, ts.Name))
		if err != nil {
			logger.Error(debFilePath)
			return false
		}
		logger.Debugf("debFilePath: %+v [0]:%s", debFilePath, debFilePath[0])
		if err, msg := CheckFileExits(debFilePath[0]); err {
			ts.Path = debFilePath[0]
			return true
		} else {
			logger.Errorf("msg: %+v err:%+v", msg, err)
			return false
		}
	}
	return false
}

func (ts *DebConfig) MergeInfo(t *DebConfig) bool {
	if ts.Info.Appid == "" {
		ts.Info.Appid = t.Info.Appid
	}
	if ts.Info.Name == "" {
		ts.Info.Name = t.Info.Name
	}
	if ts.Info.Version == "" {
		ts.Info.Version = t.Info.Version
	}
	if ts.Info.Description == "" {
		ts.Info.Description = t.Info.Description
	}
	if ts.Info.Kind == "" {
		ts.Info.Kind = t.Info.Kind
	}
	if ts.Info.Arch == "" {
		ts.Info.Arch = t.Info.Arch
	}
	return true
}

type BaseConfig struct {
	SdkInfo struct {
		Base  []BaseInfo `yaml:"base"`
		Extra ExtraInfo  `yaml:"extra"`
	} `yaml:"sdk"`
}

type BaseInfo struct {
	Type   string `yaml:"type"`
	Ref    string `yaml:"ref"`
	Hash   string `yaml:"hash"`
	Remote string `yaml:"remote"`
	Path   string
}

func (ts *BaseInfo) CheckIsoHash() bool {
	if ts.Hash == "" {
		return false
	}
	hash, err := GetFileSha256(ts.Path)
	if err != nil {
		logger.Warn(err)
		return false
	}
	if hash == ts.Hash {
		return true
	}

	return false
}

func (ts *BaseInfo) FetchIsoFile(workdir, isopath string) bool {
	//转化绝对路径
	isoAbsPath, _ := filepath.Abs(isopath)
	//如果下载目录不存在就创建目录
	CreateDir(GetFilePPath(isoAbsPath))
	if ts.Type == "iso" {
		ts.Path = isoAbsPath
		_, msg, err := ExecAndWait(1<<20, "wget", "-O", ts.Path, ts.Ref)
		if err != nil {
			logger.Errorf("msg: %+v err:%+v", msg, err)
			return false
		}
		return true
	}
	return false
}

func (ts *BaseInfo) CheckoutOstree(target string) bool {
	// configInfo.RuntimeBasedir = fmt.Sprintf("%s/runtimedir", configInfo.Workdir)
	logger.Debug("ostree checkout %s to %s", ts.Path, target)
	_, msg, err := ExecAndWait(10, "ostree", "checkout", "--repo", ts.Path, ts.Ref, target)

	if err != nil {
		logger.Errorf("msg: %v ,err: %+v", msg, err)
		return false
	}
	return false
}

func (ts *BaseInfo) InitOstree(workdir string) bool {
	if ts.Type == "ostree" {
		logger.Debug("ostree init")
		ts.Path = fmt.Sprintf("%s/runtime", workdir)
		_, msg, err := ExecAndWait(10, "ostree", "init", "--mode=bare-user-only", "--repo", ts.Path)
		if err != nil {
			logger.Errorf("msg: %v ,err: %+v", msg, err)
			return false
		}
		logger.Debug("ostree remote add", ts.Remote)

		_, msg, err = ExecAndWait(10, "ostree", "remote", "add", "runtime", ts.Remote, "--repo", ts.Path, "--no-gpg-verify")
		if err != nil {
			logger.Errorf("msg: %+v err:%+v", msg, err)
			return false
		}

		logger.Debug("ostree pull")
		_, msg, err = ExecAndWait(30, "ostree", "pull", "runtime", "--repo", ts.Path, "--mirror", ts.Ref)
		if err != nil {
			logger.Errorf("msg: %+v err:%+v", msg, err)
			return false
		}

		return true
	}
	return false
}

type ExtraInfo struct {
	repo    []string `yaml:"repo"`
	Package []string `yaml:"package"`
	Cmd     []string `yaml:"command"`
}

func GetFileSha256(filename string) (string, error) {
	logger.Debug("GetFileSha256 :", filename)
	hasher := sha256.New()
	s, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Warn(err)
		return "", err
	}
	_, err = hasher.Write(s)
	if err != nil {
		logger.Warn(err)
		return "", err
	}

	sha256Sum := hex.EncodeToString(hasher.Sum(nil))
	logger.Debug("file hash: ", sha256Sum)

	return sha256Sum, nil
}
