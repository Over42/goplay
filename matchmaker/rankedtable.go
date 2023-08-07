package matchmaker

// Contains rating and groups with this rating.
// Add O(1), Delete O(k), Get O(k),
// where k is the number of groups with similar rating.
type RankedGroupsTable map[int][]*Group

func (t *RankedGroupsTable) Add(group *Group) {
	groups, ok := (*t)[group.AvgRating]
	if !ok {
		groups = make([]*Group, 1)
		groups[0] = group
	} else {
		groups = append(groups, group)
	}
	(*t)[group.AvgRating] = groups
}

func (t *RankedGroupsTable) Delete(group *Group) {
	groups, ok := (*t)[group.AvgRating]
	if ok {
		index := -1
		for i, g := range groups {
			if groupsEqual(g, group) {
				index = i
			}
		}

		if index >= 0 {
			(*t)[group.AvgRating] = append((*t)[group.AvgRating][:index], (*t)[group.AvgRating][index+1:]...)
		}
	}
}

func (t *RankedGroupsTable) Get(rating int) ([]*Group, bool) {
	groups, ok := (*t)[rating]
	if ok {
		return groups, true
	} else {
		return nil, false
	}
}
