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

func TestBasic(t *testing.T) {
	const L = 10000
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

	// Check contents
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
			t.Fatal("Wrong unsigned key or value", r, i.Key, i.Value)
		}
	}
	if l := len(uRef); l > 0 {
		t.Fatal(l, "unsigned elements left")
	}
	for i := iMap.Iterator(); i.Next(); iRef = iRef[1:] {
		if r := iRef[0]; r != i.Key || r != *i.Value {
			t.Fatal("Wrong signed key or value", r, i.Key, i.Value)
		}
	}
	if l := len(iRef); l > 0 {
		t.Fatal(l, "Signed elements left")
	}
}

func TestSeek(t *testing.T) {
	var m MapUintUint
	var uRef = UintSlice{8, 4, 0, 5, 7, 1}
	for _, v := range uRef {
		m.Set(v, v)
	}
	var it = m.Iterator()
	for _, r := range uRef {
		it.Seek(r)
		it.Next()
		if it.Found != true || it.Key != r || *it.Value != r {
			t.Fatal("Wrong key or value from Next after Seek", r, it.Found, it.Key, it.Value)
		}
		it.Seek(r)
		it.Prev()
		if it.Found != true || it.Key != r || *it.Value != r {
			t.Fatal("Wrong key or value from Prev after Seek", r, it.Found, it.Key, it.Value)
		}
	}
	it.Seek(2)
	it.Next()
	if it.Found != true || it.Key != 4 || *it.Value != 4 {
		t.Fatal("Wrong key or value from Next after Seek with non-existent key", 4, it.Found, it.Key, it.Value)
	}
	it.Seek(2)
	it.Prev()
	if it.Found != true || it.Key != 1 || *it.Value != 1 {
		t.Fatal("Wrong key or value from Prev after Seek with non-existent key", 4, it.Found, it.Key, it.Value)
	}
}
