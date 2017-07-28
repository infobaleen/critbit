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

// If a leaf with the same key is found, ^uint(0) and leaf node are returned.
// Otherwise, the critical bit and the first child with differing prefix are returned.
func (c *nodeMapKeyTypeValueType) find(key KeyType) (uint, *nodeMapKeyTypeValueType) {
	//fmt.Println("  Find start", key)
	var crit = c.findCrit(key)
	// Keep going deeper until !(c.crit != ^uint(0) && c.crit == crit).
	for c.crit != ^uint(0) && c.crit == crit {
		//fmt.Printf("    Go deeper. Prefix: %08b, Key: %08b, Crit: %d, Dir: %d\n", c.key, key, crit, c.dir(key))
		c = &(*[2]nodeMapKeyTypeValueType)(c.child)[c.dir(key)]
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

// Add increases the frequency of the provided key by inc
func (t *MapKeyTypeValueType) Set(key KeyType, val ValueType) {
	key = t.transformKey(key)
	//fmt.Println("Set", key, val)
	//defer t.root.dbg("  ")
	// Make leaf node if tree is empty
	if t.length == 0 {
		t.length++
		t.root.key = key
		t.root.crit = ^uint(0)
		t.root.child = unsafe.Pointer(&val)
		return
	}
	// Find node with longest shared prefix and critical bit
	var crit, n = t.root.find(key)
	// Replace value if the node is a leaf with the same key
	if crit == ^uint(0) {
		n.child = unsafe.Pointer(&val)
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
	children[dir].child = unsafe.Pointer(&val)
}

// Get returns the value associated with the provided key and true if the key exists.
// Otherwise 0 and false are returned
func (t *MapKeyTypeValueType) Get(key KeyType) (ValueType, bool) {
	key = t.transformKey(key)
	//fmt.Println("Get", key)
	var zero ValueType
	if t.length == 0 {
		return zero, false
	}
	// Find leaf node
	var crit, l = t.root.find(key)
	if crit == ^uint(0) {
		return *(*ValueType)(l.child), true
	}
	return zero, false
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

type IterKeyTypeValueType struct {
	t     *MapKeyTypeValueType
	nodes []*nodeMapKeyTypeValueType
	Key   KeyType
	Value *ValueType
}

// Iterator returns a new IterKeyTypeValueType.
func (t *MapKeyTypeValueType) Iterator() *IterKeyTypeValueType {
	var i IterKeyTypeValueType
	i.t = t
	return &i
}

func (i *IterKeyTypeValueType) Next() bool {
	if i.nodes == nil {
		// First use
		if i.t.length == 0 {
			return false
		}
		i.nodes = make([]*nodeMapKeyTypeValueType, 1, 64)
		i.nodes[0] = &i.t.root
	} else {
		// Go up until a left node is found
		var d = len(i.nodes)
		for &(*[2]nodeMapKeyTypeValueType)(i.nodes[d-2].child)[0] != i.nodes[d-1] {
			if d--; d < 2 {
				return false
			}
		}
		// Go right
		i.nodes[d-1] = &(*[2]nodeMapKeyTypeValueType)(i.nodes[d-2].child)[1]
		i.nodes = i.nodes[0:d]
	}
	// Go left until next leaf is found
	var next *nodeMapKeyTypeValueType = i.nodes[len(i.nodes)-1]
	for next.crit != ^uint(0) {
		next = &(*[2]nodeMapKeyTypeValueType)(next.child)[0]
		i.nodes = append(i.nodes, next)
	}
	i.Key = i.t.transformKey(next.key)
	i.Value = (*ValueType)(next.child)
	return true
}
