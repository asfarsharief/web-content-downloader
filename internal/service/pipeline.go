package service

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"web-content-downloader/pkg/constants"
	"web-content-downloader/pkg/logger"
)

type PipelineStruct struct {
	filePath      string
	fileReader    io.Reader
	sem           chan struct{}
	urlChannel    chan UrlData
	dataChannel   chan UrlData
	errorChannel  chan UrlData
	matrixChannel chan time.Duration
	waitGroup     *sync.WaitGroup
	errorList     []UrlData
	processed     int
	total         int
}

type UrlData struct {
	index int
	url   string
	data  string
	err   error
}

type PipelineInterface interface {
	TriggerPipeline(filePath string)
	ReadAndProcessUrlsFromCsv()
	ProcessUrls()
	PersistContent()
}

func NewPipelineStruct() PipelineInterface {
	return &PipelineStruct{
		sem:           make(chan struct{}, constants.MaxWorkers),
		urlChannel:    make(chan UrlData),
		dataChannel:   make(chan UrlData),
		errorChannel:  make(chan UrlData),
		matrixChannel: make(chan time.Duration),
		waitGroup:     &sync.WaitGroup{},
	}
}

var activeCount int32 = 1

func (ps *PipelineStruct) progressBar() {
	defer ps.waitGroup.Done()
	ps.waitGroup.Add(1)
	for {
		time.Sleep(500 * time.Millisecond)
		percent := (atomic.LoadInt32(&activeCount) * 100) / int32(ps.total)
		bar := ""
		for i := 0; i < int(percent)/2; i++ {
			bar += "█"
		}
		fmt.Printf("\rProgress: [%-50s] %d%%", bar, percent)
		if atomic.LoadInt32(&activeCount) == int32(ps.total) {
			fmt.Println("\nDownload complete!")
			return
		}
	}
}

func (ps *PipelineStruct) TriggerPipeline(filePath string) {
	logger.Infof("Pipeline triggered: File: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}

	ps.total = lines
	file.Seek(0, 0)
	ps.fileReader = file
	ps.waitGroup.Add(1)

	// Start pipeline
	go ps.ReadAndProcessUrlsFromCsv()
	go ps.ProcessUrls()
	go ps.PersistContent()
	go ps.HandleErrors()
	go ps.progressBar()

	var total time.Duration
	count := 0
	for t := range ps.matrixChannel {
		total += t
		count++
	}

	// Wait for all goroutines to finish
	ps.waitGroup.Wait()

	// Close channels after everything has finished
	close(ps.dataChannel)
	close(ps.errorChannel)

	logger.Infof("Total Urls Processed: %v || Total Success: %v || Total Failed: %v", ps.processed, count, len(ps.errorList))
	logger.Infof("Total time taken: %v || Average Download Time: %v\n", total, total/time.Duration(count))
	if len(ps.errorList) > 0 {
		logger.Info("List of failed Urls with Index and error:")
		for _, err := range ps.errorList {
			fmt.Printf("%d: %v || error: %s \n", err.index, err.url, err.err)
		}
	}
}

func (ps *PipelineStruct) ReadAndProcessUrlsFromCsv() {
	defer func() {
		close(ps.urlChannel)
		ps.waitGroup.Done()
	}()
	logger.Info("Reading file...")
	reader := csv.NewReader(ps.fileReader)
	isFirst := true
	// Read CSV row by row
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			// Handle other read errors
			atomic.AddInt32(&activeCount, 1)
			logger.Error("Error reading CSV:", err)
			continue
		}
		if isFirst {
			isFirst = false
			continue
		}
		// Reserve a slot for concurrent processing
		ps.processed++
		ps.sem <- struct{}{}
		ps.waitGroup.Add(1)
		ps.urlChannel <- UrlData{
			index: ps.processed,
			url:   record[0],
		} // Send the URL to the channel for processing
	}
}

func (ps *PipelineStruct) ProcessUrls() {
	// Read from URL channel
	processUrlWg := sync.WaitGroup{}
	for data := range ps.urlChannel {
		processUrlWg.Add(1)
		go func(data UrlData, processUrlWg *sync.WaitGroup) {
			defer processUrlWg.Done()
			start := time.Now()
			resp, err := http.Get(data.url)
			if err != nil {
				// logger.Error("Error making GET request:", err)
				data.err = err
				ps.errorChannel <- data
				<-ps.sem
				ps.waitGroup.Done()
				atomic.AddInt32(&activeCount, 1)
				return
			}
			defer resp.Body.Close() // Ensure that the response body is closed after use

			// Read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				data.err = err
				ps.errorChannel <- data
				<-ps.sem
				ps.waitGroup.Done()
				atomic.AddInt32(&activeCount, 1)
				return
			}
			// Send data to the data channel
			ps.matrixChannel <- time.Since(start)
			data.data = string(body)
			ps.dataChannel <- data
			<-ps.sem // Release the semaphore slot after processing
		}(data, &processUrlWg)
	}
	processUrlWg.Wait()
	close(ps.matrixChannel)
}

func (ps *PipelineStruct) PersistContent() {
	// Persist content (process data channel)
	for data := range ps.dataChannel {
		// uniqueKey := uuid.New()
		file, err := os.Create(fmt.Sprintf("./store/%v.txt", data.index))
		if err != nil {
			// logger.Error("Error creating file:", err)
			data.err = err
			ps.errorChannel <- data
			ps.waitGroup.Done()
			atomic.AddInt32(&activeCount, 1)
			continue
		}
		defer file.Close() // Make sure to close the file when we're done

		// Write the data to the file
		_, err = file.WriteString(data.data)
		if err != nil {
			// logger.Error("Error writing to file:", err)
			data.err = err
			ps.errorChannel <- data
			ps.waitGroup.Done()
			atomic.AddInt32(&activeCount, 1)
			continue
		}
		ps.waitGroup.Done()
		atomic.AddInt32(&activeCount, 1)
	}
}

func (ps *PipelineStruct) HandleErrors() {
	for errData := range ps.errorChannel {
		ps.errorList = append(ps.errorList, errData)
	}
}
