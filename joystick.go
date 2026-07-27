package main

import (
	"fmt"
	"image/color"
	"math"
	"strconv"

	"gothoom/eui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	joystickWin           *eui.WindowData
	controllerDD          *eui.ItemData
	controllerEvents      *eui.EventHandler
	axesText, buttonsText *eui.ItemData
	walkStickDD           *eui.ItemData
	walkEvents            *eui.EventHandler
	cursorStickDD         *eui.ItemData
	cursorEvents          *eui.EventHandler
	walkDeadzoneSlider    *eui.ItemData
	walkDZEvents          *eui.EventHandler
	cursorDeadzoneSlider  *eui.ItemData
	cursorDZEvents        *eui.EventHandler
	click1Input           *eui.ItemData
	click2Input           *eui.ItemData
	click3Input           *eui.ItemData
	inputImgItem          *eui.ItemData
	inputImg              *ebiten.Image
	buttonTesterText      *eui.ItemData
	joystickIDs           []ebiten.GamepadID
	joystickNames         []string
	selectedJoystick      int
	lastAxisCount         int
	buttonCmdInputs       map[string]*eui.ItemData
)

const (
	joystickImgW = 290
	joystickImgH = 320
)

// joyButtonNames maps ebiten gamepad button IDs to human-readable names.
var joyButtonNames = map[ebiten.GamepadButton]string{
	ebiten.GamepadButton0:  "a",
	ebiten.GamepadButton1:  "b",
	ebiten.GamepadButton2:  "x",
	ebiten.GamepadButton3:  "y",
	ebiten.GamepadButton4:  "lb",
	ebiten.GamepadButton5:  "rb",
	ebiten.GamepadButton6:  "lt",
	ebiten.GamepadButton7:  "rt",
	ebiten.GamepadButton8:  "back",
	ebiten.GamepadButton9:  "start",
	ebiten.GamepadButton10: "l3",
	ebiten.GamepadButton11: "r3",
	ebiten.GamepadButton12: "dpad_up",
	ebiten.GamepadButton13: "dpad_right",
	ebiten.GamepadButton14: "dpad_down",
	ebiten.GamepadButton15: "dpad_left",
}

// joyButtonDisplayNames maps button names to display labels.
var joyButtonDisplayNames = map[string]string{
	"a":         "A",
	"b":         "B",
	"x":         "X",
	"y":         "Y",
	"lb":        "LB (Left Bumper)",
	"rb":        "RB (Right Bumper)",
	"lt":        "LT (Left Trigger)",
	"rt":        "RT (Right Trigger)",
	"back":      "Back",
	"start":     "Start",
	"l3":        "L3 (Left Stick)",
	"r3":        "R3 (Right Stick)",
	"dpad_up":   "D-Pad Up",
	"dpad_down": "D-Pad Down",
	"dpad_left": "D-Pad Left",
	"dpad_right":"D-Pad Right",
}

// joyButtonOrder defines the order buttons appear in the UI.
var joyButtonOrder = []string{
	"a", "b", "x", "y",
	"lb", "rb", "lt", "rt",
	"back", "start", "l3", "r3",
	"dpad_up", "dpad_down", "dpad_left", "dpad_right",
}

