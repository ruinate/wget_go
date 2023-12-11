package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Message interface{}

type GetModel struct {
	Url      string
	Work     int
	filename string
	mu       sync.Mutex
	wg       sync.WaitGroup
}

// 创建文件
func (w *GetModel) initFile() Message {
	// 判断文件是否存在
	if _, err := os.Stat(w.filename); !os.IsNotExist(err) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("The file name exists, please command the file name again：")
		name, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if name != "\n" {
			w.filename = name
		}
	}
	file, err := os.Create(w.filename)
	if err != nil {
		return errors.New(fmt.Sprintf("Error: File not found at %s", err))
	}
	defer file.Close()
	return nil
}

func (w *GetModel) getSize() Message {
	resp, err := http.Head(w.Url)
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting file information: %s", err))
	}
	defer resp.Body.Close()
	return int(resp.ContentLength)
}

func (w *GetModel) download(start, end int, wg *sync.WaitGroup, ch chan int64) Message {
	defer wg.Done()
	client := &http.Client{}

	req, err := http.NewRequest("GET", w.Url, nil)
	if err != nil {
		return err
	}
	// 指定要下载的字节范围
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 打开文件
	file, err := os.OpenFile(w.filename, os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Seek(int64(start), 0)
	buffer := make([]byte, 1024)
	// 循环读取和写入文件，并异步更新进度条
	for {

		n, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			fmt.Println("Error reading data from response body:", err)
			break
		}
		if n == 0 {
			break
		}
		// 写入文件
		if _, err := file.Write(buffer[:n]); err != nil {
			fmt.Println("Error writing data to file:", err)
			break
		}
		// 异步更新进度条
		ch <- int64(n)
	}

	return nil
}

func (w *GetModel) updateCh(ch chan int64, bar *pb.ProgressBar) {
	for progress := range ch {
		w.mu.Lock()
		bar.Add64(progress)
		w.mu.Unlock()
	}
}

func (w *GetModel) execute() Message {
	var (
		Size       int
		progressCh = make(chan int64, w.Work)
	)
	// 每个协程下载的文件大小
	RespSize := w.getSize()
	if err, ok := RespSize.(error); ok {
		return err
	} else {
		Size, _ = RespSize.(int)
	}

	// 补充～目录
	filename, _ := expandTilde(CONFIGFILENAME)
	var set SetModel
	file, err := os.Open(filename)
	if err != nil {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("配置文件未找到，文件是否下载到当前目录: [yes/no]")
			// Read the user input
			input, err := reader.ReadString('\n')
			if err != nil {
				return errors.New("Error reading input: " + err.Error())
			}
			// Trim leading and trailing whitespaces
			input = strings.TrimSpace(input)
			// Check if the input is valid
			if input == "yes" {
				w.filename = w.Url[strings.LastIndex(w.Url, "/")+1:]
				break
			} else if input == "no" {
				os.Exit(0)
			} else {
				logger.Println("Invalid input. Please enter yes or no.")
			}
		}
	} else {
		defer file.Close()
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&set)
		if err != nil {
			return err
		}

		w.filename = set.DownloadPath + w.Url[strings.LastIndex(w.Url, "/")+1:]
	}

	//w.filename = w.Url[strings.LastIndex(w.Url, "/")+1:]

	chunkSize := Size / w.Work
	//  添加空文件
	if err := w.initFile(); err != nil {
		return err
	}
	startTime := time.Now()
	// 设置进度条
	bar := pb.New(Size)
	// 设置进度条样式
	bar.SetTemplate(`{{string . "prefix" | blue}} {{bar . "[" "=" ">" "." "]" | green}} {{percent . }} {{speed . | green}} {{rtime . | yellow}}`)
	// 设置刷新速度（时间间隔）（默认为200毫秒）
	bar.SetRefreshRate(time.Second)
	// 强制设置io.Writer，默认为os.Stderr
	bar.SetWriter(os.Stdout)
	// 进度条将数字格式化为字节（B、KiB、MiB等）
	bar.Set(pb.Bytes, true)

	//tmpl := `{{ red "With funcs:" }} {{ bar . "<" "-" (cycle . "↖" "↗" "↘" "↙" ) "." ">"}} {{speed . | rndcolor }} {{percent .}} {{string . "my_green_string" | green}} {{string . "my_blue_string" | blue}}`
	//
	//// 开始基于我们的模板的进度条
	//bar := pb.ProgressBarTemplate(tmpl).Start(Size)
	//// 设置字符串元素的值
	////bar.Set("my_green_string", "green").Set("my_blue_string", "blue")
	//bar.Set(pb.Bytes, true)
	//// 进度条开始
	bar.Start()

	// 启动协程异步更新进度条
	go w.updateCh(progressCh, bar)

	for i := 0; i < w.Work; i++ {
		startSize := i * chunkSize
		endSize := (i + 1) * chunkSize
		// work只有一个的话，下载全部字节
		if i == w.Work-1 {
			endSize = Size
		}
		w.wg.Add(1)
		go w.download(startSize, endSize, &w.wg, progressCh)
	}
	w.wg.Wait()
	close(progressCh)
	bar.Finish()
	endTime := time.Now()
	logger.Infof("协程数量：%d    执行时间：%s \n 文件路径: %s", w.Work, endTime.Sub(startTime).String(), w.filename)
	return nil
}

func (w *GetModel) InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "get",
		Short:     "下载文件",
		Long:      ``,
		ValidArgs: []string{"w", "u"},
		Args:      cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err, ok := w.execute().(error); ok {
				logger.Errorln(err)
			}
		},
	}

}
