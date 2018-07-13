package main

import (
	"log"
	"os"
	"runtime"

	"github.com/urfave/cli"
	"github.com/vus520/go-hls/hls"
)

func DownloadCli(c *cli.Context) error {
	if c.String("url") == "" {
		log.Fatal("a m3u8 url or file is required")
	}

	if c.String("output") == "" {
		log.Fatal("output dir is srequired")
	}

	runtime.GOMAXPROCS(c.Int("procs"))

	return hls.Download(c.String("url"), c.String("output"), c.Int("thread"))
}

func main() {
	app := cli.NewApp()

	app.Name = "m3u8 downloader"
	app.Usage = "A m3u8 playlist downloader in multithreading, touch /tmp/dlm3u8.stop to stop all the process."
	app.Version = "20180713"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "url",
			Usage: "m3u8 media url",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "dir to output file",
			Value: "output/",
		},
		cli.IntFlag{
			Name:  "procs",
			Usage: "procs for procs.GOMAXPROCS",
			Value: 1,
		},
		cli.IntFlag{
			Name:  "thread",
			Usage: "threads for download",
			Value: 10,
		},
	}

	app.Action = DownloadCli
	app.Run(os.Args)
}