func makeJoystickWindow() {
	if joystickWin != nil {
		return
	}
	joystickWin = eui.NewWindow()
	joystickWin.Title = "Gamepad"
	joystickWin.Closable = true
	joystickWin.Movable = true
	joystickWin.Resizable = true
	joystickWin.AutoSize = true
	joystickWin.SetZone(eui.HZoneCenter, eui.VZoneMiddleTop)

	root := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL}
	joystickWin.AddItem(root)

	// Left column: visualizer at top, then settings
	leftCol := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL}
	root.AddItem(leftCol)

	// Right column: button commands
	rightCol := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL}
	root.AddItem(rightCol)

	joystickIDs = ebiten.AppendGamepadIDs(joystickIDs[:0])
	joystickNames = joystickNames[:0]
	for _, id := range joystickIDs {
		joystickNames = append(joystickNames, ebiten.GamepadName(id))
	}

	// Visualizer image at the top
	inputImgItem, inputImg = eui.NewImageItem(joystickImgW, joystickImgH)
	leftCol.AddItem(inputImgItem)

	// Button tester: shows last pressed button name + number
	buttonTesterText, _ = eui.NewText()
	buttonTesterText.Text = "Press any button to test"
	buttonTesterText.FontSize = 11
	buttonTesterText.Size = eui.Point{X: 290, Y: 20}
	leftCol.AddItem(buttonTesterText)

	// Spacer
	spacer := func() {
		s, _ := eui.NewText()
		s.Size = eui.Point{X: 290, Y: 6}
		leftCol.AddItem(s)
	}
	spacer()

	controllerDD, controllerEvents = eui.NewDropdown()
	controllerDD.Label = "Controller"
	controllerDD.Options = joystickNames
	controllerDD.Size = eui.Point{X: 290, Y: 24}
	controllerEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventDropdownSelected {
			selectedJoystick = ev.Index
			if selectedJoystick >= 0 && selectedJoystick < len(joystickIDs) {
				updateStickOptions(ebiten.GamepadAxisCount(joystickIDs[selectedJoystick]))
			} else {
				updateStickOptions(0)
			}
		}
	}
	leftCol.AddItem(controllerDD)

	refreshBtn, refreshEvents := eui.NewButton()
	refreshBtn.Text = "Refresh"
	refreshBtn.Size = eui.Point{X: 80, Y: 24}
	refreshEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			joystickIDs = ebiten.AppendGamepadIDs(joystickIDs[:0])
			joystickNames = joystickNames[:0]
			for _, id := range joystickIDs {
				joystickNames = append(joystickNames, ebiten.GamepadName(id))
			}
			controllerDD.Options = joystickNames
			controllerDD.Dirty = true
			if selectedJoystick >= len(joystickIDs) {
				selectedJoystick = len(joystickIDs) - 1
				controllerDD.Selected = selectedJoystick
			}
			if selectedJoystick >= 0 && selectedJoystick < len(joystickIDs) {
				updateStickOptions(ebiten.GamepadAxisCount(joystickIDs[selectedJoystick]))
			} else {
				updateStickOptions(0)
			}
			joystickWin.Refresh()
		}
	}
	leftCol.AddItem(refreshBtn)

	axesText, _ = eui.NewText()
	axesText.FontSize = 12
	axesText.Size = eui.Point{X: 290, Y: 24}
	leftCol.AddItem(axesText)

	buttonsText, _ = eui.NewText()
	buttonsText.FontSize = 12
	buttonsText.Size = eui.Point{X: 290, Y: 24}
	leftCol.AddItem(buttonsText)

	spacer()

	enableCB, enableEvents := eui.NewCheckbox()
	enableCB.Text = "Enable Gamepad"
	enableCB.Checked = gs.JoystickEnabled
	enableEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventCheckboxChanged {
			gs.JoystickEnabled = ev.Checked
			settingsDirty = true
		}
	}
	leftCol.AddItem(enableCB)

	spacer()

	walkStickDD, walkEvents = eui.NewDropdown()
	walkStickDD.Label = "Walk Stick"
	walkStickDD.Size = eui.Point{X: 290, Y: 24}
	walkEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventDropdownSelected {
			gs.JoystickWalkStick = ev.Index - 1
			settingsDirty = true
		}
	}
	leftCol.AddItem(walkStickDD)

	walkDeadzoneSlider, walkDZEvents = eui.NewSlider()
	walkDeadzoneSlider.Label = "Walk Deadzone"
	walkDeadzoneSlider.MinValue = 0.01
	walkDeadzoneSlider.MaxValue = 0.2
	walkDeadzoneSlider.Value = float32(gs.JoystickWalkDeadzone)
	walkDeadzoneSlider.Size = eui.Point{X: 290, Y: 24}
	walkDZEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventSliderChanged {
			gs.JoystickWalkDeadzone = float64(ev.Value)
			settingsDirty = true
		}
	}
	leftCol.AddItem(walkDeadzoneSlider)

	spacer()

	cursorStickDD, cursorEvents = eui.NewDropdown()
	cursorStickDD.Label = "Cursor Stick"
	cursorStickDD.Size = eui.Point{X: 290, Y: 24}
	cursorEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventDropdownSelected {
			gs.JoystickCursorStick = ev.Index - 1
			settingsDirty = true
		}
	}
	leftCol.AddItem(cursorStickDD)

	cursorDeadzoneSlider, cursorDZEvents = eui.NewSlider()
	cursorDeadzoneSlider.Label = "Cursor Deadzone"
	cursorDeadzoneSlider.MinValue = 0.01
	cursorDeadzoneSlider.MaxValue = 0.2
	cursorDeadzoneSlider.Value = float32(gs.JoystickCursorDeadzone)
	cursorDeadzoneSlider.Size = eui.Point{X: 290, Y: 24}
	cursorDZEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventSliderChanged {
			gs.JoystickCursorDeadzone = float64(ev.Value)
			settingsDirty = true
		}
	}
	leftCol.AddItem(cursorDeadzoneSlider)

	spacer()

	click1Input, _ = eui.NewInput()
	click1Input.Label = "Click1 Button"
	click1Input.Size = eui.Point{X: 290, Y: 24}
	leftCol.AddItem(click1Input)

	click2Input, _ = eui.NewInput()
	click2Input.Label = "Click2 Button"
	click2Input.Size = eui.Point{X: 290, Y: 24}
	leftCol.AddItem(click2Input)

	click3Input, _ = eui.NewInput()
	click3Input.Label = "Click3 Button"
	click3Input.Size = eui.Point{X: 290, Y: 24}
	leftCol.AddItem(click3Input)
	cmdHeader, _ := eui.NewText()
	cmdHeader.Text = "Button Commands"
	cmdHeader.FontSize = 12
	cmdHeader.Size = eui.Point{X: 290, Y: 20}
	rightCol.AddItem(cmdHeader)

	cmdSubHeader, _ := eui.NewText()
	cmdSubHeader.Text = "Enter a command (/sit) or macro name (heal)"
	cmdSubHeader.FontSize = 10
	cmdSubHeader.Size = eui.Point{X: 290, Y: 16}
	rightCol.AddItem(cmdSubHeader)

	buttonCmdInputs = make(map[string]*eui.ItemData)
	if gs.JoystickCommands == nil {
		gs.JoystickCommands = make(map[string]string)
	}
	for _, name := range joyButtonOrder {
		display := joyButtonDisplayNames[name]
		if display == "" {
			display = name
		}
		input, events := eui.NewInput()
		input.Label = display
		input.Size = eui.Point{X: 290, Y: 24}
		input.FontSize = 10
		if cmd, ok := gs.JoystickCommands[name]; ok {
			input.Text = cmd
		}
		btnName := name
		events.Handle = func(ev eui.UIEvent) {
			if ev.Type == eui.EventInputChanged {
				gs.JoystickCommands[btnName] = ev.Text
				settingsDirty = true
			}
		}
		buttonCmdInputs[name] = input
		rightCol.AddItem(input)
	}

	if gs.JoystickBindings != nil {
		if b, ok := gs.JoystickBindings["click1"]; ok {
			click1Input.Text = strconv.Itoa(int(b))
		}
		if b, ok := gs.JoystickBindings["click2"]; ok {
			click2Input.Text = strconv.Itoa(int(b))
		}
		if b, ok := gs.JoystickBindings["click3"]; ok {
			click3Input.Text = strconv.Itoa(int(b))
		}
	}

	if len(joystickIDs) > 0 {
		updateStickOptions(ebiten.GamepadAxisCount(joystickIDs[selectedJoystick]))
	} else {
		updateStickOptions(0)
	}

	joystickWin.AddWindow(false)
}

