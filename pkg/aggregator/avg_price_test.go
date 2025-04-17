package aggregator

import (
	"testing"

	apiAttr "propertytreeanalyzer/pkg/api/attribute"
	num "propertytreeanalyzer/pkg/numeric"

	"github.com/cockroachdb/apd/v3"
)

func TestAveragePriceFloat(t *testing.T) {
	agg := &treeSizeAggregator{}
	in := make(chan apiAttr.NumericAttribute, 3)
	in <- num.NewFloatAttribute(1.2)
	in <- num.NewFloatAttribute(2.8)
	in <- num.NewFloatAttribute(3.0)
	close(in)

	avg, err := agg.averagePrice(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, err := num.CastToFloatAttribute(avg)
	if err != nil {
		t.Fatalf("expected float attribute, got error: %v", err)
	}
	const eps = 1e-9
	got := f.GetFloat()
	want := (1.2 + 2.8 + 3.0) / 3.0
	if diff := got - want; diff > eps || diff < -eps {
		t.Errorf("averagePrice float = %v; want %v", got, want)
	}
}

func TestAveragePriceDecimal(t *testing.T) {
	agg := &treeSizeAggregator{}
	in := make(chan apiAttr.NumericAttribute, 2)
	d1, _, err := apd.NewFromString("10.00")
	if err != nil {
		t.Fatal(err)
	}
	d2, _, err := apd.NewFromString("20.00")
	if err != nil {
		t.Fatal(err)
	}
	in <- num.NewDecimalAttribute(d1)
	in <- num.NewDecimalAttribute(d2)
	close(in)

	avg, err := agg.averagePrice(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dec, err := num.CastToDecimalAttribute(avg)
	if err != nil {
		t.Fatalf("expected decimal attribute, got error: %v", err)
	}
	if s := dec.GetDecimal().String(); s != "15.00" {
		t.Errorf("averagePrice decimal = %v; want 15.00", s)
	}
}

func TestAveragePriceEmpty(t *testing.T) {
	agg := &treeSizeAggregator{}
	in := make(chan apiAttr.NumericAttribute)
	close(in)

	avg, err := agg.averagePrice(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if avg.GetNumericType() != apiAttr.Nothing {
		t.Errorf("empty averagePrice type = %v; want Nothing", avg.GetNumericType())
	}
}

func TestAveragePriceMixedTypes(t *testing.T) {
	agg := &treeSizeAggregator{}
	in := make(chan apiAttr.NumericAttribute, 2)
	in <- num.NewFloatAttribute(1.23)
	d, _, _ := apd.NewFromString("4.56")
	in <- num.NewDecimalAttribute(d)
	close(in)

	_, err := agg.averagePrice(in)
	if err == nil {
		t.Error("expected error mixing float and decimal, got nil")
	}
}

func TestAveragePriceConcurrency(t *testing.T) {
	agg := &treeSizeAggregator{}
	const N = 1000
	ch1 := make(chan apiAttr.NumericAttribute, N)
	ch2 := make(chan apiAttr.NumericAttribute, N)
	for i := range N {
		ch1 <- num.NewFloatAttribute(float64(i))
		ch2 <- num.NewFloatAttribute(float64(i) * 2)
	}
	close(ch1)
	close(ch2)

	done := make(chan struct{})
	go func() {
		if _, err := agg.averagePrice(ch1); err != nil {
			t.Errorf("concurrent avg1 error: %v", err)
		}
		done <- struct{}{}
	}()
	go func() {
		if _, err := agg.averagePrice(ch2); err != nil {
			t.Errorf("concurrent avg2 error: %v", err)
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}

func BenchmarkAveragePriceFloat(b *testing.B) {
	agg := &treeSizeAggregator{}
	for i := 0; i < b.N; i++ {
		in := make(chan apiAttr.NumericAttribute, 1000)
		for j := range 1000 {
			in <- num.NewFloatAttribute(float64(j))
		}
		close(in)
		_, _ = agg.averagePrice(in)
	}
}

func BenchmarkAveragePriceDecimal(b *testing.B) {
	agg := &treeSizeAggregator{}
	d, _, _ := apd.NewFromString("1.23")
	for b.Loop() {
		in := make(chan apiAttr.NumericAttribute, 1000)
		for range 1000 {
			in <- num.NewDecimalAttribute(d)
		}
		close(in)
		_, _ = agg.averagePrice(in)
	}
}
