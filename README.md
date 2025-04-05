# web-content-downloader
A golang application to download url content

## Usage

### Run the application
```
make run PATH={path} OUTPUT={output}
```
- PATH is the full path of the csv to be uploaded
- Output is the name of the directory in which the files will be downloaded. This is optional.
### Build the Go Binaries
```
make build
```

## Testing
- Create a csv file with Urls. 
- Run `make run PATH=fullpath OUTPUT=outputFolder` 

### Development Testing
- `make test` run test locally 