package main

import (
	"os"

	"web-content-downloader/lib"
	"web-content-downloader/logger"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "Web Content Downloader Pipeline"
	app.Usage = "Triggers pipeline to url's and download it's content"
	app.Version = "latest"

	app.Commands = []*cli.Command{
		{
			Name:      "trigger",
			Usage:     "Triggers the pipeline",
			UsageText: "trigger -p filePath",
			Action:    RunPipeline,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "Path",
					Aliases:  []string{"p", "PATH"},
					Usage:    "Full path of csv file",
					Required: true,
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Errorf("%s", err)
		os.Exit(1)
	}
}

// RunPipeline - Function that will run the server
func RunPipeline(c *cli.Context) error {
	csvPath := c.String("PATH")
	pipelineStruct := lib.NewPipelineStruct(csvPath)

	pipelineStruct.TriggerPipeline()
	return nil
}