func updateStickOptions(axisCount int) {
	stickCount := axisCount / 2
	opts := make([]string, stickCount+1)
	opts[0] = "none"
	for i := 0; i < stickCount; i++ {
		opts[i+1] = fmt.Sprintf("Stick %d", i+1)
	}
	if walkStickDD != nil {
		walkStickDD.Options = opts
		walkStickDD.Disabled = stickCount == 0
		if gs.JoystickWalkStick >= stickCount {
			gs.JoystickWalkStick = stickCount - 1
		}
		if gs.JoystickWalkStick < -1 {
			gs.JoystickWalkStick = -1
		}
		walkStickDD.Selected = gs.JoystickWalkStick + 1
		walkStickDD.Dirty = true
	}
	if cursorStickDD != nil {
		cursorStickDD.Options = opts
		cursorStickDD.Disabled = stickCount == 0
		if gs.JoystickCursorStick >= stickCount {
			gs.JoystickCursorStick = stickCount - 1
		}
		if gs.JoystickCursorStick < -1 {
			gs.JoystickCursorStick = -1
		}
		cursorStickDD.Selected = gs.JoystickCursorStick + 1
		cursorStickDD.Dirty = true
	}
	lastAxisCount = axisCount
	if joystickWin != nil {
		joystickWin.Refresh()
	}
}

