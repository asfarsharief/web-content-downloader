package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"web-content-downloader/pkg/constants"
	"web-content-downloader/pkg/logger"
)

type PipelineStruct struct {
	filePath     string
	fileReader   io.Reader
	sem          chan struct{}
	urlChannel   chan UrlData
	dataChannel  chan UrlData
	errorChannel chan UrlData
	waitGroup    *sync.WaitGroup
	errorList    [][]string
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
		sem:          make(chan struct{}, constants.MaxWorkers),
		urlChannel:   make(chan UrlData),
		dataChannel:  make(chan UrlData),
		errorChannel: make(chan UrlData),
		waitGroup:    &sync.WaitGroup{},
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
	ps.fileReader = file
	fmt.Println("wg start")
	ps.waitGroup.Add(1)

	// Start pipeline
	go ps.ReadAndProcessUrlsFromCsv()
	go ps.ProcessUrls()
	go ps.PersistContent()
	go ps.HandleErrors()

	// Wait for all goroutines to finish
	fmt.Println("waiting wg")
	ps.waitGroup.Wait()

	fmt.Println("here?")
	// Close channels after everything has finished
	// close(ps.urlChannel)
	close(ps.dataChannel)
	close(ps.errorChannel)

	if len(ps.errorList) > 0 {
		csvFile, err := os.Create("./store/failedUrls.csv")
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer csvFile.Close()

		writer := csv.NewWriter(csvFile)
		writer.Write([]string{"Index", "Url", "Error"})
		fmt.Println("Writing error")
		// Write data to CSV file
		for _, row := range ps.errorList {
			fmt.Println(row)
			err := writer.Write(row)
			if err != nil {
				fmt.Println("Error writing record to file:", err)
				return
			}
		}
	}
}

func (ps *PipelineStruct) ReadAndProcessUrlsFromCsv() {
	defer func() {
		close(ps.urlChannel)
		fmt.Println("wg start")
		ps.waitGroup.Done()
	}()
	// Create a new CSV reader
	reader := csv.NewReader(ps.fileReader)
	isFirst := true
	// Read CSV row by row
	index := 1
	for {
		fmt.Println("reading: ", index)
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			// Handle other read errors
			logger.Error("Error reading CSV:", err)
			continue
		}
		if isFirst {
			isFirst = false
			continue
		}
		// Reserve a slot for concurrent processing
		ps.sem <- struct{}{}
		fmt.Println(fmt.Println("wg ", index))
		ps.waitGroup.Add(1)
		fmt.Println("pushing: ", index)
		ps.urlChannel <- UrlData{
			index: index,
			url:   record[0],
		} // Send the URL to the channel for processing
		index++
	}
}

func (ps *PipelineStruct) ProcessUrls() {
	// Read from URL channel
	for data := range ps.urlChannel {
		fmt.Println("Processing URL:", data.index, data.url)

		resp, err := http.Get(data.url)
		if err != nil {
			logger.Error("Error making GET request:", err)
			data.err = err
			ps.errorChannel <- data
			<-ps.sem
			fmt.Println("wg ", data.index)
			ps.waitGroup.Done()
			continue
		}
		defer resp.Body.Close() // Ensure that the response body is closed after use

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Error reading response body:", err)
			data.err = err
			ps.errorChannel <- data
			<-ps.sem
			fmt.Println("wg ", data.index)
			ps.waitGroup.Done()
			continue
		}

		// Send data to the data channel
		data.data = string(body)
		ps.dataChannel <- data
		<-ps.sem // Release the semaphore slot after processing
	}
	fmt.Println("Existing from processUrls...")
}

func (ps *PipelineStruct) PersistContent() {
	// Persist content (process data channel)
	for data := range ps.dataChannel {
		fmt.Println("Persisting:", data.index, data.url)
		// uniqueKey := uuid.New()
		file, err := os.Create(fmt.Sprintf("./store/%v.txt", data.index))
		if err != nil {
			fmt.Println("Error creating file:", err)
			data.err = err
			ps.errorChannel <- data
			fmt.Println("wg ", data.index)
			ps.waitGroup.Done()
			continue
		}
		defer file.Close() // Make sure to close the file when we're done

		// Write the data to the file
		_, err = file.WriteString(data.data)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			data.err = err
			ps.errorChannel <- data
			fmt.Println("wg ", data.index)
			ps.waitGroup.Done()
			continue
		}
		fmt.Println("wg ", data.index)
		ps.waitGroup.Done()
	}
	fmt.Println("Existing PersistContent...")
}

func (ps *PipelineStruct) HandleErrors() {
	for errData := range ps.errorChannel {
		fmt.Println("Error encountered: ", errData.index)
		ps.errorList = append(ps.errorList, []string{string(errData.index), errData.url, errData.err.Error()})
	}
}
