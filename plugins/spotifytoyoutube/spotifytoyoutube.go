package main

import (
	"bytes"
	"encoding/json"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"golang.org/x/net/html"
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
			Regexes:  []string{spotifyRegex.String()},
			MatchMin: 1,
		}},
	}
}

func SpotifyToYoutubeResponse(r bot.Response) {
	// Get the Spotify link from the message
	//

	spotifyUrl := spotifyRegex.FindStringSubmatch(r.E.Content)
	if len(spotifyUrl) == 0 {
		_, _ = cmd.SendEmbed(r.E, "", "Error: Couldn't find Spotify link in message", bot.ErrorColor)
		return
	}

	// Get Artist and Song Title from Spotify
	//

	content, resp, err := util.RequestUrl(spotifyUrl[0], http.MethodGet)
	if err != nil {
		_, _ = cmd.SendEmbed(r.E, "", "Error: "+err.Error(), bot.ErrorColor)
		return
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = cmd.SendEmbed(r.E, "", "Error: Spotify returned a `"+strconv.Itoa(resp.StatusCode)+"` status code, expected `200`", bot.ErrorColor)
		return
	}

	node, err := util.ExtractNode(string(content), func(node *html.Node) bool { return node.Data == "title" && node.FirstChild.Data != "Spotify" })
	if err != nil {
		_, _ = cmd.SendEmbed(r.E, "", "Error: "+err.Error(), bot.ErrorColor)
		return
	}

	text := &bytes.Buffer{}
	util.ExtractNodeText(node, text)
	log.Printf("SpotifyToYoutube: text: %s\n", text.String())

	res := spotifyTitleRegex.FindStringSubmatch(text.String())
	if len(res) == 0 {
		_, _ = cmd.SendEmbed(r.E, "", "Error: Couldn't parse Spotify song title", bot.ErrorColor)
		return
	}

	log.Printf("SpotifyToYoutube: res: [%s]\n", strings.Join(res, ", "))

	if len(res) != 4 {
		_, _ = cmd.SendEmbed(r.E, "", "Error: `res` is not 4: `["+strings.Join(res, ", ")+"]`", bot.ErrorColor)
		return
	}

	// Get available instances from invidious
	//

	fn := func() ([]byte, error) {
		b, _, err := util.RequestUrl("https://api.invidious.io/instances.json?sort_by=users,health", http.MethodGet)
		return b, err
	}

	instancesStr, err := util.RetryFunc(fn, 2, 300) // This will take a max of ~16 seconds to execute, with a 5s timeout
	if err != nil {
		_, _ = cmd.SendEmbed(r.E, "", "Error: "+err.Error(), bot.ErrorColor)
		return
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
		_, _ = cmd.SendEmbed(r.E, "", "Error: Couldn't find any Invidious instance to search with", bot.ErrorColor)
		return
	}
	log.Printf("SpotifyToYoutube: searchUrls %s\n", searchUrls)

	// Query all available search URLs
	//

	content = util.RequestUrlRetry(searchUrls, http.MethodGet, http.StatusOK)
	if content == nil {
		_, _ = cmd.SendEmbed(r.E, "", "Error: no non-nil response from `searchUrls`", bot.ErrorColor)
		return
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
		_, _ = cmd.SendEmbed(r.E, "", "Error: "+err.Error(), bot.ErrorColor)
		return
	}

	if len(searchResults) == 0 {
		_, _ = cmd.SendEmbed(r.E, "", "Error: No search results found", bot.ErrorColor)
		return
	}
	log.Printf("SpotifyToYoutube: searchResults[0] %s\n", searchResults[0])

	_, _ = cmd.SendMessage(r.E, "https://youtu.be/"+searchResults[0].ID)
}
