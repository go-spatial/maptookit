package grating

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/encoding/geojson"
)

func TestLabelForRow(t *testing.T) {

	type tcase struct {
		Grating *Grating
		Rows    []int
		Labels  []string
	}

	fn := func(tc tcase) func(*testing.T) {
		return func(t *testing.T) {

			for i := range tc.Rows {
				t.Run(fmt.Sprintf("%v row %v", i, tc.Rows[i]),
					func(t *testing.T) {
						lbl := tc.Grating.LabelForRow(tc.Rows[i])
						if lbl != tc.Labels[i] {
							t.Errorf("label, expected %s got %s", tc.Labels[i], lbl)
						}
					},
				)
			}
		}
	}

	seq := func(start, end int) (r []int) {

		if start > end {
			start, end = end, start
		}

		r = make([]int, 0, end-start)

		for i := start; i <= end; i++ {
			r = append(r, i)
		}
		return r
	}

	tests := map[string]tcase{
		"Rows: 10 row: 1": {
			Grating: &Grating{Rows: 11},
			Rows:    seq(0, 10),
			Labels:  strings.Split("M K J H G F E D C B A", " "),
		},
		"Rows: 22 row: 1": {
			Grating: &Grating{Rows: 23},
			Rows:    seq(0, 22),
			Labels:  strings.Split("AB AA Z Y X W V U T R P N M K J H G F E D C B A", " "),
		},
		"Rows: 10 row: 1 FlipLabel": {
			Grating: &Grating{Rows: 10, FlipYLabel: true},
			Rows:    seq(0, 10),
			Labels:  []string{"A", "B", "C", "D", "E", "F", "G", "H", "J", "K", ""},
		},
		"Rows: 10 row: 1 FlipLabel:no ": {
			Grating: &Grating{Rows: 10},
			Rows:    seq(0, 9),
			Labels:  strings.Split("K J H G F E D C B A", " "),
		},
		"Rows: 10 row: 11": {
			Grating: &Grating{Rows: 10},
			Rows:    []int{10, 11},
			Labels:  []string{"", ""},
		},
		"issue #219 nome": func() tcase {
			grate, err := NewGrating(
				0, 0,
				1024*10, 768*10,
				10, 10,
				true,
			)
			if err != nil {
				panic(err)
			}
			return tcase{
				Grating: grate,
				Rows:    seq(0, 9),
				Labels:  strings.Split("K J H G F E D C B A", " "),
			}
		}(),
	}

	for name, tc := range tests {
		t.Run(name, fn(tc))
	}
}

func TestGeoJSONFrom(t *testing.T) {
	type tcase struct {
		bds       *geom.Extent
		rows      uint
		cols      uint
		rectangle bool
		flipped   bool

		features geojson.FeatureCollection
		err      error
	}
	fn := func(tc tcase) func(*testing.T) {
		return func(t *testing.T) {

			features, err := GeoJSONFrom(tc.bds, tc.rows, tc.cols, tc.flipped, tc.rectangle)
			if err != nil {
				t.Errorf("error, expected nil, got %v", err)
			}
			val, err := json.Marshal(features)
			if err != nil {
				t.Errorf("json marshal error, expected nil, got %v", err)
				return
			}
			t.Logf("JSON:%s\n", val)
		}
	}
	tests := map[string]tcase{
		"simple": {
			rectangle: true,
			rows:      3,
			cols:      3,
			bds:       &geom.Extent{-122.48015, 48.753224, -122.38391, 48.781091},
		},
	}
	for name, tc := range tests {
		t.Run(name, fn(tc))
	}
}
