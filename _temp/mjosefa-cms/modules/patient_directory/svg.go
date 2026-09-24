//go:build !wasm

package patient_directory

import (
	"webtyp.com/svg"
	"webtyp.com/svg/sprite"
)

// Icons es un tipo que aloja IconID/IconSvg para su descubrimiento y uso en config.
type Icons struct{}

func (m *Icons) IconSvg() *sprite.Sprite {
	return sprite.NewSprite(
		// Perfil de persona: figura simple para "pacientes". Un solo path,
		// currentColor — misma convención que el resto de íconos de módulo.
		sprite.Define(svg.Icon(ID), "0 0 16 16",
			sprite.Path("M8 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6zm0 1c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"),
		),
	)
}
