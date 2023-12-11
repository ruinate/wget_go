package core

import (
	"encoding/json"
	"errors"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/user"
	"path/filepath"
)

type SetModel struct {
	DownloadPath string `json:"download_path"`
}

const (
	CONFIGPATH     string = "~/.wget_go"
	CONFIGFILENAME string = "~/.wget_go/.env"
)

func expandTilde(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(usr.HomeDir, path[1:]), nil
	}

	return path, nil
}

// 判断文件夹和文件是否存在;不存在返回错误;存在返回nil
func (s *SetModel) checkFileNotExist() error {
	configPath, err := expandTilde(CONFIGPATH)
	if err != nil {
		return err
	}
	// 判断文件夹目录是否存在;不存在返回错误;存在
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		err = os.MkdirAll(configPath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	// 判断.env 文件
	envPath, err := expandTilde(CONFIGFILENAME)
	if err != nil {
		return err
	}
	if _, err = os.Stat(envPath); !os.IsNotExist(err) {
	}
	file, err := os.Create(envPath)
	if err != nil {
		return errors.New(fmt.Sprintf("Error: File not found at %s", err))
	}
	defer file.Close()

	return nil
}

func (s *SetModel) execute() Message {
	err := s.checkFileNotExist()
	if err != nil {
		return err
	}

	var downloadPath string
	if len(s.DownloadPath) > 0 && s.DownloadPath[0] == '~' {
		usr, err := user.Current()
		if err != nil {
			return err
		}
		downloadPath = filepath.Join(usr.HomeDir, s.DownloadPath[1:]) + "/"
	}

	filename, _ := expandTilde(CONFIGFILENAME)
	// 写入文件
	file, err := os.OpenFile(filename, os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(SetModel{DownloadPath: downloadPath})
	if err != nil {
		return err
	}
	logger.Infof("下载路径设置为：%s", downloadPath)
	return nil
}

func (s *SetModel) InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "set",
		Short:     "设置下载目录",
		Long:      ``,
		ValidArgs: []string{"-f"},
		Args:      cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err, ok := s.execute().(error); ok {
				logger.Errorln(err)
			}
		},
	}
}
