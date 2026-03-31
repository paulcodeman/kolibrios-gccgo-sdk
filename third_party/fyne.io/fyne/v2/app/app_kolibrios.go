//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/driver/kolibrios"
)

// NewWithID returns a new app instance using the KolibriOS software driver.
func NewWithID(id string) fyne.App {
	return newAppWithDriver(kolibrios.NewDriver(), id)
}
