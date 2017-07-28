package critbit

import (
	"math/rand"
	"sort"
	"testing"
)

type UintSlice []uint

func (s UintSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s UintSlice) Len() int           { return len(s) }
func (s UintSlice) Less(i, j int) bool { return s[i] < s[j] }

func TestCritbit(t *testing.T) {
	const L = 100
	var iRef = make([]int, L)
	var uRef = make([]uint, L)
	var iMap MapIntInt
	var uMap MapUintUint

	// Add random entries to tree and reference array
	for i := 0; i < L; i++ {
		var el = int(rand.Uint64())
		if _, ok := iMap.Get(el); ok {
			i--
			continue
		}
		uRef[i] = uint(el)
		iRef[i] = el
		uMap.Set(uint(el), uint(el))
		iMap.Set(el, el)
	}

	// Check Get
	for _, r := range uRef {
		if el, ok := uMap.Get(r); !ok || r != el {
			t.Fatal("Unsigned entry not found or wrong value", r, el, ok)
		}
	}
	for _, r := range iRef {
		if el, ok := iMap.Get(r); !ok || r != el {
			t.Fatal("Signed entry not found or wrong value", r, el, ok)
		}
	}

	// Check iterator
	sort.Ints(iRef)
	sort.Sort(UintSlice(uRef))
	for i := uMap.Iterator(); i.Next(); uRef = uRef[1:] {
		if r := uRef[0]; r != i.Key || r != *i.Value {
			t.Fatal("Wrong unsigned key or value", r, i.Key, *i.Value)
		}
	}
	if len(uRef) > 0 {
		t.Fatal("Unsigned elements left")
	}
	for i := iMap.Iterator(); i.Next(); iRef = iRef[1:] {
		if r := iRef[0]; r != i.Key || r != *i.Value {
			t.Fatal("Wrong signed key or value", r, i.Key, *i.Value)
		}
	}
	if len(iRef) > 0 {
		t.Fatal("Signed elements left")
	}
}
