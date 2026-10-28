package main

// manualPageControlsVisible is the one rule for page-specific manual controls.
// Selecting a Field Manual page in the guided setup does not imply a tab switch.
// In particular, SAVE TO DISK on Settings may preselect the narrated guide,
// but that guide must remain hidden until Field Manual is the active tab.
func manualPageControlsVisible(activeScreen, selectedPage, controlPage string) bool {
	return activeScreen == "manual" && selectedPage == controlPage
}
