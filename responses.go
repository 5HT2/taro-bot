package main

import (
	"encoding/json"
	"fmt"
	"github.com/diamondburned/arikawa/v3/gateway"
	"net/http"
	"net/url"
	"strings"
)

type Response struct {
	Embed        bool     `json:"embed"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	ReflectFunc  string   `json:"reflect_func,omitempty"`
	Regexes      []string `json:"regexes"`
	MatchMin     int      `json:"match_min,omitempty"`
	LockChannels []int64  `json:"lock_channels,omitempty"`
}

type ResponseReflection struct {
	e *gateway.MessageCreateEvent
}

func (r ResponseReflection) PrefixResponse() []string {
	return []string{GetGuildConfig(int64(r.e.GuildID)).Prefix}
}

func (r ResponseReflection) SpotifyToYoutubeResponse() []string {
	// Get Artist and Song Title from Spotify
	//

	content, _, err := RequestUrl(r.e.Content, http.MethodGet)
	if err != nil {
		return []string{"Error: " + err.Error()}
	}

	node, err := ExtractNode(string(content), func(str string) bool { return str == "title" })
	if err != nil {
		return []string{"Error: " + err.Error()}
	}

	res := strings.Split(strings.TrimPrefix(strings.TrimSuffix(RenderNode(node), " | Spotify</title>"), "<title>"), " - song by ")

	if len(res) < 2 {
		return []string{"Error: `res` is less than 2: `" + fmt.Sprint(res) + "`"}
	}

	// Get available instances from invidious
	//

	instancesStr, _, err := RequestUrl("https://api.invidious.io/instances.json?sort_by=users,health", http.MethodGet)
	if err != nil {
		return []string{"Error: " + err.Error()}
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

	searchQuery := "/api/v1/search?q=" + url.PathEscape(res[1]+" - "+res[0]) // Artist - Song Title
	searchUrls := make([]string, 0)

	for _, instance := range instances {
		// instance[0] is the instance URI, instance[1] is the object with said instance's info
		if instance[1].API == true {
			searchUrls = append(searchUrls, instance[1].URI+searchQuery) // this will use more memory but reduces code complexity
		}
	}
	if len(searchUrls) == 0 {
		return []string{"Error: Couldn't find any Invidious instance to search with"}
	}

	// Query all available search URLs
	//

	content = RequestUrlRetry(searchUrls, http.MethodGet, http.StatusOK)
	if content == nil {
		return []string{"Error: no non-nil response from `searchUrls`"}
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
		return []string{"Error: " + err.Error()}
	}

	if len(searchResults) == 0 {
		return []string{"Error: No search results found"}
	}

	return []string{"https://youtu.be/" + searchResults[0].ID}
}
