package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

const url string = "http://localhost:8080"
const byteSteps int = 5

var wg sync.WaitGroup

func main() {
	bestAPostion := 0
	validResult := false
	mtx := sync.Mutex{}
	//Get main HTML
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	htmlBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(htmlBody))

	//Get file url-s
	fileCounter := 0
	for i := 1; ; i++ {
		if strings.Contains(string(htmlBody), "file"+strconv.Itoa(i)) {
			fmt.Printf("File%d found\n", i)
			fileCounter++
		} else {
			break
		}
	}

	//Download files
	localIndexes := make([]int, fileCounter)
	for i := 1; i <= fileCounter; i++ {
		wg.Add(1)
		go func(fileId int) {

			fileName := "file" + strconv.Itoa(fileId) + ".txt"
			fileUrl := url + "/" + fileName
			bestLocalA := -1

			req, err := http.NewRequest("GET", fileUrl, nil)
			if err != nil {
				panic(err)
			}
			out, err := os.Create(fileName)
			if err != nil {
				panic(err)
			}
			defer out.Close()
			for i := 0; ; i += byteSteps {
				fmt.Println("asd")
				rangeStr := "bytes=" + strconv.Itoa(i) + "-" + strconv.Itoa(i+byteSteps)
				req.Header.Set("Range", rangeStr)
				res, e := new(http.Client).Do(req)
				if e != nil {
					panic(e)
				}
				defer res.Body.Close()

				StreamBytesPart, err := io.ReadAll(res.Body)
				if err != nil {
					panic(err)
				}

				StreamStringPart := string(StreamBytesPart)
				fmt.Println(StreamStringPart)
				mtx.Lock()
				if strings.Index(StreamStringPart, "A") != -1 && bestLocalA == -1 {
					bestLocalA = i + strings.Index(StreamStringPart, "A")
					if !validResult || validResult && bestLocalA <= bestAPostion {
						validResult = true
						bestAPostion = bestLocalA
					}
				}
				if bestLocalA == -1 && i > bestAPostion && validResult || bestLocalA != -1 && bestAPostion < bestLocalA && validResult {
					bestLocalA = -1
					mtx.Unlock()
					break
				}
				mtx.Unlock()
				if strings.Compare(StreamStringPart, "invalid range: failed to overlap\n") == 0 {
					break
				}
				if _, err := out.WriteString(StreamStringPart); err != nil {
					log.Fatal(err)
				}
			}
			mtx.Lock()
			localIndexes[fileId-1] = bestLocalA
			mtx.Unlock()
			wg.Done()
		}(i)
	}
	//Wait for all routines to finish
	wg.Wait()

	//Delete the unnecessary file chunks and files
	fmt.Printf("Best Global position is: %d\n", bestAPostion)
	fmt.Println(localIndexes)
	for i := 1; i <= fileCounter; i++ {
		if localIndexes[i-1] == -1 || localIndexes[i-1] < bestAPostion {
			fileName := "file" + strconv.Itoa(i) + ".txt"
			fmt.Printf("Remove: %s\n", fileName)
			err := os.Remove(fileName)
			if err != nil {
				panic(err)
			}
		}
	}

}