func drawJoystickDisplay(id ebiten.GamepadID) {
	if inputImg == nil {
		return
	}
	inputImg.Clear()
	imgW := float32(joystickImgW)

	axisCount := ebiten.GamepadAxisCount(id)
	buttonCount := ebiten.GamepadButtonCount(id)

	isPressed := func(b int) bool {
		if b >= buttonCount {
			return false
		}
		return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton(b))
	}

	// Controller body outline
	bodyCol := color.NRGBA{50, 50, 60, 255}
	bodyX, bodyY := float32(20), float32(20)
	bodyW, bodyH := imgW-40, float32(180)
	vector.FillRect(inputImg, bodyX, bodyY, bodyW, bodyH, bodyCol, true)
	vector.StrokeRect(inputImg, bodyX, bodyY, bodyW, bodyH, 1, color.NRGBA{90, 90, 100, 255}, false)

	// Grips (angled rectangles on bottom-left and bottom-right)
	gripCol := color.NRGBA{45, 45, 55, 255}
	vector.FillRect(inputImg, bodyX+2, bodyY+bodyH-10, 40, 30, gripCol, true)
	vector.FillRect(inputImg, bodyX+bodyW-42, bodyY+bodyH-10, 40, 30, gripCol, true)

	// Helper to draw a round button with label + number
	drawRoundBtn := func(cx, cy, r float32, btnIdx int, lbl string) {
		bg := color.NRGBA{60, 60, 70, 255}
		outline := color.NRGBA{90, 90, 100, 255}
		if isPressed(btnIdx) {
			bg = color.NRGBA{0, 200, 0, 255}
			outline = color.NRGBA{0, 255, 0, 255}
		}
		vector.FillCircle(inputImg, cx, cy, r, bg, true)
		vector.StrokeCircle(inputImg, cx, cy, r, 1, outline, false)
		sc := float32(0.5)
		if len(lbl) > 1 {
			sc = 0.4
		}
		tw, _ := text.Measure(lbl, mainFont, 0)
		op := &text.DrawOptions{}
		op.GeoM.Scale(float64(sc), float64(sc))
		op.GeoM.Translate(float64(cx-float32(tw)*sc/2), float64(cy-4*sc))
		text.Draw(inputImg, lbl, mainFont, op)
		numStr := fmt.Sprintf("#%d", btnIdx)
		nw, _ := text.Measure(numStr, mainFont, 0)
		nop := &text.DrawOptions{}
		nop.GeoM.Scale(0.3, 0.3)
		nop.GeoM.Translate(float64(cx-float32(nw)*0.15), float64(cy+r+2))
		text.Draw(inputImg, numStr, mainFont, nop)
	}

	// Draw a dpad cross
	drawDpad := func(cx, cy, size float32) {
		pressed := color.NRGBA{0, 200, 0, 255}
		off := color.NRGBA{55, 55, 65, 255}
		half := size / 2
		arm := size / 3
		// Vertical arm
		vCol := off
		if isPressed(12) || isPressed(14) {
			vCol = pressed
		}
		vector.FillRect(inputImg, cx-arm/2, cy-half, arm, size, vCol, true)
		// Horizontal arm
		hCol := off
		if isPressed(13) || isPressed(15) {
			hCol = pressed
		}
		vector.FillRect(inputImg, cx-half, cy-arm/2, size, arm, hCol, true)
		// Direction labels
		sc := float32(0.35)
		type dir struct {
			lbl   string
			dx, dy, btn int
		}
		dirs := []dir{
			{"U", 0, -1, 12}, {"D", 0, 1, 14}, {"L", -1, 0, 15}, {"R", 1, 0, 13},
		}
		for _, d := range dirs {
			lx := cx + float32(d.dx)*(half+3)
			ly := cy + float32(d.dy)*(half+3)
			lw, _ := text.Measure(d.lbl, mainFont, 0)
			dop := &text.DrawOptions{}
			dop.GeoM.Scale(float64(sc), float64(sc))
			dop.ColorScale.ScaleWithColor(color.NRGBA{120, 120, 130, 255})
			if isPressed(d.btn) {
				dop.ColorScale.ScaleWithColor(color.NRGBA{0, 255, 0, 255})
			}
			dop.GeoM.Translate(float64(lx-float32(lw)*sc/2), float64(ly-4*sc))
			text.Draw(inputImg, d.lbl, mainFont, dop)
		}
	}

	// Draw stick with deadzone
	drawStick := func(cx, cy, stickIdx float32, dz float64) {
		r := float32(22)
		// Stick base
		vector.FillCircle(inputImg, cx, cy, r, color.NRGBA{35, 35, 40, 255}, true)
		vector.StrokeCircle(inputImg, cx, cy, r, 1, color.NRGBA{70, 70, 80, 255}, false)
		// Deadzone ring
		if dz > 0 {
			dzR := float32(dz) * r
			vector.StrokeCircle(inputImg, cx, cy, dzR, 1, color.NRGBA{200, 80, 80, 200}, false)
		}
		// Stick position
		if stickIdx >= 0 {
			ai := int(stickIdx) * 2
			if ai+1 < axisCount {
				ax := float32(ebiten.GamepadAxisValue(id, ai))
				ay := float32(ebiten.GamepadAxisValue(id, ai+1))
				dist := float32(math.Sqrt(float64(ax*ax + ay*ay)))
				if dist > 1 {
					ax /= dist
					ay /= dist
					dist = 1
				}
				dotCol := color.NRGBA{0, 200, 0, 255}
				if dist < float32(dz) {
					dotCol = color.NRGBA{200, 60, 60, 255}
				}
				vector.FillCircle(inputImg, cx+ax*r*0.8, cy+ay*r*0.8, 5, dotCol, true)
				vector.FillCircle(inputImg, cx+ax*r*0.8, cy+ay*r*0.8, 2, color.NRGBA{255, 255, 255, 255}, true)
			}
		}
	}

	// Positions (relative to body)
	lbX, lbY := bodyX+35, bodyY+12
	rbX := bodyX+bodyW-35
	rbY := lbY
	ltX := bodyX+15
	rtX := bodyX+bodyW-15

	// Bumpers
	drawRoundBtn(lbX, lbY, 10, 4, "LB")
	drawRoundBtn(rbX, rbY, 10, 5, "RB")

	// Triggers
	drawRoundBtn(ltX, lbY-4, 8, 6, "LT")
	drawRoundBtn(rtX, rbY-4, 8, 7, "RT")

	// Left stick + D-pad
	lStickX := bodyX + 55
	lStickY := bodyY + 70
	drawStick(lStickX, lStickY, float32(gs.JoystickWalkStick), gs.JoystickWalkDeadzone)
	// L3
	if buttonCount > 10 {
		drawRoundBtn(lStickX, lStickY-30, 6, 10, "L3")
	}

	dpadX := bodyX + 110
	dpadY := bodyY + 100
	drawDpad(dpadX, dpadY, 36)

	// Center buttons
	drawRoundBtn(bodyX+bodyW/2-14, bodyY+35, 7, 8, "Bk")
	drawRoundBtn(bodyX+bodyW/2+14, bodyY+35, 7, 9, "St")

	// Right stick + face buttons
	rStickX := bodyX + bodyW - 55
	rStickY := bodyY + 70
	drawStick(rStickX, rStickY, float32(gs.JoystickCursorStick), gs.JoystickCursorDeadzone)
	// R3
	if buttonCount > 11 {
		drawRoundBtn(rStickX, rStickY-30, 6, 11, "R3")
	}

	// Face buttons: A=0, B=1, X=2, Y=3 in diamond
	faceX := bodyX + bodyW - 110
	faceY := bodyY + 75
	faceR := float32(11)
	drawRoundBtn(faceX, faceY+18, faceR, 0, "A")     // bottom
	drawRoundBtn(faceX+18, faceY, faceR, 1, "B")     // right
	drawRoundBtn(faceX-18, faceY, faceR, 2, "X")     // left
	drawRoundBtn(faceX, faceY-18, faceR, 3, "Y")     // top

	// Axis bars at the bottom
	barY0 := bodyY + bodyH + 8
	barTitleOp := &text.DrawOptions{}
	barTitleOp.GeoM.Scale(0.5, 0.5)
	barTitleOp.GeoM.Translate(4, float64(barY0+10))
	text.Draw(inputImg, fmt.Sprintf("Axes (%d)  Buttons (%d)", axisCount, buttonCount), mainFont, barTitleOp)

	barY := barY0 + 16
	for a := 0; a < axisCount; a++ {
		ay := barY + float32(a*14)
		val := ebiten.GamepadAxisValue(id, a)
		vector.FillRect(inputImg, 4, ay, imgW-8, 10, color.NRGBA{40, 40, 40, 255}, true)
		vector.FillRect(inputImg, float32(int(imgW/2-1)), ay, 2, 10, color.NRGBA{80, 80, 80, 255}, true)
		barMid := (imgW - 8) / 2
		barLen := float32(val) * barMid
		barX := 4 + barMid
		if barLen < 0 {
			barX = barX + barLen
			barLen = -barLen
		}
		barCol := color.NRGBA{0, 150, 0, 255}
		if math.Abs(float64(val)) < 0.01 {
			barCol = color.NRGBA{60, 60, 60, 255}
		}
		vector.FillRect(inputImg, barX, ay+1, barLen, 8, barCol, true)
		lblOp := &text.DrawOptions{}
		lblOp.GeoM.Scale(0.35, 0.35)
		lblOp.GeoM.Translate(float64(imgW-45), float64(ay+8))
		text.Draw(inputImg, fmt.Sprintf("%d:%.2f", a, val), mainFont, lblOp)
	}

	inputImgItem.Dirty = true
}

