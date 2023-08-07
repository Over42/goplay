package matchmaker

type Player struct {
	ID           int `json:"id"`
	Rating       int `json:"rating"`
	ping         int
	wonLastMatch bool
	ready        bool
}

// Base struct for matchmaker, may consist of one player
type Group struct {
	ID               string
	Players          []Player
	Size             int
	AvgRating        int
	SumRating        int
	SelectedForMatch bool
	matchFound       chan string
	cancelSearch     chan bool
}

type Team struct {
	groups     []*Group
	numPlayers int
}

func (g *Group) calcRating() {
	for i := range g.Players {
		g.SumRating += g.Players[i].Rating
	}

	g.AvgRating = g.SumRating / len(g.Players)
}

func groupsEqual(a, b *Group) bool {
	return a.ID == b.ID
}

func (t *Team) add(group *Group) {
	t.groups = append(t.groups, group)
	t.numPlayers += group.Size
}

func (t *Team) remove(group *Group) {
	index := -1
	for i := range t.groups {
		if groupsEqual(t.groups[i], group) {
			index = i
		}
	}

	if index >= 0 {
		t.groups = append(t.groups[:index], t.groups[index+1:]...)
		t.numPlayers -= group.Size
	}
}

func (t *Team) fill(m *matchmaker, avgRating int) error {
	for t.numPlayers < m.params.TeamSize {
		playersToAdd := m.params.TeamSize - t.numPlayers
		g, err := m.findGroupWithSameRating(avgRating, playersToAdd)
		if err != nil {
			return err
		}

		t.add(g)
	}
	return nil
}
