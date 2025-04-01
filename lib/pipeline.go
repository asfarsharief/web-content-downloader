package lib

import "web-content-downloader/logger"

type PipelineStruct struct {
	filePath string
}

type PipelineInterface interface {
	TriggerPipeline()
}

func NewPipelineStruct(path string) PipelineInterface {
	return &PipelineStruct{
		filePath: path,
	}
}

func (ps *PipelineStruct) TriggerPipeline() {
	logger.Infof("Pipeline triggered: File: %s", ps.filePath)
}
