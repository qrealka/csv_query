package aggregator

import (
	"context"
	"testing"

	apiAttr "propertytreeanalyzer/pkg/api/attribute"
	apiGroupify "propertytreeanalyzer/pkg/api/groupify"

	"github.com/xyproto/randomstring"
)

// baseAttr implements apiAttr.BaseAttribute for testing.
type baseAttr string

func (b baseAttr) String() string { return string(b) }

// mockGroupItem implements apiGroupify.StreetGroupItem.
type mockGroupItem struct {
	key    string
	street string
}

func (m mockGroupItem) Key() apiAttr.BaseAttribute         { return baseAttr(m.key) }
func (m mockGroupItem) StreetName() apiGroupify.StreetName { return apiGroupify.StreetName(m.street) }

// mockStreetAttr implements apiAttr.StreetAttribute.
type mockStreetAttr struct {
	street string
	val    string
}

func (m mockStreetAttr) StreetName() string     { return m.street }
func (m mockStreetAttr) AttributeValue() string { return m.val }
func (m mockStreetAttr) EqualTo(other apiAttr.StreetAttribute) bool {
	return m.street == other.StreetName() && m.val == other.AttributeValue()
}

func TestProcess_Int(t *testing.T) {
	// one group "g1" with street "s1", prices 10.00 and 20.00 → avg 15.00
	groups := make(chan apiGroupify.StreetGroupItem, 1)
	groups <- mockGroupItem{"g1", "s1"}
	close(groups)

	streets := make(chan apiAttr.StreetAttribute, 3)
	streets <- mockStreetAttr{"s1", "1"}
	streets <- mockStreetAttr{"s1", "2"}
	streets <- mockStreetAttr{"s1", "4"}
	close(streets)

	agg := NewAvgPriceBy(groups)
	out, err := agg.Process(t.Context(), streets)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 group, got %d", len(out))
	}
	// expect exactly 15
	if out[0].AverageValue() != "2.33" {
		t.Errorf("decimal avg = %s, want 2.33", out[0].AverageValue())
	}
}

func TestProcess_Float(t *testing.T) {
	// one group "gA" with street "foo", prices 1.5 and 2.5 → avg 2.0
	groups := make(chan apiGroupify.StreetGroupItem, 1)
	groups <- mockGroupItem{"gA", "foo"}
	close(groups)

	streets := make(chan apiAttr.StreetAttribute, 2)
	streets <- mockStreetAttr{"foo", "1.11"}
	streets <- mockStreetAttr{"foo", "2.0007"}
	close(streets)

	agg := NewAvgPriceBy(groups)
	out, err := agg.Process(t.Context(), streets)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 group, got %d", len(out))
	}
	if out[0].AverageValue() != "1.56" {
		t.Errorf("float avg = %s, want 1.56", out[0].AverageValue())
	}
}

func TestProcess_MultipleGroups(t *testing.T) {
	// two groups g1 and g2
	groups := make(chan apiGroupify.StreetGroupItem, 2)
	groups <- mockGroupItem{"g1", "s1"}
	groups <- mockGroupItem{"g2", "s2"}
	close(groups)

	streets := make(chan apiAttr.StreetAttribute, 4)
	streets <- mockStreetAttr{"s1", "10"}
	streets <- mockStreetAttr{"s1", "20"}
	streets <- mockStreetAttr{"s2", "3"}
	streets <- mockStreetAttr{"s2", "7"}
	close(streets)

	agg := NewAvgPriceBy(groups)
	out, err := agg.Process(t.Context(), streets)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(out))
	}

	// check g1
	if out[0].GroupKey() != "g1" {
		t.Errorf("first key = %s, want g1", out[0].GroupKey())
	}
	if out[0].AverageValue() != "15.00" {
		t.Errorf("g1 avg = %s, want 15.00", out[0].AverageValue())
	}
	// check g2
	if out[1].GroupKey() != "g2" {
		t.Errorf("second key = %s, want g2", out[1].GroupKey())
	}
	if out[1].AverageValue() != "5.00" {
		t.Errorf("g2 avg = %s, want 5.00", out[1].AverageValue())
	}
}

func TestProcess_NoData(t *testing.T) {
	// no groups → should return empty slice, no error
	groups := make(chan apiGroupify.StreetGroupItem)
	close(groups)
	streets := make(chan apiAttr.StreetAttribute)
	close(streets)

	agg := NewAvgPriceBy(groups)
	out, err := agg.Process(context.Background(), streets)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty output, got %v", out)
	}
}

func BenchmarkProcess(b *testing.B) {
	const Ngroups = 5
	const Nper = 10000
	// prepare static group definitions
	baseGroups := make([]mockGroupItem, Ngroups)
	for i := range Ngroups {
		name := randomstring.HumanFriendlyEnglishString(5)
		street := randomstring.HumanFriendlyEnglishString(15)
		baseGroups[i] = mockGroupItem{key: name, street: street}
	}

	b.ResetTimer()

	for b.Loop() {
		groups := make(chan apiGroupify.StreetGroupItem, Ngroups)
		streets := make(chan apiAttr.StreetAttribute, Ngroups*Nper)
		// feed groups
		for _, g := range baseGroups {
			groups <- g
		}
		close(groups)
		// feed streets
		for _, g := range baseGroups {
			for range Nper {
				streets <- mockStreetAttr{
					street: g.street,
					val:    "100",
				}
			}
		}
		close(streets)
		agg := NewAvgPriceBy(groups)
		if _, err := agg.Process(context.Background(), streets); err != nil {
			b.Fatal(err)
		}
	}
}
