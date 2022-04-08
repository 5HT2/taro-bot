package main

import (
	"bytes"
	"encoding/json"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	spotifyRegex      = regexp.MustCompile(`https?://open\.spotify\.com/track/[a-zA-Z0-9][^\s]{2,}`)
	spotifyTitleRegex = regexp.MustCompile(`(.*) - song( and lyrics)? by (.*) \| Spotify`)
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "Spotify to YouTube",
		Description: "Turns Spotify links into YouTube links",
		Version:     "1.0.0",
		Commands:    []bot.CommandInfo{},
		Responses: []bot.ResponseInfo{{
			Fn:       SpotifyToYoutubeResponse,
			Embed:    false,
			Regexes:  []string{spotifyRegex.String()},
			MatchMin: 1,
		}},
	}
}

func SpotifyToYoutubeResponse(r bot.Response) string {
	// Get the Spotify link from the message
	//

	spotifyUrl := spotifyRegex.FindStringSubmatch(r.E.Content)
	if len(spotifyUrl) == 0 {
		return "Error: Couldn't find Spotify link in message"
	}

	// Get Artist and Song Title from Spotify
	//

	content, resp, err := util.RequestUrl(spotifyUrl[0], http.MethodGet)
	if err != nil {
		return "Error: " + err.Error()
	}
	if resp.StatusCode != http.StatusOK {
		return "Error: Spotify returned a `" + strconv.Itoa(resp.StatusCode) + "` status code, expected `200`"
	}

	node, err := util.ExtractNode(string(content), func(str string) bool { return str == "title" })
	if err != nil {
		return "Error: " + err.Error()
	}

	text := &bytes.Buffer{}
	util.ExtractNodeText(node, text)
	log.Printf("SpotifyToYoutube: text: %s\n", text.String())

	res := spotifyTitleRegex.FindStringSubmatch(text.String())
	if len(res) == 0 {
		return "Error: Couldn't parse Spotify song title"
	}

	log.Printf("SpotifyToYoutube: res: [%s]\n", strings.Join(res, ", "))

	if len(res) != 4 {
		return "Error: `res` is not 4: `[" + strings.Join(res, ", ") + "]`"
	}

	// Get available instances from invidious
	//

	fn := func() ([]byte, error) {
		b, _, err := util.RequestUrl("https://api.invidious.io/instances.json?sort_by=users,health", http.MethodGet)
		return b, err
	}

	instancesStr, err := util.RetryFunc(fn, 2, 300) // This will take a max of ~16 seconds to execute, with a 5s timeout
	if err != nil {
		return "Error: " + err.Error()
	}

	type InvidiousInstance struct {
		Flag   string `json:"flag"`
		Region string `json:"region"`
		API    bool   `json:"api"`
		URI    string `json:"uri"`
	}

	type InvidiousInstanceResponse [][]InvidiousInstance
	var instances InvidiousInstanceResponse
	// For some reason this will always error even though it gives the expected result
	_ = json.Unmarshal(instancesStr, &instances)

	// Make list of instances to query
	//

	artistAndSong := strings.ReplaceAll(res[3]+" - "+res[1], "\"", "") // Remove quotes
	searchQuery := "/api/v1/search?q=" + url.PathEscape(artistAndSong) // Artist - Song Title
	searchUrls := make([]string, 0)

	for _, instance := range instances {
		// instance[0] is the instance URI, instance[1] is the object with said instance's info
		if instance[1].API == true {
			searchUrls = append(searchUrls, instance[1].URI+searchQuery) // this will use more memory but reduces code complexity
		}
	}
	if len(searchUrls) == 0 {
		return "Error: Couldn't find any Invidious instance to search with"
	}
	log.Printf("SpotifyToYoutube: searchUrls %s\n", searchUrls)

	// Query all available search URLs
	//

	content = util.RequestUrlRetry(searchUrls, http.MethodGet, http.StatusOK)
	if content == nil {
		return "Error: no non-nil response from `searchUrls`"
	}

	// Parse returned YouTube result
	//

	type YoutubeSearchResult struct {
		Title string `json:"title"`
		ID    string `json:"videoId"`
	}
	var searchResults []YoutubeSearchResult
	err = json.Unmarshal(content, &searchResults)
	if err != nil {
		return "Error: " + err.Error()
	}

	if len(searchResults) == 0 {
		return "Error: No search results found"
	}
	log.Printf("SpotifyToYoutube: searchResults[0] %s\n", searchResults[0])

	return "https://youtu.be/" + searchResults[0].ID
}
