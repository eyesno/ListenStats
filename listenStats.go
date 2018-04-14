package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Info struct {
	ReleaseMBID   string   `json:"release_mbid,omitempty"`
	ArtistMBIDs   []string `json:"artist_mbids,omitempty"`
	RecordingMBID string   `json:"recording_mbid,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	MSID          string   `json:"artist_msid,omitempty"`
	ArtistName    string   `json:"artist_name,omitempty"`
	ISRC          string   `json:"isrc,omitempty"`

	ReleaseGroupMBID string `json:"release_group_mbid,omitempty"`

	ReleaseMSID string `json:"release_msid,omitempty"`
	ReleaseName string `json:"release_name,omitempty"`
	SpotifyId   string `json:"spotify_id,omitempty"`

	TrackMBID   string   `json:"track_mbid,omitempty"`
	TrackName   string   `json:"track_name,omitempty"`
	TrackNumber string   `json:"tracknumber,omitempty"`
	WorkMBIDs   []string `json:"work_mbids,omitempty"`
}

type MetaData struct {
	AdditionalInfo Info   `json:"additional_info,omitempty"`
	ArtistName     string `json:"artist_name,omitempty"`
	TrackName      string `json:"track_name,omitempty"`
}

type Listen struct {
	ListenTime    int64    `json:"listened_at,omitempty"`
	RecordingMSID string   `json:"recording_msid,omitempty"`
	TrackMetaData MetaData `json:"track_metadata,omitempty"`
}

type PayloadData struct {
	Count   int      `json:"count"`
	Listens []Listen `json:"listens"`
}

type Response struct {
	Payload PayloadData `json:"payload"`
}

type ListenUnit struct {
	ArtistName string
	TrackName  string
	Count      int
}

// Base URL constant
const LISTENBRAINZ_BASE_URL = "https://api.listenbrainz.org"
const LISTENBRAINZ_GET_LISTENS = "/1/user/{u}/listens"
const LISTENBRAINZ_USER_TOKEN = "8b31923a-919f-4e64-ab8e-0a6d75ddcda0"
const LISTENBRAINZ_USER_NAME = "punkscience"

var allListens PayloadData

func fetchListensFromTime(w http.ResponseWriter, minTime int64) int64 {
	var lastTime int64 = 0
	client := &http.Client{}

	request := LISTENBRAINZ_BASE_URL + LISTENBRAINZ_GET_LISTENS
	request = strings.Replace(request, "{u}", LISTENBRAINZ_USER_NAME, -1)

	req, err := http.NewRequest("GET", request, nil)

	req.Header.Add("Authorization", LISTENBRAINZ_USER_TOKEN)
	q := req.URL.Query()

	timestamp := fmt.Sprintf("%d", minTime)
	count := fmt.Sprintf("%d", 100)

	q.Set("min_ts", timestamp)
	q.Set("count", count)

	req.URL.RawQuery = q.Encode()
	fmt.Printf("%s", req.URL)

	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		defer resp.Body.Close()

		var serverRsp Response

		decodeJson := json.NewDecoder(resp.Body)

		err = decodeJson.Decode(&serverRsp)

		if err != nil {
			log.Fatal(err)
		}

		//		fmt.Fprintf(w, "There are %d listens.\n", serverRsp.Payload.Count)

		// Make sure we add up the total listens
		for i := 0; i < serverRsp.Payload.Count; i++ {
			allListens.Listens = append(allListens.Listens, serverRsp.Payload.Listens[i])
			allListens.Count++

			//listenTime := time.Unix(serverRsp.Payload.Listens[i].ListenTime, 0)
			lastTime = serverRsp.Payload.Listens[i].ListenTime

			// fmt.Fprintf(w, "%s %s - %s\n",
			// 	listenTime.Format("3:04PM"),
			// 	serverRsp.Payload.Listens[i].TrackMetaData.ArtistName,
			// 	serverRsp.Payload.Listens[i].TrackMetaData.TrackName)
		}
	}

	return lastTime
}

func getWeeklyStats(w http.ResponseWriter) {
	tNow := time.Now()
	lastTime := tNow.AddDate(0, 0, -7).Unix()

	// Get all the listens for the last 7 days
	for lastTime != 0 {
		lastTime = fetchListensFromTime(w, lastTime)
	}

	var listenIndex []ListenUnit

	// Iterate through the listens and give a count to anything unique
	for i := 0; i < len(allListens.Listens); i++ {
		artist := allListens.Listens[i].TrackMetaData.ArtistName
		track := allListens.Listens[i].TrackMetaData.TrackName

		if !alreadyTracked(listenIndex, artist, track) {
			count := countInList(allListens.Listens, artist, track)

			var listen ListenUnit
			listen.ArtistName = allListens.Listens[i].TrackMetaData.ArtistName
			listen.TrackName = allListens.Listens[i].TrackMetaData.TrackName
			listen.Count = count
			listenIndex = append(listenIndex, listen)
			//fmt.Fprintf(w, "%s - %s (%d)\n", artist, track, count)
		}
	}

	fmt.Fprintf(w, "In the past 7 days, you've listened to %d tracks. Nice one.\n\n", len(allListens.Listens))

	// Sort'em...
	sort.Slice(listenIndex[:], func(i, j int) bool {
		return listenIndex[i].Count > listenIndex[j].Count
	})

	// Write'em
	for i := 0; i < 10; i++ {
		fmt.Fprintf(w, "%s - %s\n", listenIndex[i].ArtistName,
			listenIndex[i].TrackName)
	}
}

func alreadyTracked(list []ListenUnit, artist string, track string) bool {
	for i := 0; i < len(list); i++ {
		if strings.ToLower(list[i].ArtistName) == strings.ToLower(artist) &&
			strings.ToLower(list[i].TrackName) == strings.ToLower(track) {
			return true
		}
	}
	return false
}

func countInList(list []Listen, artist string, track string) int {
	count := 0
	for i := 0; i < len(list); i++ {
		if strings.ToLower(list[i].TrackMetaData.ArtistName) == strings.ToLower(artist) &&
			strings.ToLower(list[i].TrackMetaData.TrackName) == strings.ToLower(track) {
			count++
		}
	}

	return count
}

func handler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path[1:] == "listenerStats" {
		getWeeklyStats(w)

		return
	}

	fmt.Fprintf(w, "I am not sure what you're looking for, but you're not going to find it here.")

}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
