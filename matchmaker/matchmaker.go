package matchmaker

import (
	"goplay/config"
	"goplay/repository"

	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

type matchmaker struct {
	repository          repository.Repository
	searchQueue         *list.List
	rankedTable         RankedGroupsTable
	preparingMatchTeams []Team
	waitingMatchPlayers map[int]*Player
	penalizedPlayers    map[int]time.Time
	params              *config.MatchmakerConfig
	serverConfig        *config.ServerConfig
	matchReadyCallback  func(teams []Team, sendTo string) string
}

type Matchmaker interface {
	AddGroup(ctx context.Context, id string, playerIDs []int, matchFound chan string, searchCancelled chan bool) error
	RemoveGroup(id string)
	SetPlayerReady(id int)
	Run()
}

func NewMatchmaker(repository repository.Repository, cfg *config.Config, onMatchReady func(teams []Team, sendTo string) string) Matchmaker {
	return &matchmaker{
		repository:         repository,
		searchQueue:        list.New(),
		rankedTable:        make(RankedGroupsTable),
		params:             &cfg.Matchmaker,
		serverConfig:       &cfg.Server,
		matchReadyCallback: onMatchReady,
	}
}

func (m *matchmaker) Run() {
	for {
		m.makeMatch()
	}
}

func (m *matchmaker) AddGroup(ctx context.Context, id string, playerIDs []int, matchFound chan string, searchCancelled chan bool) error {
	context, cancel := context.WithTimeout(ctx, m.serverConfig.DBRequestTimeout)
	defer cancel()

	playersInfo, err := m.repository.GetUsersById(context, playerIDs)
	if err != nil {
		return err
	}

	players := make([]Player, len(playersInfo))
	for i := range players {
		players[i] = Player{
			ID:     int(playersInfo[i].ID),
			Rating: playersInfo[i].Rating,
		}
	}

	group := &Group{
		ID:           id,
		Players:      players,
		Size:         len(players),
		matchFound:   matchFound,
		cancelSearch: searchCancelled,
	}

	err = m.checkRatingSpread(group)
	if err != nil {
		return err
	}

	err = m.checkPenalty(group)
	if err != nil {
		return err
	}

	group.calcRating()

	m.searchQueue.PushBack(group)
	m.rankedTable.Add(group)

	return nil
}

func (m *matchmaker) RemoveGroup(id string) {
	g := &Group{
		ID: id,
	}
	var group *Group = nil
	for e := m.searchQueue.Front(); e != nil; e = e.Next() {
		if groupsEqual(g, e.Value.(*Group)) {
			group = e.Value.(*Group)
		}
	}
	if group == nil {
		return
	}

	group.cancelSearch <- true

	for i := range m.preparingMatchTeams {
		m.preparingMatchTeams[i].remove(group)
	}

	m.removeGroupFromSearch(group)
}

func (m *matchmaker) SetPlayerReady(id int) {
	m.waitingMatchPlayers[id].ready = true
}

func (m *matchmaker) removeGroupFromSearch(group *Group) {
	for e := m.searchQueue.Front(); e != nil; e = e.Next() {
		g := e.Value.(*Group)
		if groupsEqual(g, group) {
			m.searchQueue.Remove(e)
			break
		}
	}

	m.rankedTable.Delete(group)
}

func (m *matchmaker) removeTeamsFromSearch(teams []Team) {
	for i, team := range teams {
		for j := range team.groups {
			m.removeGroupFromSearch(teams[i].groups[j])
		}
	}
}

func (m *matchmaker) returnGroupToSearch(group *Group) {
	m.searchQueue.PushBack(group)
	m.rankedTable.Add(group)
}

func (m *matchmaker) makeMatch() {
	if m.searchQueue.Len() == 0 {
		return
	}

	m.preparingMatchTeams = make([]Team, m.params.TeamCount)
	firstInQueue := m.searchQueue.Front()
	group := firstInQueue.Value.(*Group)
	m.preparingMatchTeams[0].add(group)
	group.SelectedForMatch = true
	avgRating := group.AvgRating

	allTeamsFull := false
	for !allTeamsFull {
		for i := range m.preparingMatchTeams {
			err := m.preparingMatchTeams[i].fill(m, avgRating)
			if err != nil {
				m.resetGroupsInRankedTable(m.preparingMatchTeams)
				m.searchQueue.MoveToBack(firstInQueue)
				return
			}
		}
		// Groups can cancel (exit) search at any time
		allTeamsFull = m.checkAllTeamsFull(m.preparingMatchTeams)
	}

	m.removeTeamsFromSearch(m.preparingMatchTeams)
	teams := make([]Team, len(m.preparingMatchTeams))
	copy(teams, m.preparingMatchTeams)
	go m.createMatch(teams)
}

func (m *matchmaker) findGroupWithSameRating(avgRating int, size int) (*Group, error) {
	for i := 0; i < m.params.MaxRatingSpreadToSearch; i++ {
		groups, found := m.rankedTable.Get(avgRating + i)
		if !found {
			groups, found = m.rankedTable.Get(avgRating - i)
		}

		if found {
			for j := range groups {
				if (groups[j].Size <= size) && (!groups[j].SelectedForMatch) {
					groups[j].SelectedForMatch = true
					return groups[j], nil
				}
			}
		}
	}

	return nil, errors.New("can't find group with similar rating")
}

func (m *matchmaker) resetGroupsInRankedTable(teams []Team) {
	for i, team := range teams {
		for j := range team.groups {
			teams[i].groups[j].SelectedForMatch = false
		}
	}
}

func (m *matchmaker) checkAllTeamsFull(matchTeams []Team) (ok bool) {
	for i := range matchTeams {
		if matchTeams[i].numPlayers < m.params.TeamSize {
			return false
		}
	}

	return true
}

func (m *matchmaker) createMatch(teams []Team) {
	if !m.params.CheckReadiness {
		serverID := m.matchReadyCallback(teams, m.serverConfig.ServerManagerAddr)
		m.notifyMatchFound(teams, serverID)
	} else {
		m.addWaitingPlayers(teams)
		allPlayersReady, notReadyPlayers := m.checkAllPlayersReady(teams)
		if allPlayersReady {
			serverID := m.matchReadyCallback(teams, m.serverConfig.ServerManagerAddr)
			m.notifyMatchFound(teams, serverID)
		} else {
			// Groups where any of players didn't accept the match are removed from search
			m.returnGroupsToSearch(teams, notReadyPlayers)

			if m.params.PenaltyForUnacceptedMatch {
				m.addPenalty(notReadyPlayers)
			}
		}
		m.removeWaitingPlayers(teams)
	}
}

func (m *matchmaker) addWaitingPlayers(teams []Team) {
	for i, team := range teams {
		for j, group := range team.groups {
			for k, player := range group.Players {
				m.waitingMatchPlayers[player.ID] = &teams[i].groups[j].Players[k]
			}
		}
	}
}

func (m *matchmaker) removeWaitingPlayers(teams []Team) {
	for _, team := range teams {
		for _, group := range team.groups {
			for _, player := range group.Players {
				delete(m.waitingMatchPlayers, player.ID)
			}
		}
	}
}

// Some players may lost connection in process of search,
// so in most cases we have to check that they are ready to play,
// which can be done explicitly (players press 'Accept' button) or
// implicitly (automatically send 'player ready' request after 'match ready' response).
func (m *matchmaker) checkAllPlayersReady(teams []Team) (success bool, notReadyPlayers []*Player) {
	var allPlayersReady bool
	timer := time.Duration(m.params.SecondsToAcceptMatch) * time.Second
	for start := time.Now(); time.Since(start) < timer; {
		allPlayersReady = true
		for _, team := range teams {
			for _, group := range team.groups {
				for _, player := range group.Players {
					if !player.ready {
						allPlayersReady = false
					}
				}

			}
		}
		if allPlayersReady {
			return true, nil
		}
	}

	notReady := make([]*Player, 0)
	for i, team := range teams {
		for j, group := range team.groups {
			for k, player := range group.Players {
				if !player.ready {
					notReady = append(notReady, &teams[i].groups[j].Players[k])
				}
			}
		}
	}

	return allPlayersReady, notReady
}

func (m *matchmaker) returnGroupsToSearch(teams []Team, notReadyPlayers []*Player) {
	for i, team := range teams {
		for j, group := range team.groups {
			for _, player := range group.Players {
				for _, notReady := range notReadyPlayers {
					if player.ID == notReady.ID {
						m.removeGroupFromSearch(teams[i].groups[j])
					} else {
						m.returnGroupToSearch(teams[i].groups[j])
					}
				}
			}
		}
	}
}

func (m *matchmaker) checkRatingSpread(group *Group) error {
	if m.params.MaxRatingSpreadInGroup < 0 {
		return nil
	}

	min, max := 0, 0
	for _, player := range group.Players {
		if player.Rating < min {
			min = player.Rating
		}
		if player.Rating > max {
			max = player.Rating
		}
	}

	if max-min > m.params.MaxRatingSpreadInGroup {
		return errors.New("spread of rating in the group is too high")
	}

	return nil
}

func (m *matchmaker) addPenalty(players []*Player) {
	t := time.Now()
	for _, player := range players {
		m.penalizedPlayers[player.ID] = t
	}
}

func (m *matchmaker) checkPenalty(group *Group) error {
	for _, player := range group.Players {
		penaltyTime, hasPenalty := m.penalizedPlayers[player.ID]
		if hasPenalty {
			if time.Since(penaltyTime) < time.Duration(m.params.PenaltySeconds)*time.Second {
				return errors.New("some players have penalty for unaccepted match")
			} else {
				delete(m.penalizedPlayers, player.ID)
			}
		}
	}

	return nil
}

func RequestServer(teams []Team, sendTo string) string {
	teamsAndPlayerIds := make([][]int, len(teams))
	for i, team := range teams {
		for _, group := range team.groups {
			for k, player := range group.Players {
				teamsAndPlayerIds[i][k] = player.ID
			}
		}
	}

	reqBody, err := json.Marshal(teamsAndPlayerIds)
	if err != nil {
		log.Fatalf("failed to marshall teams: %s", err)
	}

	req, err := http.NewRequest("POST", sendTo, bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("failed to build request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to send request: %s", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("failed to read body of response: %s", err)
	}

	serverId := string(resBody)
	return serverId
}

func (m *matchmaker) notifyMatchFound(teams []Team, serverID string) {
	for i := range teams {
		for j := range teams[i].groups {
			teams[i].groups[j].matchFound <- serverID
		}
	}
}
