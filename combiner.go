/*
Copyright 2020 Brian Pursley

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bufio"
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

func main() {
	ctx := context.Background()

	if len(os.Args) != 2 {
		log.Fatalf("missing url argument")
	}
	url := os.Args[1]

	client, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		log.Fatalf("failed to create storage client: %v", err)
	}
	defer client.Close()

	bucketName := "kubernetes-jenkins"
	var prefix string
	if strings.Contains(url, bucketName) {
		prefix = strings.Split(url, "/"+bucketName+"/")[1]
	} else {
		log.Fatalf("unable to determine prefix from the specified url")
	}

	bucket := client.Bucket(bucketName)
	objectNames, err := getObjectNames(ctx, bucket, prefix)
	if err != nil {
		log.Fatalf("failed to get object names from bucket %v with prefix %v: %v", bucket, prefix, err)
	}

	resultChan := make(chan []string, 16)
	errorChan := make(chan error)

	for i, name := range objectNames {
		go func(i int, name string) {
			reader, err := bucket.Object(name).NewReader(ctx)
			if err != nil {
				errorChan <- fmt.Errorf("failed to create new reader for %v: %v", name, err)
			}
			defer reader.Close()
			scanner := bufio.NewScanner(reader)
			maxTokenSize := 32 * 1024 * 1024
			buf := make([]byte, 0, maxTokenSize)
			scanner.Buffer(buf, maxTokenSize)
			scanner.Split(bufio.ScanLines)

			nameWithoutPrefix := strings.TrimPrefix(name, prefix)
			shortName := shortName(nameWithoutPrefix)

			var lineTime time.Time
			emptyTime := time.Time{}
			firstTime := emptyTime
			dayNumber := 0
			var rowNumber = 0
			var lines []string
			for scanner.Scan() {
				rowNumber++
				line := scanner.Text()
				lineTime, err = parseLineTime(line, lineTime)
				if err != nil {
					errorChan <- fmt.Errorf("unable to parse line time: %v", err)
				}
				if firstTime == emptyTime {
					firstTime = lineTime
				}
				if lineTime.Hour() < firstTime.Hour()-1 {
					dayNumber = 1
				}

				sortKey := fmt.Sprintf("%d:%02d:%02d:%02d.%09d:%04d:%08d", dayNumber, lineTime.Hour(), lineTime.Minute(), lineTime.Second(), lineTime.Nanosecond(), i, rowNumber)
				displayTime := fmt.Sprintf("%02d:%02d:%02d.%09d", lineTime.Hour(), lineTime.Minute(), lineTime.Second(), lineTime.Nanosecond())
				lines = append(lines, fmt.Sprintf("%s %s %-62s %s", sortKey, displayTime, "["+shortName+"]", line))
			}
			if scanner.Err() != nil {
				errorChan <- scanner.Err()
			}

			resultChan <- lines
		}(i, name)
	}

	var combinedLines []string
	for i := 0; i < len(objectNames); i++ {
		select {
		case lines := <-resultChan:
			combinedLines = append(combinedLines, lines...)
		case err := <-errorChan:
			log.Fatal(err)
		}
	}

	sort.Strings(combinedLines)

	bw := bufio.NewWriter(os.Stdout)
	defer bw.Flush()
	for _, line := range combinedLines {
		// Write line to output without sort key (first 35 chars)
		if _, err := bw.WriteString(line[35:] + "\n"); err != nil {
			log.Fatalf("failed to write string: %v", err)
		}
	}
}

func getObjectNames(ctx context.Context, bucket *storage.BucketHandle, prefix string) ([]string, error) {
	q := &storage.Query{Prefix: prefix}
	if err := q.SetAttrSelection([]string{"Name"}); err != nil {
		return nil, fmt.Errorf("failed to set attr selection: %v", err)
	}
	objects := bucket.Objects(ctx, q)
	var objectNames []string
	for {
		objAttrs, err := objects.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterator error: %v", err)
		}
		if strings.HasSuffix(objAttrs.Name, ".log") || strings.HasSuffix(objAttrs.Name, "build-log.txt") {
			objectNames = append(objectNames, objAttrs.Name)
		}
	}
	return objectNames, nil
}

var timeNanoPattern = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}.\d{9})`)  // Example: 22:10:34.002031939
var timeMicroPattern = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}.\d{6})`) // Example: 22:10:34.002031
var timeMilliPattern = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}.\d{3})`) // Example: 22:10:34.002
var timePattern = regexp.MustCompile(`(\d{2}:\d{2}:\d{2})`)            // Example: 22:10:34

const (
	timeNanoLayout  = "15:04:05.000000000"
	timeMicroLayout = "15:04:05.000000"
	timeMilliLayout = "15:04:05.000"
	timeLayout      = "15:04:05"
)

func parseLineTime(line string, defaultValue time.Time) (time.Time, error) {
	if match := timeNanoPattern.FindStringSubmatch(line); match != nil {
		return time.Parse(timeNanoLayout, match[1])
	}
	if match := timeMicroPattern.FindStringSubmatch(line); match != nil {
		return time.Parse(timeMicroLayout, match[1])
	}
	if match := timeMilliPattern.FindStringSubmatch(line); match != nil {
		return time.Parse(timeMilliLayout, match[1])
	}
	if match := timePattern.FindStringSubmatch(line); match != nil {
		return time.Parse(timeLayout, match[1])
	}
	return defaultValue, nil
}

func shortName(name string) string {
	if len(name) > 60 {
		return name[:17] + "..." + name[len(name)-40:]
	} else {
		return name
	}
}
