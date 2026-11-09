package main

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestTmpOverlap(t *testing.T) {
	type owner struct{ id, val string }
	keys := map[string][]owner{}
	for _, tw := range catalog() {
		for _, c := range tw.Changes {
			k := strings.ToLower(c.Root + `\` + c.Path + `\` + c.Name)
			keys[k] = append(keys[k], owner{tw.ID, fmt.Sprintf("%s/%d/%s", c.Kind, c.D, c.S)})
		}
		for _, n := range tw.Nvidia {
			k := fmt.Sprintf("NV:0x%X", n.ID)
			keys[k] = append(keys[k], owner{tw.ID, fmt.Sprintf("%d", n.Value)})
		}
		for _, task := range tw.Tasks {
			keys["TASK:"+strings.ToLower(task)] = append(keys["TASK:"+strings.ToLower(task)], owner{tw.ID, "off"})
		}
	}
	pairs := map[string]int{}
	detail := map[string][]string{}
	for k, os := range keys {
		if len(os) < 2 {
			continue
		}
		ids := []string{}
		for _, o := range os {
			ids = append(ids, o.id)
		}
		sort.Strings(ids)
		pk := strings.Join(ids, " + ")
		pairs[pk]++
		if len(detail[pk]) < 3 {
			vals := []string{}
			for _, o := range os {
				vals = append(vals, o.id+"="+o.val)
			}
			detail[pk] = append(detail[pk], k+" -> "+strings.Join(vals, " | "))
		}
	}
	var pk []string
	for k := range pairs {
		pk = append(pk, k)
	}
	sort.Strings(pk)
	for _, k := range pk {
		t.Logf("%s  (%d claves en común)", k, pairs[k])
		for _, d := range detail[k] {
			t.Logf("     %s", d)
		}
	}
	t.Logf("pares con solapamiento: %d", len(pk))
}
