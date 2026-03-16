package main

type pairKey struct {
	a int
	b string
}

func checkNilMapRead() (bool, string) {
	var m map[string]int
	if len(m) != 0 {
		return false, "nil map read/delete: FAIL (len)"
	}
	if v := m["missing"]; v != 0 {
		return false, "nil map read/delete: FAIL (value)"
	}
	if _, ok := m["missing"]; ok {
		return false, "nil map read/delete: FAIL (ok)"
	}
	delete(m, "missing")
	return true, "nil map read/delete: PASS"
}

func checkStringMap() (bool, string) {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	if m["a"] != 1 || m["b"] != 2 {
		return false, "string map: FAIL (insert)"
	}
	m["a"] = 3
	if m["a"] != 3 {
		return false, "string map: FAIL (update)"
	}
	delete(m, "b")
	if _, ok := m["b"]; ok {
		return false, "string map: FAIL (delete)"
	}
	return true, "string map: PASS"
}

func checkRehashInsert() (bool, string) {
	const count = 1024
	m := make(map[int32]int32)
	for i := int32(0); i < count; i++ {
		m[i] = i*3 + 1
	}
	if len(m) != int(count) {
		return false, "rehash insert: FAIL (len)"
	}
	for i := int32(0); i < count; i++ {
		if m[i] != i*3+1 {
			return false, "rehash insert: FAIL (value)"
		}
	}
	return true, "rehash insert: PASS"
}

func checkTombstoneReuse() (bool, string) {
	m := make(map[uint32]uint32)
	for i := uint32(0); i < 200; i++ {
		m[i] = i + 1
	}
	for i := uint32(0); i < 100; i++ {
		delete(m, i)
	}
	for i := uint32(200); i < 300; i++ {
		m[i] = i + 1
	}
	if len(m) != 200 {
		return false, "tombstone reuse: FAIL (len)"
	}
	for i := uint32(0); i < 100; i++ {
		if _, ok := m[i]; ok {
			return false, "tombstone reuse: FAIL (deleted)"
		}
	}
	for i := uint32(100); i < 200; i++ {
		if m[i] != i+1 {
			return false, "tombstone reuse: FAIL (kept)"
		}
	}
	for i := uint32(200); i < 300; i++ {
		if m[i] != i+1 {
			return false, "tombstone reuse: FAIL (new)"
		}
	}
	return true, "tombstone reuse: PASS"
}

func checkRangeSum() (bool, string) {
	m := map[int]int{
		1: 2,
		3: 4,
		5: 6,
	}
	sumKeys := 0
	sumValues := 0
	for k, v := range m {
		sumKeys += k
		sumValues += v
	}
	if sumKeys != 9 || sumValues != 12 {
		return false, "range: FAIL"
	}
	return true, "range: PASS"
}

func checkStructKey() (bool, string) {
	m := map[pairKey]int{}
	m[pairKey{a: 1, b: "a"}] = 10
	m[pairKey{a: 2, b: "b"}] = 20
	if m[pairKey{a: 1, b: "a"}] != 10 {
		return false, "struct key: FAIL (lookup)"
	}
	if _, ok := m[pairKey{a: 3, b: "c"}]; ok {
		return false, "struct key: FAIL (ghost)"
	}
	return true, "struct key: PASS"
}

func checkInterfaceKey() (bool, string) {
	m := map[interface{}]string{}
	m["name"] = "alpha"
	m[uint32(7)] = "seven"
	m[pairKey{a: 3, b: "x"}] = "pair"
	if m["name"] != "alpha" {
		return false, "interface key: FAIL (string)"
	}
	if m[uint32(7)] != "seven" {
		return false, "interface key: FAIL (uint32)"
	}
	if m[pairKey{a: 3, b: "x"}] != "pair" {
		return false, "interface key: FAIL (struct)"
	}
	return true, "interface key: PASS"
}

func checkFloatKey() (bool, string) {
	m := map[float64]string{}
	m[-0.0] = "zero"
	if m[0.0] != "zero" {
		return false, "float key: FAIL (+0/-0)"
	}
	m[1.5] = "onehalf"
	if m[1.5] != "onehalf" {
		return false, "float key: FAIL (value)"
	}
	return true, "float key: PASS"
}
