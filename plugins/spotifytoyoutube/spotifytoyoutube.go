package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	p                 *plugins.Plugin
	spotifyRegex      = regexp.MustCompile(`https?://open\.spotify\.com/track/[a-zA-Z0-9][^\s]{2,}`)
	spotifyTitleRegex = regexp.MustCompile(`(.*) - song( and lyrics)? by (.*) \| Spotify`)

	instances InvidiousInstanceResponse
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Spotify to YouTube",
		Description: "Turns Spotify links into YouTube links",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          YoutubeCommand,
			FnName:      "YoutubeCommand",
			Name:        "youtube",
			Aliases:     []string{"yt"},
			Description: "Search YouTube for a video!",
		}},
		Responses: []bot.ResponseInfo{{
			Fn:       SpotifyToYoutubeResponse,
			Regexes:  []string{spotifyRegex.String()},
			MatchMin: 1,
		}},
		Jobs: []bot.JobInfo{{
			Fn:        func() { updateInstances("hourly job") },
			Tag:       "invidious-instances-update",
			Scheduler: bot.Scheduler.Every(1).Hour(),
		}},
	}
	return p
}

type InvidiousInstance struct {
	Flag   string `json:"flag"`
	Region string `json:"region"`
	API    bool   `json:"api"`
	URI    string `json:"uri"`
}

type InvidiousInstanceResponse [][]InvidiousInstance

type SearchResult struct {
	Type  string `json:"type"`
	ID    string `json:"videoId"`
	Title string `json:"title"`
}

func (r SearchResult) String() string {
	return fmt.Sprintf("[%s, %s, %s]", r.Type, r.ID, r.Title)
}

func YoutubeCommand(c bot.Command) error {
	args, _ := cmd.ParseStringSliceArg(c.Args, 1, -1)
	s := strings.Join(args, " ")
	if len(s) == 0 {
		return bot.GenericSyntaxError("YoutubeCommand", s, "expected video title")
	}

	searchResult, err := queryYoutube(s, true)
	if err != nil {
		return err
	}

	if searchResult == nil {
		_, err = cmd.SendEmbed(c.E, p.Name, "Error: No search results found", bot.ErrorColor)
		return err
	}

	_, err = cmd.SendMessage(c.E, "https://youtu.be/"+searchResult.ID)
	return err
}

func SpotifyToYoutubeResponse(r bot.Response) {
	// Get the Spotify link from the message
	//

	spotifyUrl := spotifyRegex.FindStringSubmatch(r.E.Content)
	if len(spotifyUrl) == 0 {
		_, _ = cmd.SendEmbed(r.E, p.Name, "Error: Couldn't find Spotify link in message", bot.ErrorColor)
		return
	}

	// Get Artist and Song Title from Spotify
	//

	content, resp, err := util.RequestUrl(spotifyUrl[0], http.MethodGet)
	if err != nil {
		_, _ = cmd.SendEmbed(r.E, p.Name, "Error: "+err.Error(), bot.ErrorColor)
		return
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = cmd.SendEmbed(r.E, p.Name, "Error: Spotify returned a `"+strconv.Itoa(resp.StatusCode)+"` status code, expected `200`", bot.ErrorColor)
		return
	}

	node, err := util.ExtractNode(string(content), func(node *html.Node) bool { return node.Data == "title" && node.FirstChild.Data != "Spotify" })
	if err != nil {
		_, _ = cmd.SendEmbed(r.E, p.Name, "Error: "+err.Error(), bot.ErrorColor)
		return
	}

	text := &bytes.Buffer{}
	util.ExtractNodeText(node, text)
	log.Printf("SpotifyToYoutube: text: %s\n", text.String())

	res := spotifyTitleRegex.FindStringSubmatch(text.String())
	if len(res) == 0 {
		_, _ = cmd.SendEmbed(r.E, p.Name, "Error: Couldn't parse Spotify song title", bot.ErrorColor)
		return
	}

	log.Printf("SpotifyToYoutube: res: [%s]\n", strings.Join(res, ", "))

	if len(res) != 4 {
		_, _ = cmd.SendEmbed(r.E, p.Name, "Error: `res` is not 4: `["+strings.Join(res, ", ")+"]`", bot.ErrorColor)
		return
	}

	artistAndSong := res[3] + " - " + res[1] // Artist - Song Title
	searchResult, err := queryYoutube(artistAndSong, true)
	if err != nil {
		_, _ = cmd.SendEmbed(r.E, p.Name, "Error:\n"+err.Error(), bot.ErrorColor)
		return
	}

	if searchResult == nil {
		_, _ = cmd.SendEmbed(r.E, p.Name, "Error: No search results found", bot.ErrorColor)
		return
	}

	_, _ = cmd.SendMessage(r.E, "https://youtu.be/"+searchResult.ID)
}

func queryYoutube(query string, firstRun bool) (*SearchResult, error) {
	if instancesEmpty() {
		updateInstances("queryYoutube called")
	}

	// Make list of instances to query
	//

	query = url.PathEscape(strings.ReplaceAll(query, "\"", "")) // remove quotes and path escape
	searchQuery := "/api/v1/search?q=" + query
	searchUrls := make([]string, 0)

	for _, instance := range instances {
		// instance[0] is the instance URI, instance[1] is the object with said instance's info
		if instance[1].API == true {
			searchUrls = append(searchUrls, instance[1].URI+searchQuery) // this will use more memory but reduces code complexity
		}
	}
	if len(searchUrls) == 0 {
		updateInstances("queryYoutube searchUrls == 0")
		if firstRun {
			return queryYoutube(query, false)
		}
		return nil, bot.GenericError("queryYoutube", "Searching query", "No Invidious instances found")
	}

	log.Printf("queryYoutube: searchUrls %s\n", searchUrls)

	// Query all available search URLs
	//

	content := util.RequestUrlRetry(searchUrls, http.MethodGet, http.StatusOK)
	if content == nil {
		return nil, bot.GenericError("queryYoutube", "Searching `searchUrls`", "nil response received")
	}

	// Parse returned YouTube result
	//

	var searchResults []SearchResult
	err := json.Unmarshal(content, &searchResults)
	if err != nil {
		return nil, err
	}

	var searchResult *SearchResult = nil
	// pick first result with Type "video"
	for _, r := range searchResults {
		if r.Type != "video" {
			continue
		}
		searchResult = &r
		break
	}

	log.Printf("queryYoutube: searchResult %s\n", searchResult)

	return searchResult, nil
}

func updateInstances(source string) {
	log.Printf("updateInstances: updating because: %s\n", source)

	getInstancesFn := func() ([]byte, error) {
		b, _, err := util.RequestUrl("https://api.invidious.io/instances.json?sort_by=users,health", http.MethodGet)
		return b, err
	}

	instancesStr, err := util.RetryFunc(getInstancesFn, 2, 300) // This will take a max of ~16 seconds to execute, with a 5s timeout
	if err != nil {
		log.Printf("updateInstances: %v\n", err)
	}

	// For some reason this will always error even though it gives the expected result
	_ = json.Unmarshal(instancesStr, &instances)
}

func instancesEmpty() bool {
	found := false
	for _, instance := range instances {
		log.Printf("instance: %v\n", instance)
		if instance[1].API == true {
			found = true
			break
		}
	}
	return !found
}
