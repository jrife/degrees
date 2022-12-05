package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/grailbio/base/tsv"
)

type Title struct {
	TConst         string `tsv:"tconst"`
	TitleType      string `tsv:"titleType"`
	PrimaryTitle   string `tsv:"primaryTitle"`
	OriginalTitle  string `tsv:"originalTitle"`
	IsAdult        string `tsv:"isAdult"`
	StartYear      string `tsv:"startYear"`
	EndYear        string `tsv:"endYear"`
	RuntimeMinutes string `tsv:"runtimeMinutes"`
	Genres         string `tsv:"genres"`
}

type Principal struct {
	TConst     string `tsv:"tconst"`
	Ordering   int    `tsv:"ordering"`
	NConst     string `tsv:"nconst"`
	Category   string `tsv:"category"`
	Job        string `tsv:"job"`
	Characters string `tsv:"characters"`
}

type Name struct {
	NConst            string `tsv:"nconst"`
	PrimaryName       string `tsv:"primaryName"`
	BirthYear         string `tsv:"birthYear"`
	DeathYear         string `tsv:"deathYear"`
	PrimaryProfession string `tsv:"primaryProfession"`
	KnownForTitles    string `tsv:"knownForTitles"`
}

type Episode struct {
	TConst        string `tsv:"tconst"`
	ParentTConst  string `tsv:"parentTconst"`
	SeasonNumber  string `tsv:"seasonNumber"`
	EpisodeNumber string `tsv:"episodeNumber"`
}

type node struct {
	name      string
	parent    string
	job       map[string]string
	neighbors map[string]*node
}

var (
	from string
	to   string
)

func init() {
	flag.StringVar(&from, "from", "", "")
	flag.StringVar(&to, "to", "", "")
}

func main() {
	flag.Parse()

	principals, _ := os.OpenFile("./title.principals.tsv", os.O_RDONLY, 0)
	basics, _ := os.OpenFile("./title.basics.tsv", os.O_RDONLY, 0)
	names, _ := os.OpenFile("./name.basics.tsv", os.O_RDONLY, 0)
	episodes, _ := os.OpenFile("./title.episode.tsv", os.O_RDONLY, 0)

	graph := map[string]*node{}
	r := tsv.NewReader(basics)
	r.HasHeaderRow = true
	r.UseHeaderNames = true

	var title Title
	var err error
	fmt.Printf("Populating title index...\n")
	for err = r.Read(&title); err != io.EOF; err = r.Read(&title) {
		if err != nil {
			continue
		}

		// Only include movies and tv
		if title.TitleType != "movie" { //} && !strings.Contains(title.TitleType, "tv") {
			continue
		}

		graph[title.TConst] = &node{
			name:      title.PrimaryTitle,
			job:       map[string]string{},
			neighbors: map[string]*node{},
		}
	}

	r = tsv.NewReader(principals)
	r.HasHeaderRow = true
	r.UseHeaderNames = true

	var principal Principal
	fmt.Printf("Populating principal index...\n")
	for err = r.Read(&principal); err != io.EOF; err = r.Read(&principal) {
		if err != nil {
			continue
		}

		if principal.Category != "actor" && principal.Category != "actress" && principal.Category != "self" {
			continue
		}

		titleNode, ok := graph[principal.TConst]

		// Skip if this refers to a title we skipped
		if !ok {
			continue
		}

		principalNode, ok := graph[principal.NConst]

		if !ok {
			principalNode = &node{
				name:      principal.NConst,
				neighbors: map[string]*node{},
			}
			graph[principal.NConst] = principalNode
		}

		titleNode.neighbors[principal.NConst] = principalNode
		titleNode.job[principal.NConst] = principal.Category
		principalNode.neighbors[principal.TConst] = titleNode
	}

	r = tsv.NewReader(names)
	r.HasHeaderRow = true
	r.UseHeaderNames = true

	var name Name
	fmt.Printf("Enriching principal index...\n")
	for err = r.Read(&name); err != io.EOF; err = r.Read(&name) {
		if err != nil {
			continue
		}

		principalNode, ok := graph[name.NConst]

		// Skip if this refers to a title we skipped
		if !ok {
			continue
		}

		principalNode.name = name.PrimaryName
	}

	r = tsv.NewReader(episodes)
	r.HasHeaderRow = true
	r.UseHeaderNames = true

	var episode Episode
	for err = r.Read(&episode); err != io.EOF; err = r.Read(&episode) {
		if err != nil {
			continue
		}

		titleNode, ok := graph[episode.TConst]

		// Skip if this refers to a title we skipped
		if !ok {
			continue
		}

		titleNode.parent = episode.ParentTConst
	}

	fmt.Printf("Performing graph search...\n")
	path := find(graph, from, to, map[string]struct{}{}, 7)

	if path == nil {
		fmt.Printf("No connection found within 7 jumps\n")
		return
	}

	fmt.Printf("%+v\n", path)
	fmt.Printf("============================\n")
	fmt.Printf("%s --> %s\n", graph[path[0]].name, graph[path[len(path)-1]].name)
	fmt.Printf("============================\n")

	for i := 1; i < len(path); i += 2 {
		titleNode := graph[path[i]]

		fmt.Printf("%s: %s (%s) --> %s (%s)\n", titleDescription(graph, path[i]), graph[path[i-1]].name, titleNode.job[path[i-1]], graph[path[i+1]].name, titleNode.job[path[i+1]])
	}
}

func titleDescription(graph map[string]*node, tConst string) string {
	node := graph[tConst]

	if node.parent != "" {
		parentNode := graph[node.parent]

		return fmt.Sprintf("%s (%s)", parentNode.name, node.name)
	}

	return node.name
}

func find(graph map[string]*node, from, to string, visited map[string]struct{}, limit int) []string {
	if from == to {
		return []string{from}
	} else if limit == 0 {
		return nil
	}

	for n := range graph[from].neighbors {
		if _, ok := visited[n]; ok {
			continue
		}

		visited[from] = struct{}{}
		path := find(graph, n, to, visited, limit-1)

		if path != nil {
			return append(path, from)
		}

		delete(visited, from)
	}

	return nil
}
