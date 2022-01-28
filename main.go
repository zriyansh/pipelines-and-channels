// this is an implementation of pipelining and using channels to pass message.

// Data of an Airport is taken here and we calculate the wind direction at that very airport using METAR data and finally see which direction sees a lot of wind.
// This is a small example of data mining.

package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	windRegex     = regexp.MustCompile(`\d* METAR.*EGLL \d*Z [A-Z ]*(\d{5}KT|VRB\d{2}KT).*=`)
	tafValidation = regexp.MustCompile(`.*TAF.*`)
	comment       = regexp.MustCompile(`\w*#.*`)
	metarClose    = regexp.MustCompile(`.*=`)
	variableWind  = regexp.MustCompile(`.*VRB\d{2}KT`)
	validWind     = regexp.MustCompile(`\d{5}KT`)
	windDirOnly   = regexp.MustCompile(`(\d{3})\d{2}KT`)
	windDist      [8]int
)

func parseToArray(textChannel chan string, metarChannel chan []string) {
	for text := range textChannel {
		lines := strings.Split(text, "\n")
		metarSlice := make([]string, 0, len(lines)) // size is 0, but capacity is len(lines)
		metarStr := ""
		for _, line := range lines {
			if tafValidation.MatchString(line) {
				break
			}
			if !comment.MatchString(line) {
				metarStr += strings.Trim(line, " ")
			}
			if metarClose.MatchString(line) {
				metarSlice = append(metarSlice, metarStr)
				metarStr = ""
			}
		}
		metarChannel <- metarSlice
	}
	close(metarChannel)
}

func extractWindDirection(metarsChannel chan []string, windsChannel chan []string) {
	for metars := range metarsChannel {
		winds := make([]string, 0, len(metars))
		for _, metar := range metars {
			if windRegex.MatchString(metar) {
				winds = append(winds, windRegex.FindAllStringSubmatch(metar, -1)[0][1])
			}
		}
		windsChannel <- winds
	}
	close(windsChannel)
}

func mineWindDistribution(windsChannel chan []string, distChannel chan [8]int) {
	for winds := range windsChannel {
		for _, wind := range winds {
			if variableWind.MatchString(wind) {
				for i := 0; i < 8; i++ {
					windDist[i]++
				}
			} else if validWind.MatchString(wind) {
				windStr := windDirOnly.FindAllStringSubmatch(wind, -1)[0][1]
				if d, err := strconv.ParseFloat(windStr, 64); err == nil {
					dirIndex := int(math.Round(d/45.0)) % 8
					windDist[dirIndex]++
				}
			}
		}
	}
	distChannel <- windDist
	close(distChannel)
}

func main() {
	textChannel := make(chan string)
	metarChannel := make(chan []string)
	windsChannel := make(chan []string)
	resultChannel := make(chan [8]int)

	// change to array, each metar report is a separate item in the array
	go parseToArray(textChannel, metarChannel)

	// extract wind direction from the metar data code.
	go extractWindDirection(metarChannel, windsChannel)

	// assign to N, NE, E, SE, S, SW, W, NW, 070 means East, so ++ value at that position in int array
	go mineWindDistribution(windsChannel, resultChannel)
	absPath, _ := filepath.Abs("./metarfiles/")
	files, _ := ioutil.ReadDir(absPath)
	for _, file := range files {
		dat, err := ioutil.ReadFile(filepath.Join(absPath, file.Name()))
		if err != nil {
			panic(err)
		}
		text := string(dat)
		textChannel <- text

	}
	close(textChannel)
	results := <-resultChannel
	fmt.Printf("%v ", results)
}
