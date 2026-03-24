package main

import (
	"strings"
	"ui"
)

type shellNodeRef struct {
	id     string
	role   string
	action string
}

func (app *App) resetShellNodeRegistry() {
	if app == nil {
		return
	}
	app.shellNodesByID = map[string]*ui.DocumentNode{}
	app.shellNodesByRole = map[string][]*ui.DocumentNode{}
	app.shellNodesByAction = map[string][]*ui.DocumentNode{}
	app.shellNodeDisplay = map[*ui.DocumentNode]ui.DisplayMode{}
	app.shellTitleNode = nil
	app.shellStatusNode = nil
	app.shellBackNode = nil
	app.shellForwardNode = nil
	app.shellReloadNode = nil
	app.shellHomeNode = nil
	app.shellAddressNode = nil
}

func (app *App) registerShellNode(source *Node, node *ui.DocumentNode, defaults shellNodeRef) *ui.DocumentNode {
	if app == nil || node == nil {
		return node
	}
	if app.shellNodesByID == nil || app.shellNodesByRole == nil || app.shellNodesByAction == nil || app.shellNodeDisplay == nil {
		app.resetShellNodeRegistry()
	}

	ref := defaults
	if source != nil {
		if ref.id == "" {
			ref.id = strings.TrimSpace(attrValue(source, "id"))
		}
		if ref.role == "" {
			ref.role = strings.TrimSpace(attrValue(source, "data-role"))
		}
		if ref.action == "" {
			ref.action = strings.TrimSpace(attrValue(source, "data-action"))
		}
	}

	if ref.id != "" {
		app.shellNodesByID[ref.id] = node
	}
	if ref.role != "" {
		app.shellNodesByRole[ref.role] = append(app.shellNodesByRole[ref.role], node)
	}
	if ref.action != "" {
		app.shellNodesByAction[ref.action] = append(app.shellNodesByAction[ref.action], node)
	}

	if display, ok := node.Style.GetDisplay(); ok {
		app.shellNodeDisplay[node] = display
	} else if node.Kind == ui.DocumentNodeText {
		app.shellNodeDisplay[node] = ui.DisplayInline
	} else {
		app.shellNodeDisplay[node] = ui.DisplayBlock
	}

	switch ref.role {
	case "title":
		app.shellTitleNode = node
	case "status":
		app.shellStatusNode = node
	case "address":
		app.shellAddressNode = node
	}
	switch ref.action {
	case "back":
		app.shellBackNode = node
	case "forward":
		app.shellForwardNode = node
	case "reload":
		app.shellReloadNode = node
	case "home":
		app.shellHomeNode = node
	}

	return node
}

func (app *App) shellNodeByID(id string) *ui.DocumentNode {
	if app == nil {
		return nil
	}
	id = strings.TrimSpace(id)
	if id == "" || app.shellNodesByID == nil {
		return nil
	}
	return app.shellNodesByID[id]
}

func (app *App) shellNodesForRole(role string) []*ui.DocumentNode {
	if app == nil {
		return nil
	}
	role = strings.TrimSpace(role)
	if role == "" || app.shellNodesByRole == nil {
		return nil
	}
	nodes := app.shellNodesByRole[role]
	if len(nodes) == 0 {
		return nil
	}
	result := make([]*ui.DocumentNode, 0, len(nodes))
	for _, node := range nodes {
		if node != nil {
			result = append(result, node)
		}
	}
	return result
}

func (app *App) shellNodesForAction(action string) []*ui.DocumentNode {
	if app == nil {
		return nil
	}
	action = strings.TrimSpace(action)
	if action == "" || app.shellNodesByAction == nil {
		return nil
	}
	nodes := app.shellNodesByAction[action]
	if len(nodes) == 0 {
		return nil
	}
	result := make([]*ui.DocumentNode, 0, len(nodes))
	for _, node := range nodes {
		if node != nil {
			result = append(result, node)
		}
	}
	return result
}

func setShellNodeText(node *ui.DocumentNode, text string) bool {
	if node == nil || node.Text == text {
		return false
	}
	node.Text = text
	return true
}

func setShellNodeValue(node *ui.DocumentNode, value string) bool {
	if node == nil || node.Value == value {
		return false
	}
	node.Value = value
	return true
}

func (app *App) setShellNodeVisible(node *ui.DocumentNode, visible bool) bool {
	if node == nil {
		return false
	}
	current, ok := node.Style.GetDisplay()
	if visible {
		target, found := app.shellNodeDisplay[node]
		if !found {
			target = ui.DisplayBlock
		}
		if ok && current == target {
			return false
		}
		node.Style.SetDisplay(target)
		return true
	}
	if ok && current == ui.DisplayNone {
		return false
	}
	node.Style.SetDisplay(ui.DisplayNone)
	return true
}

func (app *App) setShellTextByID(id string, text string) bool {
	return setShellNodeText(app.shellNodeByID(id), text)
}

func (app *App) setShellTextByRole(role string, text string) bool {
	changed := false
	for _, node := range app.shellNodesForRole(role) {
		if setShellNodeText(node, text) {
			changed = true
		}
	}
	return changed
}

func (app *App) setShellValueByID(id string, value string) bool {
	return setShellNodeValue(app.shellNodeByID(id), value)
}

func (app *App) setShellValueByRole(role string, value string) bool {
	changed := false
	for _, node := range app.shellNodesForRole(role) {
		if setShellNodeValue(node, value) {
			changed = true
		}
	}
	return changed
}

func (app *App) setShellVisibleByID(id string, visible bool) bool {
	return app.setShellNodeVisible(app.shellNodeByID(id), visible)
}

func setShellButtonEnabled(node *ui.DocumentNode, enabled bool) bool {
	if node == nil || node.Focusable == enabled {
		return false
	}
	applyShellButtonState(node, enabled)
	return true
}

func (app *App) setShellEnabledByID(id string, enabled bool) bool {
	return setShellButtonEnabled(app.shellNodeByID(id), enabled)
}

func (app *App) setShellEnabledByAction(action string, enabled bool) bool {
	changed := false
	for _, node := range app.shellNodesForAction(action) {
		if setShellButtonEnabled(node, enabled) {
			changed = true
		}
	}
	return changed
}
