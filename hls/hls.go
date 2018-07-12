package hls

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/grafov/m3u8"
)

var wg sync.WaitGroup

// GetPlaylist fetch content from remote url and return a list of segments
func GetPlaylist(url string) (*m3u8.MediaPlaylist, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, nil
	}

	playlist, listType, err := m3u8.DecodeFrom(res.Body, true)
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

func DownloadSegments(u, output string) error {
	//读取ts文件列表
	urls, err := BuildSegments(u)

	if err != nil {
		return err
	}

	fmt.Printf("m3u8:%s, ts:%d\n", u, len(urls))

	if len(urls) == 0 {
		return nil
	}

	limiter := make(chan bool, 10)
	for k, u := range urls {
		wg.Add(1)
		go tsDownload(u, output, k, limiter)
		limiter <- true
	}

	wg.Wait()
	return nil
}

//多线程下载ts文件
func tsDownload(tsFile string, savePath string, jobId int, limiter chan bool) bool {
	defer wg.Done()

	res, err := http.Get(tsFile)
	time.Sleep(time.Second * 1)

	<-limiter

	if err != nil {
		fmt.Printf("url:%s, error:%s\n", tsFile, err)
		return false
	}

	if res.StatusCode != 200 {
		fmt.Printf("url:%s, error:%d\n", tsFile, res.StatusCode)
		return false
	}

	//保留原文件路径
	uri, _ := url.Parse(tsFile)
	newPath := savePath + "/" + path.Dir(uri.Path)
	os.MkdirAll(newPath, os.ModePerm)

	file := fmt.Sprintf("%s/%s", newPath, path.Base(uri.Path))

	fmt.Printf("id:%d, ts:%s, save to:%s\n", jobId, uri, file)

	out, err := os.Create(file)
	if err != nil {
		fmt.Printf("url:%s, create file error:%s\n", tsFile, err)
		return false
	}
	defer out.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		fmt.Printf("url:%s, write file error:%s\n", tsFile, err)
		return false
	}

	return true
}

// Download hls segments into a single output file based on the remote index
func Download(u, output string) error {
	err := DownloadSegments(u, output)
	if err != nil {
		return err
	}

	return nil
}
