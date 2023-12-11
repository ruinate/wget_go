// Package main -----------------------------
// @file      : wget.go
// @author    : fzf
// @contact   : fzf54122@163.com
// @time      : 2023/12/11 上午9:51
// -------------------------------------------
package main

import (
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"runtime"
	"wget_go/core"
)

const VERSION string = "0.1.0"

var (
	WgetCmd = &cobra.Command{
		Use:       "wget_go",
		Short:     "Display one or many resources",
		Long:      ``,
		ValidArgs: []string{"get", "set"},
		Args:      cobra.OnlyValidArgs,
		Version:   VERSION,
	}

	getCmd core.GetModel
	setCmd core.SetModel
)

func init() {
	get := getCmd.InitCmd()
	set := setCmd.InitCmd()
	WgetCmd.AddCommand(get, set)
	get.Flags().StringVarP(&getCmd.Url, "url", "u", "https://example.com/", "下载地址")
	get.Flags().IntVarP(&getCmd.Work, "work", "w", runtime.NumCPU(), "下载协程数")
	//
	set.Flags().StringVarP(&setCmd.DownloadPath, "file", "f", "~/Downloads/", "文件下载路径")
}

func main() {
	if err := WgetCmd.Execute(); err != nil {
		logger.Errorln(err)
	}
}
