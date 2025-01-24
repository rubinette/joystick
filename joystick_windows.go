//go:build windows
// +build windows

package joystick

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// XINPUT_DEADZONE_LEFT_THUMB Deadzone for the Left Thumb Stick (-32767 to 32767)
	XINPUT_DEADZONE_LEFT_THUMB = 7849
	// XINPUT_DEADZONE_RIGHT_THUMB Deadzone for the Right Thumb Stick (-32767 to 32767)
	XINPUT_DEADZONE_RIGHT_THUMB = 8689
	// TRIGGER_TRESHOLD Threshold for the left and right triggers (0 to 255)
	XINPUT_DEADZONE_TRIGGER = 30
)

type XInputState struct {
	PacketNumber uint32
	Gamepad      XInputGamepad
}

type XInputGamepad struct {
	Buttons      uint16
	LeftTrigger  uint8
	RightTrigger uint8
	ThumbLX      int16
	ThumbLY      int16
	ThumbRX      int16
	ThumbRY      int16
}

type joystickImpl struct {
	id          int
	axisCount   int
	buttonCount int
	state       XInputState
	button      int
	axes        []float64
}

var (
	xinputDLL      = windows.MustLoadDLL("xinput1_4.dll") // Use xinput1_4 (available on Windows 8+)
	xinputGetState = xinputDLL.MustFindProc("XInputGetState")
	xinputSetState = xinputDLL.MustFindProc("XInputSetState")
	xinputEnable   = xinputDLL.MustFindProc("XInputEnable")
)

func Open(id int) (Joystick, error) {
	if id < 0 || id > 3 {
		return nil, fmt.Errorf("invalid joystick id: %d", id)
	}
	return &joystickImpl{id: id, axisCount: 6, buttonCount: 16}, nil
}

func (js *joystickImpl) Read() (State, error) {
	ret, _, _ := xinputGetState.Call(uintptr(js.id), uintptr(unsafe.Pointer(&js.state)))
	if ret != 0 {
		return State{}, fmt.Errorf("joystick %d is not connected", js.id)
	}

	gamepad := js.state.Gamepad
	state := State{
		Buttons: uint32(gamepad.Buttons),
		AxisData: []int{
			applyDeadzone(int(gamepad.ThumbLX), XINPUT_DEADZONE_LEFT_THUMB),
			applyDeadzone(int(gamepad.ThumbLY), XINPUT_DEADZONE_LEFT_THUMB),
			applyDeadzone(int(gamepad.ThumbRX), XINPUT_DEADZONE_RIGHT_THUMB),
			applyDeadzone(int(gamepad.ThumbRY), XINPUT_DEADZONE_RIGHT_THUMB),
			applyTriggerDeadzone(int(gamepad.LeftTrigger), XINPUT_DEADZONE_TRIGGER),
			applyTriggerDeadzone(int(gamepad.RightTrigger), XINPUT_DEADZONE_TRIGGER),
		},
	}
	return state, nil
}

func applyDeadzone(value int, deadzone int) int {
	if value > deadzone {
		return scaleValue(value, deadzone, 32767, 0, 32767)
	} else if value < -deadzone {
		return scaleValue(value, -32767, -deadzone, -32767, 0)
	}
	return 0
}

func applyTriggerDeadzone(value int, deadzone int) int {
	if value > deadzone {
		return scaleValue(value, deadzone, 255, 0, 255)
	}
	return 0
}

func scaleValue(value, srcMin, srcMax, tgtMin, tgtMax int) int {
	// Handle edge case: zero source range
	srcRange := srcMax - srcMin
	if srcRange == 0 {
		panic("source range cannot be zero")
	}

	tgtRange := tgtMax - tgtMin

	// Scale the value
	scaledValue := (value-srcMin)*tgtRange/srcRange + tgtMin

	return scaledValue
}

func (js *joystickImpl) Close() {
	// No cleanup needed for XInput
}

func (js *joystickImpl) AxisCount() int {
	return js.axisCount // Two thumbsticks (2x2) + two triggers
}

func (js *joystickImpl) ButtonCount() int {
	return js.buttonCount // Includes A, B, X, Y, Start, Back, etc.
}

func (js *joystickImpl) Name() string {
	return fmt.Sprintf("XInput Controller %d", js.id)
}