func updateJoystickWindow() {
	joystickWin.Refresh()
	newIDs := inpututil.AppendJustConnectedGamepadIDs(nil)
	if len(newIDs) > 0 {
		for _, id := range newIDs {
			joystickIDs = append(joystickIDs, id)
			joystickNames = append(joystickNames, ebiten.GamepadName(id))
		}
		if controllerDD != nil {
			controllerDD.Options = joystickNames
			controllerDD.Dirty = true
			joystickWin.Refresh()
		}
	}
	for i := 0; i < len(joystickIDs); {
		id := joystickIDs[i]
		if inpututil.IsGamepadJustDisconnected(id) {
			joystickIDs = append(joystickIDs[:i], joystickIDs[i+1:]...)
			joystickNames = append(joystickNames[:i], joystickNames[i+1:]...)
			if controllerDD != nil {
				controllerDD.Options = joystickNames
				controllerDD.Dirty = true
				if selectedJoystick >= len(joystickIDs) {
					selectedJoystick = len(joystickIDs) - 1
					controllerDD.Selected = selectedJoystick
				}
				joystickWin.Refresh()
			}
			continue
		}
		i++
	}

	if len(joystickIDs) == 0 {
		updateStickOptions(0)
	}

	if selectedJoystick < 0 || selectedJoystick >= len(joystickIDs) {
		updateStickOptions(0)
		axesText.Text = ""
		buttonsText.Text = ""
		axesText.Dirty = true
		buttonsText.Dirty = true
		return
	}

	id := joystickIDs[selectedJoystick]

	axisCount := ebiten.GamepadAxisCount(id)
	if axisCount != lastAxisCount {
		updateStickOptions(axisCount)
	}

	buttonCount := ebiten.GamepadButtonCount(id)
	axesText.Text = fmt.Sprintf("Axes: %d  Buttons: %d", axisCount, buttonCount)
	axesText.Dirty = true

	// Show which sticks are mapped to which axes
	walkInfo := "none"
	if gs.JoystickWalkStick >= 0 {
		walkInfo = fmt.Sprintf("Stick %d (axes %d,%d)", gs.JoystickWalkStick+1, gs.JoystickWalkStick*2, gs.JoystickWalkStick*2+1)
	}
	cursorInfo := "none"
	if gs.JoystickCursorStick >= 0 {
		cursorInfo = fmt.Sprintf("Stick %d (axes %d,%d)", gs.JoystickCursorStick+1, gs.JoystickCursorStick*2, gs.JoystickCursorStick*2+1)
	}
	buttonsText.Text = fmt.Sprintf("Walk: %s  Cursor: %s", walkInfo, cursorInfo)
	buttonsText.Dirty = true

	btns := inpututil.AppendJustPressedGamepadButtons(id, nil)
	if len(btns) > 0 {
		if buttonTesterText != nil {
			for _, btn := range btns {
				num := int(btn)
				name := joyButtonNames[btn]
				if name != "" {
					if display, ok := joyButtonDisplayNames[name]; ok {
						buttonTesterText.Text = fmt.Sprintf("Last pressed: %s (button %d)", display, num)
					} else {
						buttonTesterText.Text = fmt.Sprintf("Last pressed: %s (button %d)", name, num)
					}
				} else {
					buttonTesterText.Text = fmt.Sprintf("Last pressed: button %d", num)
				}
				buttonTesterText.Dirty = true
			}
		}

		if click1Input != nil && click1Input.Focused {
			if gs.JoystickBindings == nil {
				gs.JoystickBindings = make(map[string]ebiten.GamepadButton)
			}
			gs.JoystickBindings["click1"] = btns[len(btns)-1]
			click1Input.Text = strconv.Itoa(int(gs.JoystickBindings["click1"]))
			click1Input.Dirty = true
			settingsDirty = true
		} else if click2Input != nil && click2Input.Focused {
			if gs.JoystickBindings == nil {
				gs.JoystickBindings = make(map[string]ebiten.GamepadButton)
			}
			gs.JoystickBindings["click2"] = btns[len(btns)-1]
			click2Input.Text = strconv.Itoa(int(gs.JoystickBindings["click2"]))
			click2Input.Dirty = true
			settingsDirty = true
		} else if click3Input != nil && click3Input.Focused {
			if gs.JoystickBindings == nil {
				gs.JoystickBindings = make(map[string]ebiten.GamepadButton)
			}
			gs.JoystickBindings["click3"] = btns[len(btns)-1]
			click3Input.Text = strconv.Itoa(int(gs.JoystickBindings["click3"]))
			click3Input.Dirty = true
			settingsDirty = true
		}
	}

	drawJoystickDisplay(id)
}

// joystickButtonByName converts a button name string (e.g., "a", "dpad_up")
// to its ebiten gamepad button integer, or -1 if not found.
func joystickButtonByName(name string) int {
	for btn, n := range joyButtonNames {
		if n == name {
			return int(btn)
		}
	}
	return -1
}
