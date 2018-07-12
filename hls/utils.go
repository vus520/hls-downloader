/**
 * A few function in php
 */

package hls

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

//检查文件是否存在
func IsFile(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

//检查目录是否存在
func IsDir(path string) bool {
	p, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	} else {
		return p.IsDir()
	}
}

//写内容到文件
func FilePutContents(filename string, content string) (val bool, err error) {
	if IsFile(filename) {
		return
	}

	fout, err := os.Create(string(filename))

	if err != nil {
		return false, err
	}

	defer fout.Close()

	fout.WriteString(content)

	return true, nil
}

func FileGetContents(uri string) (str string, err error) {
	url := strings.ToLower(strings.TrimSpace(uri))

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		resp, err := http.Get(url)

		if err != nil {
			return "", err
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return "", errors.New(resp.Status)
		}

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return "", err
		}

		return string(body), nil
	}

	data, err := ioutil.ReadFile(uri)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func Unlink(uri string) error {
	if IsFile(uri) {
		err := os.Remove(uri)
		return err
	}

	return nil
}
