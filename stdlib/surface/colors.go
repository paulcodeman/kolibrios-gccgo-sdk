package surface

import "kos"

const (
	Black   kos.Color = 0x000000
	Gray    kos.Color = 0x808080
	Silver  kos.Color = 0xC0C0C0
	White   kos.Color = 0xFFFFFF
	Fuchsia kos.Color = 0xFF00FF
	Purple  kos.Color = 0x800080
	Red     kos.Color = 0xFF0000
	Maroon  kos.Color = 0x800000
	Yellow  kos.Color = 0xFFFF00
	Olive   kos.Color = 0x808000
	Lime    kos.Color = 0x00FF00
	Green   kos.Color = 0x008000
	Aqua    kos.Color = 0x00FFFF
	Teal    kos.Color = 0x008080
	Blue    kos.Color = 0x0000FF
	Navy    kos.Color = 0x000080
)

func ColorValueAndAlpha(color kos.Color) (uint32, uint8) {
	value := uint32(color)
	rgb := value & 0xFFFFFF
	if value > 0xFFFFFF {
		return rgb, uint8(value >> 24)
	}
	return rgb, 255
}

func CombineAlpha(a uint8, b uint8) uint8 {
	return uint8((int(a)*int(b) + 127) / 255)
}
