package kos

type WindowLayer int32

const (
	WindowLayerDesktop    WindowLayer = -2
	WindowLayerAlwaysBack WindowLayer = -1
	WindowLayerNormal     WindowLayer = 0
	WindowLayerAlwaysTop  WindowLayer = 1
)

func SetWindowLayerBehaviour(layer WindowLayer) bool {
	return SetWindowLayerBehaviourRaw(-1, int(layer)) != 0
}

func SetWindowLayerBehaviourForPID(pid int, layer WindowLayer) bool {
	return SetWindowLayerBehaviourRaw(pid, int(layer)) != 0
}
