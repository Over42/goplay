package matchmaker

import (
	"testing"
)

func TestAddSameRating(t *testing.T) {
	rankedTable := make(RankedGroupsTable)

	group1 := Group{ID: "1", AvgRating: 10}
	group2 := Group{ID: "2", AvgRating: 10}
	group3 := Group{ID: "3", AvgRating: 20}

	rankedTable.Add(&group1)
	rankedTable.Add(&group2)
	rankedTable.Add(&group3)

	if len(rankedTable) != 2 {
		t.Errorf("got %d, want %d", len(rankedTable), 2)
	}
}

func TestDeleteSameRating(t *testing.T) {
	rankedTable := make(RankedGroupsTable)

	group1 := Group{ID: "1", AvgRating: 10}
	group2 := Group{ID: "2", AvgRating: 10}

	rankedTable.Add(&group1)
	rankedTable.Add(&group2)

	rankedTable.Delete(&group2)

	if len(rankedTable) != 1 {
		t.Errorf("got %d, want %d", len(rankedTable), 1)
	}

	elem := rankedTable[10]
	if elem[0].ID != "1" {
		t.Errorf("got %s, want %d", elem[0].ID, 1)
	}
}

func TestGet(t *testing.T) {
	rankedTable := make(RankedGroupsTable)

	group1 := Group{ID: "1", AvgRating: 10}
	group2 := Group{ID: "2", AvgRating: 10}

	rankedTable.Add(&group1)
	rankedTable.Add(&group2)

	g, ok := rankedTable.Get(10)
	if (!ok) || (len(g) != 2) {
		t.Errorf("got %d, want %d", len(g), 2)
	}
}
