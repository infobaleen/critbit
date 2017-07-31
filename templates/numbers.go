package critbit

import (
	"unsafe"

	"github.com/cheekybits/genny/generic"
)

type KeyType generic.Number
type ValueType generic.Type

// MapKeyTypeValueType implements an associative array of ValueType indexed by KeyType.
type MapKeyTypeValueType struct {
	length int
	root   nodeMapKeyTypeValueType
}

func NewMapKeyTypeValueType() *MapKeyTypeValueType {
	var r MapKeyTypeValueType
	return &r
}

type nodeMapKeyTypeValueType struct {
	key   KeyType        // Key prefix up to critical bit
	crit  uint           // Position of critical bit  (LSB=0; ^uint(0) indicates leaf)
	child unsafe.Pointer // Pointer to children or value ([2]nodeMapKeyTypeValueType or ValueType)
}

// Return walking direction
func (c nodeMapKeyTypeValueType) dir(key KeyType) int {
	return int((key >> c.crit) & 1)
}

// Return number of highest (first) bit that is different between child prefix and provided prefix.
// If there are no differences within the prefix, the returned value is c.crit.
func (c nodeMapKeyTypeValueType) findCrit(key KeyType) uint {
	// Isolate differences in prefix
	key = ((key ^ c.key) >> (c.crit + 1)) << (c.crit + 1)
	// Zero bits from lowest to highest until there are no differences left
	var crit uint = c.crit
	for key != 0 {
		crit++
		key = key &^ (1 << crit)
	}
	return crit
}

func (c *nodeMapKeyTypeValueType) children() *[2]nodeMapKeyTypeValueType {
	return (*[2]nodeMapKeyTypeValueType)(c.child)
}

func (c *nodeMapKeyTypeValueType) value() *ValueType {
	return (*ValueType)(c.child)
}

// If a leaf with the same key is found, ^uint(0) and leaf node are returned.
// Otherwise, the critical bit and the first child with differing prefix are returned.
func (c *nodeMapKeyTypeValueType) find(key KeyType) (uint, *nodeMapKeyTypeValueType) {
	//fmt.Println("  Find start", key)
	var crit = c.findCrit(key)
	// Keep going deeper until !(c.crit != ^uint(0) && c.crit == crit).
	for c.crit != ^uint(0) && c.crit == crit {
		//fmt.Printf("    Go deeper. Prefix: %08b, Key: %08b, Crit: %d, Dir: %d\n", c.key, key, crit, c.dir(key))
		c = &(c.children())[c.dir(key)]
		crit = c.findCrit(key)
	}
	//fmt.Println("    Find end", crit, key, c.key)
	return crit, c
}

func (t *MapKeyTypeValueType) transformKey(key KeyType) KeyType {
	var mask KeyType = 1
	if mask-2 < 0 {
		mask = mask << 7
		for mask > 0 {
			mask = mask << 8
		}
		return key ^ mask
	}
	return key
}

// SetP inserts or replaces the value associated with the specified key.
// The specified value pointer can be used to modify the value without using Set.
func (t *MapKeyTypeValueType) SetP(key KeyType, val *ValueType) {
	key = t.transformKey(key)
	//fmt.Println("Set", key, val)
	//defer t.root.dbg("  ")
	// Make leaf node if tree is empty
	if t.length == 0 {
		t.length++
		t.root.key = key
		t.root.crit = ^uint(0)
		t.root.child = unsafe.Pointer(val)
		return
	}
	// Find node with longest shared prefix and critical bit
	var crit, n = t.root.find(key)
	// Replace value if the node is a leaf with the same key
	if crit == ^uint(0) {
		n.child = unsafe.Pointer(val)
		return
	}
	// Make new child nodes for found node and new value
	var children = [2]nodeMapKeyTypeValueType{*n, *n}
	// Overwrite found node
	n.child = unsafe.Pointer(&children)
	n.crit = crit
	// Set one child to value
	var dir = n.dir(key)
	children[dir].key = key
	children[dir].crit = ^uint(0)
	children[dir].child = unsafe.Pointer(val)
}

// Set inserts or replaces the value associated with the specified key.
func (t *MapKeyTypeValueType) Set(key KeyType, val ValueType) {
	t.SetP(key, &val)
}

// Get returns the internal pointer to the value associated with the specified key and true if the key exists.
// Otherwise nil and false are returned. The pointer can be used to modify the value without using Set.
func (t *MapKeyTypeValueType) GetP(key KeyType) (*ValueType, bool) {
	key = t.transformKey(key)
	//fmt.Println("Get", key)
	if t.length == 0 {
		return nil, false
	}
	// Find leaf node
	var crit, l = t.root.find(key)
	if crit == ^uint(0) {
		return (*ValueType)(l.child), true
	}
	return nil, false
}

