package aggregator

import (
	"context"
	"testing"

	apiAttr "propertytreeanalyzer/pkg/api/attribute"
	apiGroupify "propertytreeanalyzer/pkg/api/groupify"
	num "propertytreeanalyzer/pkg/numeric"

	"github.com/cockroachdb/apd/v3"
	"github.com/xyproto/randomstring"
)

// baseAttr implements apiAttr.BaseAttribute for testing.
type baseAttr string

func (b baseAttr) String() string { return string(b) }

// mockGroupItem implements apiGroupify.StreetGroupItem.
type mockGroupItem struct {
	key    string
	street apiGroupify.StreetName
}

func (m mockGroupItem) Key() apiAttr.BaseAttribute         { return baseAttr(m.key) }
func (m mockGroupItem) StreetName() apiGroupify.StreetName { return m.street }

// mockStreetAttr implements apiAttr.StreetAttribute.
type mockStreetAttr struct {
	street string
	val    apiAttr.NumericAttribute
}

func (m mockStreetAttr) StreetName() string                       { return m.street }
func (m mockStreetAttr) AttributeValue() apiAttr.NumericAttribute { return m.val }
func (m mockStreetAttr) EqualTo(other apiAttr.StreetAttribute) bool {
	return m.street == other.StreetName() && m.val.EqualTo(other.AttributeValue())
}

// func sortResults(rs []api.AverageByGroup) {
// 	sort.Slice(rs, func(i, j int) bool {
// 		return rs[i].GroupKey().String() < rs[j].GroupKey().String()
// 	})
// }

func TestProcess_Decimal(t *testing.T) {
	// one group "g1" with street "s1", prices 10.00 and 20.00 → avg 15.00
	groups := make(chan apiGroupify.StreetGroupItem, 1)
	groups <- mockGroupItem{"g1", apiGroupify.ParseStreetName("s1")}
	close(groups)

	streets := make(chan apiAttr.StreetAttribute, 2)
	streets <- mockStreetAttr{"s1", num.NewDecimalAttribute(apd.New(10, 0))}
	streets <- mockStreetAttr{"s1", num.NewDecimalAttribute(apd.New(20, 0))}
	close(streets)

	agg := NewAvgPriceBy(groups)
	out, err := agg.Process(t.Context(), streets)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 group, got %d", len(out))
	}
	decOut, err := num.CastToDecimalAttribute(out[0].AverageValue())
	if err != nil {
		t.Fatal(err)
	}
	// expect exactly 15
	if decOut.GetDecimal().Cmp(apd.New(15, 0)) != 0 {
		t.Errorf("decimal avg = %s, want 15", decOut.GetDecimal().String())
	}
}

func TestProcess_Float(t *testing.T) {
	// one group "gA" with street "foo", prices 1.5 and 2.5 → avg 2.0
	groups := make(chan apiGroupify.StreetGroupItem, 1)
	groups <- mockGroupItem{"gA", apiGroupify.ParseStreetName("foo")}
	close(groups)

	streets := make(chan apiAttr.StreetAttribute, 2)
	streets <- mockStreetAttr{"foo", num.NewFloatAttribute(1.5)}
	streets <- mockStreetAttr{"foo", num.NewFloatAttribute(2.5)}
	close(streets)

	agg := NewAvgPriceBy(groups)
	out, err := agg.Process(t.Context(), streets)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 group, got %d", len(out))
	}
	floatOut, err := num.CastToFloatAttribute(out[0].AverageValue())
	if err != nil {
		t.Fatal(err)
	}
	if got := floatOut.GetFloat(); got != 2.0 {
		t.Errorf("float avg = %v, want 2.0", got)
	}
}

func TestProcess_MultipleGroups(t *testing.T) {
	// two groups g1 and g2
	groups := make(chan apiGroupify.StreetGroupItem, 2)
	groups <- mockGroupItem{"g1", apiGroupify.ParseStreetName("s1")}
	groups <- mockGroupItem{"g2", apiGroupify.ParseStreetName("s2")}
	close(groups)

	streets := make(chan apiAttr.StreetAttribute, 4)
	streets <- mockStreetAttr{"s1", num.NewFloatAttribute(10)}
	streets <- mockStreetAttr{"s1", num.NewFloatAttribute(20)}
	streets <- mockStreetAttr{"s2", num.NewDecimalAttribute(apd.New(3, 0))}
	streets <- mockStreetAttr{"s2", num.NewDecimalAttribute(apd.New(7, 0))}
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
	if out[0].GroupKey().String() != "g1" {
		t.Errorf("first key = %s, want g1", out[0].GroupKey().String())
	}
	f1, err := num.CastToFloatAttribute(out[0].AverageValue())
	if err != nil {
		t.Fatal(err)
	}
	if f1.GetFloat() != 15.0 {
		t.Errorf("g1 avg = %v, want 15.0", f1.GetFloat())
	}
	// check g2
	if out[1].GroupKey().String() != "g2" {
		t.Errorf("second key = %s, want g2", out[1].GroupKey().String())
	}
	d2, err := num.CastToDecimalAttribute(out[1].AverageValue())
	if err != nil {
		t.Fatal(err)
	}
	if d2.GetDecimal().Cmp(apd.New(5, 0)) != 0 {
		t.Errorf("g2 avg = %s, want 5.00", d2.GetDecimal().String())
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
		baseGroups[i] = mockGroupItem{key: name, street: apiGroupify.ParseStreetName(street)}
	}

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
			for j := range Nper {
				streets <- mockStreetAttr{
					street: g.street.String(),
					val:    num.NewFloatAttribute(float64(j%100) + 0.5),
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
