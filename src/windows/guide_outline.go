package main

// A native COMBOBOX is created with a tall rectangle to make room for its
// *dropdown list*. That 180-260px height is not the visible closed selector.
// The guided-setup outline must follow the closed selection field, not the
// invisible popup area (which used to extend over the footer panel).
func guideVisibleHeight(isCombo bool, registeredHeight int) int {
	if registeredHeight <= 0 {
		return 0
	}
	if isCombo && registeredHeight > 30 {
		return 30
	}
	return registeredHeight
}
