package matchmaker

import (
	"goplay/config"

	"container/list"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

// Test match quality on low player count
func TestLowPlayerCount(t *testing.T) {
	numPlayers := 1000

	cfg := &config.Config{
		Matchmaker: config.MatchmakerConfig{
			TeamSize:                5,
			TeamCount:               2,
			MaxRatingSpreadToSearch: 100,
			MaxRatingSpreadInGroup:  -1,
			CheckReadiness:          false,
		},
	}

	mm := &matchmaker{
		repository:         nil,
		searchQueue:        list.New(),
		rankedTable:        make(RankedGroupsTable),
		params:             &cfg.Matchmaker,
		serverConfig:       &cfg.Server,
		matchReadyCallback: writeToFile,
	}

	groups := generateGroups(numPlayers, 5, 1000)

	for i := range groups {
		mm.returnGroupToSearch(&groups[i])
	}

	mm.Run()
}

func generateGroups(count int, maxPlayers, maxRating int) []Group {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	groups := make([]Group, count)
	uid := 0
	for i := 0; i < count; i++ {
		playerCount := r.Intn(maxPlayers) + 1
		players := make([]Player, playerCount)
		for j := range players {
			players[j] = Player{
				ID:     uid,
				Rating: r.Intn(maxRating) + 1,
			}

			uid++
		}

		groups[i] = Group{
			ID:      strconv.Itoa(i),
			Players: players,
			Size:    len(players),
		}
	}

	return groups
}

func writeToFile(teams []Team, filename string) string {
	f, err := os.OpenFile("mm_test.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	matchInfo := struct {
		ID    int
		Teams [][]Player
	}{
		ID:    rand.Intn(10000),
		Teams: make([][]Player, len(teams)),
	}

	for i, team := range teams {
		for _, group := range team.groups {
			matchInfo.Teams[i] = append(matchInfo.Teams[i], group.Players...)
		}
	}

	matchJson, _ := json.Marshal(matchInfo)

	_, err = f.Write(matchJson)
	if err != nil {
		log.Fatal(err)
	}

	return ""
}
