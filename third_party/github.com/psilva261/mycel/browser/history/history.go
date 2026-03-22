package history

import (
	"net/url"
	"strings"
)

type History struct {
	items []Item
}

func (history History) URL() *url.URL {
	if len(history.items) == 0 {
		return nil
	}
	return history.items[len(history.items)-1].URL
}

func (history *History) Push(target *url.URL, oldScroll int) {
	if history == nil || target == nil {
		return
	}
	if len(history.items) > 0 {
		last := history.items[len(history.items)-1].URL
		if last != nil && last.String() == target.String() {
			return
		}
		history.setScroll(oldScroll)
	}
	history.items = append(history.items, Item{URL: target})
}

func (history *History) Back() {
	if history != nil && len(history.items) > 1 {
		history.items = history.items[:len(history.items)-1]
	}
}

func (history History) Scroll() int {
	if len(history.items) == 0 {
		return 0
	}
	return history.items[len(history.items)-1].Scroll
}

func (history *History) setScroll(scroll int) {
	if history == nil || len(history.items) == 0 {
		return
	}
	history.items[len(history.items)-1].Scroll = scroll
}

func (history History) String() string {
	addrs := make([]string, 0, len(history.items))
	for i := 0; i < len(history.items); i++ {
		if history.items[i].URL != nil {
			addrs = append(addrs, history.items[i].URL.String())
		}
	}
	return strings.Join(addrs, ", ")
}

type Item struct {
	*url.URL
	Scroll int
}
