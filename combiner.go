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
)

var lineFormat1 = regexp.MustCompile(`^\w\d{4} (\d{2}:\d{2}:\d{2}.\d{6})`)                  // Example: I1234 22:10:34.002031
var lineFormat2 = regexp.MustCompile(`^\w\d{4} (\d{2}:\d{2}:\d{2}.\d{3})`)                  // Example: I1234 22:10:34.002
var lineFormat3 = regexp.MustCompile(`^[A-Z][a-z]+ \d+ (\d{2}:\d{2}:\d{2}.\d{6})`)          // Example: Aug 24 22:10:34.000000
var lineFormat4 = regexp.MustCompile(`^[A-Z][a-z]+ \d+ (\d{2}:\d{2}:\d{2}.\d{3})`)          // Example: Aug 24 22:10:34.000
var lineFormat5 = regexp.MustCompile(`^time="\d{4}-\d{2}-\d{2}T(\d{2}:\d{2}:\d{2}.\d{9})Z`) // Example: time="2020-09-01T05:59:37.283814575Z"
var lineFormat6 = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}.\d{6})`)                           // Example: 22:10:34.002031
var lineFormat7 = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}.\d{3})`)                           // Example: 22:10:34.002
var lineFormat8 = regexp.MustCompile(`(\d{2}:\d{2}:\d{2})`)                                 // Example: 22:10:34

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

			dayKey := 0
			noTimeKey := strings.Repeat(" ", 18)
			timeKey := noTimeKey
			firstTimeKey := noTimeKey

			var rowNumber = 0
			var lines []string
			lines = append(lines, fmt.Sprintf("%d:%s:%08d [%04d] --> %s", dayKey, noTimeKey, rowNumber, i, nameWithoutPrefix))
			for scanner.Scan() {
				rowNumber++
				line := scanner.Text()
				timeKey = getTimeKey(line, timeKey)
				if firstTimeKey == noTimeKey {
					firstTimeKey = timeKey
				}
				// TODO: For some log files, this isn't a reliable way to determine if it spans a day...
				//if timeKey < firstTimeKey {
				//	dayKey = 1 // Handle if log file spans midnight
				//} else {
				//	dayKey = 0
				//}
				lines = append(lines, fmt.Sprintf("%d:%s:%08d [%04d] %-62s %s", dayKey, timeKey, rowNumber, i, "["+shortName+"]", line))
			}
			if scanner.Err() != nil {
				log.Fatal(scanner.Err())
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
		// Write line to output without time key (first 30 chars)
		if _, err := bw.WriteString(line[30:] + "\n"); err != nil {
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

func getTimeKey(line, defaultTimeKey string) string {
	if match := lineFormat1.FindStringSubmatch(line); match != nil {
		return match[1] + "000"
	}
	if match := lineFormat2.FindStringSubmatch(line); match != nil {
		return match[1] + "000000"
	}
	if match := lineFormat3.FindStringSubmatch(line); match != nil {
		return match[1] + "000"
	}
	if match := lineFormat4.FindStringSubmatch(line); match != nil {
		return match[1] + "000000"
	}
	if match := lineFormat5.FindStringSubmatch(line); match != nil {
		return match[1]
	}
	if match := lineFormat6.FindStringSubmatch(line); match != nil {
		return match[1] + "000"
	}
	if match := lineFormat7.FindStringSubmatch(line); match != nil {
		return match[1] + "000000"
	}
	if match := lineFormat8.FindStringSubmatch(line); match != nil {
		return match[1] + "000000000"
	}
	return defaultTimeKey
}

func shortName(name string) string {
	if len(name) > 60 {
		return name[:17] + "..." + name[len(name)-40:]
	} else {
		return name
	}
}
