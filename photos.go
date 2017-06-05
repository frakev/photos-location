package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"os"
	"encoding/json"
	"path/filepath"
	"path"
	"strings"
	"flag"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/deckarep/golang-set"
	"github.com/alexcesaro/log/stdlog"
)

var (
	apiKey = flag.String("key", "", "Your Google API Key")
	directory = flag.String("directory", "/", "Directory to search")
	logger = stdlog.GetFromFlags()
)

type GoogleMapsResp struct {
	HTMLAttributions []interface{} `json:"html_attributions"`
	Results []struct {
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			Viewport struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		Icon string `json:"icon"`
		ID string `json:"id"`
		Name string `json:"name"`
		Photos []struct {
			Height int `json:"height"`
			HTMLAttributions []string `json:"html_attributions"`
			PhotoReference string `json:"photo_reference"`
			Width int `json:"width"`
		} `json:"photos,omitempty"`
		PlaceID string `json:"place_id"`
		Reference string `json:"reference"`
		Scope string `json:"scope"`
		Types []string `json:"types"`
		Vicinity string `json:"vicinity"`
		OpeningHours struct {
			OpenNow bool `json:"open_now"`
			WeekdayText []interface{} `json:"weekday_text"`
		} `json:"opening_hours,omitempty"`
		Rating float64 `json:"rating,omitempty"`
	} `json:"results"`
	Status string `json:"status"`
}

func main() {

	if *apiKey == "" {
		logger.Error("Api Key is mandatory!")
		os.Exit(2)
	}
	searchDir := *directory
	var fileTypes = mapset.NewSetFromSlice([]interface{}{".jpg", ".jpeg", ".png", ".bmp"})
	fileList := []string{}
	err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return nil
	})

	if err != nil {
		logger.Error(err)
	}

	for _, file := range fileList {
        fileInfo, err := os.Stat(file)
        if err != nil {
            logger.Error(err)
        }
		if fileInfo.IsDir() {
			continue
		}
		if fileTypes.Contains(strings.ToLower(path.Ext(file))) {
			err := getExif(file)
	        if err != nil {
				logger.Error(err)
				continue
			}
		}
	}
}

func getExif(fname string) error {

	logger.Info("File:",fname)

    f, err := os.Open(fname)
    if err != nil {
        logger.Error(err)
		return err
    }

    exif.RegisterParsers(mknote.All...)

    x, err := exif.Decode(f)
    if err != nil {
        logger.Error(err)
		return err
    }

    camModel, _ := x.Get(exif.Model) // normally, don't ignore errors!
	logger.Debug(camModel)

    focal, _ := x.Get(exif.FocalLength)
    numer, denom, _ := focal.Rat2(0) // retrieve first (only) rat. value
	logger.Debug("Focal:", numer, denom)

    // Two convenience functions exist for date/time taken and GPS coords:
    tm, _ := x.DateTime()
	logger.Debug("Taken: ", tm)

    lat, long, _ := x.LatLong()
	logger.Debug("Latitude, Longitude: ", lat, ", ", long)
	if lat != 0 && long != 0 {
	    getLocation(lat, long)
	} else {
		logger.Warning("Can't get location!")
	}

	return nil
}

func getLocation(lat float64, long float64) {

	client := &http.Client{}
	request := fmt.Sprintf("https://maps.googleapis.com/maps/api/place/nearbysearch/json?location=%g,%g&radius=500&key=%s", lat, long, apiKey)

	req, err := http.NewRequest("GET", request, nil)
	if err != nil {
		logger.Error(err)
	}
	resp, err := client.Do(req)

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var googleResp GoogleMapsResp

    err = json.Unmarshal(body, &googleResp)
    if err != nil {
        logger.Error(err)
    }
	if googleResp.Status == "OK" {
		logger.Info("Location:", googleResp.Results[0].Name)
	}
	if googleResp.Status == "ZERO_RESULTS" {
		logger.Warning("Location not found!")
	}
	if googleResp.Status == "REQUEST_DENIED" {
		logger.Error("Request denied!")
	}
	if googleResp.Status == "INVALID_REQUEST" {
		logger.Warning("Invalid request!")
	}
	if googleResp.Status == "UNKNOWN_ERROR" || googleResp.Status == "" {
		logger.Error("Unknown error!")
	}
	if googleResp.Status == "OVER_QUERY_LIMIT" {
		logger.Debug("Google response:", googleResp.Status)
		logger.Error("Quota exceeded! Please wait several minutes...")
		os.Exit(1)
	}
}
