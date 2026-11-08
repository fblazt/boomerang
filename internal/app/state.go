package app

import (
	"github.com/fblazt/boomerang/internal/models"
)

// AppState is an alias to models.AppState for backwards compatibility.
type AppState = models.AppState

// NewAppState initializes an AppState with default values.
func NewAppState(redrawFn func()) *AppState {
	return models.NewAppState(redrawFn)
}