// Get returns the value associated with the specified key and true if the key exists.
// Otherwise 0 and false are returned. If a nil pointer was associated with the key,
// Get will panic (use GetP instead).
func (t *MapKeyTypeValueType) Get(key KeyType) (ValueType, bool) {
	var v ValueType
	var p, ok = t.GetP(key)
	if ok {
		v = *p
	}
	return v, ok
}

// Length returns the number of distinct keys in the multiset
func (t *MapKeyTypeValueType) Length() int {
	return t.length
}

// func (c *nodeMapKeyTypeValueType) dbg(p string) {
// 	if c.crit != ^uint(0) {
// 		fmt.Printf(p+"Node: %08b %d\n", c.key, c.crit)
// 		p += "  "
// 		var children = (*[2]nodeMapKeyTypeValueType)(c.child)
// 		children[0].dbg(p)
// 		children[1].dbg(p)
// 	} else {
// 		fmt.Printf(p+"Leaf: %08b %d\n", c.key, *(*ValueType)(c.child))
// 	}
// }

// IterKeyTypeValueType The iterator becomes invalid
// if a new value is inserted in the underlying map, until the Reset or Jump method is called.
type IterKeyTypeValueType struct {
	t     *MapKeyTypeValueType
	nodes []*nodeMapKeyTypeValueType
	Found bool       // Initially false (also after calling Reset). Otherwise the return value of the last call to Next, Prev or Jump.
	Key   KeyType    // Key found by last call to Next, Prev.
	Value *ValueType // Pointer to value associated with key found by most last call to Next, Prev or Jump
}

// Iterator returns a new IterKeyTypeValueType.
func (t *MapKeyTypeValueType) Iterator() *IterKeyTypeValueType {
	var i IterKeyTypeValueType
	i.t = t
	i.Reset()
	return &i
}

// Jump advances the iterator to the specified key, independent of current state. If the exact key is found
// it returns true, otherwise false. Independent of whether the key is found or not, the next call to Prev or Next
// will advance the iterator to the next lower or higher key if there is one.
func (i *IterKeyTypeValueType) Seek(key KeyType) {
	key = i.t.transformKey(key)
	i.Reset()
	if i.t.length == 0 {
		return
	}
	var last = &i.t.root
	//var crit = c.findCrit(key)
	for last.crit != ^uint(0) && last.findCrit(key) == last.crit {
		i.nodes = append(i.nodes, last)
		last = &last.children()[last.dir(key)]
	}
	if last.crit != ^uint(0) || key != last.key {
		i.nodes = i.nodes[0 : len(i.nodes)-1]
	}
}

// Reset restores the initial state.
func (i *IterKeyTypeValueType) Reset() {
	i.Found = false
	i.Key = i.t.transformKey(0)
	i.Value = nil
	if i.nodes == nil {
		i.nodes = make([]*nodeMapKeyTypeValueType, 0, 64)
	}
	if i.t.length > 0 {
		i.nodes = i.nodes[0:1]
		i.nodes[0] = &i.t.root
	}
}

// Next advances the iterator to the next higher key and populates the iterators public Fields.
// If the iterator is in the initial state, the first call to Next will set the iterator to the lowest key.
// The return value is True unless there is no next higher key to advance to.
func (i *IterKeyTypeValueType) Next() bool {
	i.step(1)
	return i.Found
}

func (i *IterKeyTypeValueType) step(dir int) {
	// Comments describe behavior with dir == 1 (iterate from left to right).
	i.Found = false
	if len(i.nodes) == 0 { // Invalid
		return
	}
	// Go left until next leaf is found
	var next *nodeMapKeyTypeValueType = i.nodes[len(i.nodes)-1]
	for next.crit != ^uint(0) {
		next = &(*[2]nodeMapKeyTypeValueType)(next.child)[1-dir]
		i.nodes = append(i.nodes, next)
	}
	// Found leaf. Store data.
	i.Key = i.t.transformKey(next.key)
	i.Value = (*ValueType)(next.child)
	i.Found = true
	// Go up until a left node is found. Then one to the right if there is one.
	var d = len(i.nodes)
	var last = i.nodes[d-1]
	for d > 1 {
		var previous = i.nodes[d-2]
		if &(*[2]nodeMapKeyTypeValueType)(previous.child)[1-dir] == last {
			// Found left node. Replace with right node.
			i.nodes[d-1] = &(*[2]nodeMapKeyTypeValueType)(previous.child)[dir]
			i.nodes = i.nodes[0:d]
			return
		}
		last = previous
		i.nodes[d-1] = nil // Help gc
		d--
	}
	i.nodes = i.nodes[0:0]
}
