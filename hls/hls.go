package hls

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/grafov/m3u8"
)

var (
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger

	wg         sync.WaitGroup
	killSignal = "/tmp/dlm3u8.stop"
)

func init() {
	logFile, err := os.OpenFile("logs.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("打开日志文件失败：", err)
	}
	Info = log.New(io.MultiWriter(os.Stdout, logFile), "Info:", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(io.MultiWriter(os.Stdout, logFile), "Warning:", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(io.MultiWriter(os.Stderr, logFile), "Error:", log.Ldate|log.Ltime|log.Lshortfile)
}

// GetPlaylist fetch content from remote url and return a list of segments
func GetPlaylist(url string) (*m3u8.MediaPlaylist, error) {
	t, err := FileGetContents(url)
	if err != nil {
		return nil, err
	}

	playlist, listType, err := m3u8.DecodeFrom(strings.NewReader(t), true)
	if err != nil {
		return nil, err
	}

	switch listType {
	case m3u8.MEDIA:
		p := playlist.(*m3u8.MediaPlaylist)
		return p, nil
	default:
		return nil, nil
	}
}

func BuildSegments(u string) ([]string, error) {
	playlistURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	p, err := GetPlaylist(u)
	if err != nil {
		Info.Printf("url: %s, error: no m3u8 data found\n", u)
		return nil, err
	}

	var urls []string

	for _, v := range p.Segments {
		if v == nil {
			continue
		}

		var segmentURI string
		if strings.HasPrefix(v.URI, "http") {
			segmentURI, err = url.QueryUnescape(v.URI)
			if err != nil {
				return nil, err
			}
		} else {
			msURL, err := playlistURL.Parse(v.URI)
			if err != nil {
				continue
			}

			segmentURI, err = url.QueryUnescape(msURL.String())
			if err != nil {
				return nil, err
			}
		}
		urls = append(urls, segmentURI)
	}

	return urls, nil
}

func DownloadSegments(u, output string, thread int) error {
	//读取ts文件列表
	urls, err := BuildSegments(u)

	if err != nil {
		return err
	}

	Info.Printf("m3u8:%s, ts:%d\n", u, len(urls))

	if len(urls) == 0 {
		return nil
	}

	limiter := make(chan bool, thread)
	for k, u := range urls {

		if IsFile(killSignal) {
			Warning.Println(killSignal + " exists, waiting job finish and go to kill")
			return nil
		}

		wg.Add(1)

		limiter <- true
		go tsDownload(u, output, k, limiter)

	}

	wg.Wait()
	return nil
}

//多线程下载ts文件
func tsDownload(tsFile string, savePath string, jobId int, limiter chan bool) bool {
	defer wg.Done()

	//根据URL原来的路径保存文件，如果文件存在，就跳过
	uri, _ := url.Parse(tsFile)
	newPath := savePath + "/" + path.Dir(uri.Path)
	file := fmt.Sprintf("%s/%s", newPath, path.Base(uri.Path))

	if IsFile(file) {
		Info.Println(file + " exists, ignore.")
		<-limiter
	}

	//开始时间
	s := time.Now().Unix()

	res, err := http.Get(tsFile)
	time.Sleep(time.Second * 2)

	<-limiter

	if err != nil {
		Warning.Printf("url:%s, error:%s\n", tsFile, err)
		return false
	}

	if res.StatusCode != 200 {
		Warning.Printf("url:%s, error:%d\n", tsFile, res.StatusCode)
		return false
	}

	//创建文件目录
	os.MkdirAll(newPath, os.ModePerm)
	Info.Printf("tsid:%d, ts:%s, save to:%s, size:%d, use time:%d s\n", jobId, uri, file, res.ContentLength, time.Now().Unix()-s)

	out, err := os.Create(file)
	if err != nil {
		Warning.Printf("url:%s, create file error:%s\n", tsFile, err)
		return false
	}
	defer out.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		Warning.Printf("url:%s, write file error:%s", tsFile, err)
		return false
	}

	return true
}

// Download hls segments into a single output file based on the remote index
func Download(u, output string, thread int) error {

	if IsFile(killSignal) {
		log.Fatal(killSignal + " exists, terminated.")
		return nil
	}

	err := DownloadSegments(u, output, thread)
	if err != nil {
		return err
	}

	return nil
}
