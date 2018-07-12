package main

import (
	"log"
	"os"
	"runtime"

	"github.com/urfave/cli"
	"github.com/vus520/go-hls/hls"
)

// DownloadCli is the cli wrapper for download functionality
func DownloadCli(c *cli.Context) error {
	if c.String("url") == "" {
		log.Fatal("url required")
	}

	if c.String("output") == "" {
		log.Fatal("output required")
	}

	return hls.Download(c.String("url"), c.String("output"))
}

func main() {

	runtime.GOMAXPROCS(1)

	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "url",
			Usage: "m3u8 media url",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "path to output file",
			Value: "output",
		},
		cli.IntFlag{
			Name:  "thread",
			Usage: "threads for download",
			Value: 1,
		},
	}

	app.Action = DownloadCli

	app.Run(os.Args)
}
