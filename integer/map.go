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

// NewMapKeyTypeValueType returns a new map with keys of type KeyType and values of type ValueType
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
// As third value, the parent of the child is returned. If the child is the receiver of the method
// parent is nil.
func (c *nodeMapKeyTypeValueType) find(key KeyType) (crit uint, child, parent *nodeMapKeyTypeValueType) {
	child = c
	crit = child.findCrit(key)
	// Keep going deeper until a leaf or an incompatible range is found.
	for child.crit != ^uint(0) && child.crit == crit {
		parent = child
		child = &(child.children())[child.dir(key)]
		crit = child.findCrit(key)
	}
	return
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

// Rem removes the value associated with the specified key from the map.
func (t *MapKeyTypeValueType) Rem(key KeyType) {
	if t.length == 0 {
		return
	}
	key = t.transformKey(key)
	var crit, _, parent = t.root.find(key)
	if crit == ^uint(0) {
		if parent != nil {
			*parent = parent.children()[1-parent.dir(key)]
		}
		t.length--
	}
}

// SetP inserts or replaces the value associated with the specified key.
// The specified value pointer can be used to modify the value without using Set.
func (t *MapKeyTypeValueType) SetP(key KeyType, val *ValueType) {
	if val == nil {
		t.Rem(key)
		return
	}
	key = t.transformKey(key)
	// Make leaf node if tree is empty
	if t.length == 0 {
		t.length++
		t.root.key = key
		t.root.crit = ^uint(0)
		t.root.child = unsafe.Pointer(val)
		return
	}
	// Find node with longest shared prefix and critical bit
	var crit, n, _ = t.root.find(key)
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
	t.length++
}

// Set inserts or replaces the value associated with the specified key.
func (t *MapKeyTypeValueType) Set(key KeyType, val ValueType) {
	t.SetP(key, &val)
}

// Get returns the internal pointer to the value associated with the specified key.
// If the there is no such key it returns nil. The pointer can be used to modify the value without using Set.
func (t *MapKeyTypeValueType) GetP(key KeyType) *ValueType {
	key = t.transformKey(key)
	if t.length > 0 {
		// Find leaf node
		var crit, l, _ = t.root.find(key)
		if crit == ^uint(0) {
			return (*ValueType)(l.child)
		}
	}
	return nil
}

// Get returns the value associated with the specified key and true if the key exists.
// Otherwise 0 and false are returned. If a nil pointer was associated with the key,
// Get will panic (use GetP instead).
func (t *MapKeyTypeValueType) Get(key KeyType) (ValueType, bool) {
	var v = t.GetP(key)
	if v == nil {
		var zero ValueType
		return zero, false
	}
	return *v, true
}

// Length returns the number of distinct keys in the multiset
func (t *MapKeyTypeValueType) Length() int {
	return t.length
}

// func (c *nodeMapKeyTypeValueType) dbg(p string) {
// 	if c.crit != ^uint(0) {
// 		fmt.Printf(p+"Node: %08b %d\n", ((c.key>>c.crit)|1)<<c.crit, c.crit)
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
	t       *MapKeyTypeValueType
	nodes   []*nodeMapKeyTypeValueType
	lastDir int
	Key     KeyType    // Key found by last call to Next, Prev.
	Value   *ValueType // Initially nil (also after calling Reset). Otherwise Pointer to value associated with key found by last call to Next, Prev (nil if no key was found).
}

// Iterator returns a new IterKeyTypeValueType.
func (t *MapKeyTypeValueType) Iterator() *IterKeyTypeValueType {
	var i IterKeyTypeValueType
	i.t = t
	i.Reset()
	return &i
}

// Seek initializes the iterator in a state that will be advanced to the specified key
// on the next call to Prev or Next. If the key does not exist, the next call to Prev or Next
// will advance the iterator to the next lower or higher key respectively (or the respective end of the map).
func (i *IterKeyTypeValueType) Seek(key KeyType) {
	key = i.t.transformKey(key)
	i.Reset()
	if i.t.length == 0 {
		return
	}
	var last = &i.t.root
	for last.crit != ^uint(0) && last.findCrit(key) == last.crit {
		i.nodes = append(i.nodes, last)
		last = &last.children()[last.dir(key)]
	}
	if last.crit != ^uint(0) || key != last.key {
		// Key not found.
		if len(i.nodes) == 1 {
			// Didn't get beyond root node. There are no keys this high or low. Simulate a Next or Prev call that reached the end
			i.lastDir = 1
			if key > last.key {
				i.lastDir = 0
			}
			i.nodes = i.nodes[0:0]
		}
	} else {
		// Key found
		i.nodes = append(i.nodes, last)
	}
}

// Reset restores the iterator to the initial state.
func (i *IterKeyTypeValueType) Reset() {
	i.Key = i.t.transformKey(0)
	i.Value = nil
	i.lastDir = 2
	if i.nodes == nil {
		i.nodes = make([]*nodeMapKeyTypeValueType, 0, 64)
	} else {
		i.nodes = i.nodes[0:0]
	}
}

// Next advances the iterator to the next higher key and populates the iterators public Fields.
// If the iterator is in the initial state, the first call to Next will set the iterator to the lowest key.
// The return value is true unless there is no next higher key to advance to.
func (i *IterKeyTypeValueType) Next() bool {
	i.step(1)
	return i.Value != nil
}

// Prev advances the iterator to the next lower key and populates the iterators public Fields.
// If the iterator is in the initial state, the first call to Prev will set the iterator to the highest key.
// The return value is true unless there is no next lower key to advance to.
func (i *IterKeyTypeValueType) Prev() bool {
	i.step(0)
	return i.Value != nil
}

func (i *IterKeyTypeValueType) step(dir int) {
	// Check if iterator is at some node from a Seek, a leaf from step or at an end
	if len(i.nodes) == 0 {
		// Iterator is at end of map.
		if i.lastDir != dir && i.t.length > 0 {
			// Direction changed or not defined yet. Use root as starting point.
			i.lastDir = dir
			i.nodes = append(i.nodes, &i.t.root)
		} else {
			// End of map.
			i.Value = nil
			return
		}
	} else if i.lastDir == 2 {
		// At node from Seek. Do nothing if this is a leaf. Otherwise take one step in the opposite direction of dir.
		if current := i.nodes[len(i.nodes)-1]; current.crit != ^uint(0) {
			i.nodes = append(i.nodes, &current.children()[dir])
		}
	} else {
		// Iterator is at some leaf from previous call to step. Comments describe behavior with dir == 1 (left to right).
		// Go up until we are at a left child. Then go to the sibling.
		for {
			// Check if there is a parent
			if len(i.nodes) == 1 {
				// No parent. Set end of map state.
				i.nodes = i.nodes[0:0]
				i.Value = nil
				return
			}
			// If current node is left, replace it with the right one and stop going up.
			var rigthChild = &i.nodes[len(i.nodes)-2].children()[dir]
			if rigthChild != i.nodes[len(i.nodes)-1] {
				i.nodes[len(i.nodes)-1] = rigthChild
				break
			}
			// Go up
			i.nodes[len(i.nodes)-1] = nil // Help gc
			i.nodes = i.nodes[0 : len(i.nodes)-1]
		}
	}
	// Find next leaf by walking in correct direction. Comments describe behavior with dir == 1 (left to right).
	// Go left until next leaf is found.
	var current = i.nodes[len(i.nodes)-1]
	for current.crit != ^uint(0) {
		current = &current.children()[1-dir]
		i.nodes = append(i.nodes, current)
	}
	// Found leaf. Store data.
	i.Key = i.t.transformKey(current.key)
	i.Value = current.value()
}
