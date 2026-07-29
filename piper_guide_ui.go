package main

import (
	"image"
	_ "image/png"
	"os"
	"strconv"

	"gothoom/eui"
)

var (
	piperGuideWin    *eui.WindowData
	piperGuideImages []*eui.ItemData
	piperGuidePage   int
	piperGuideImg    *eui.ItemData
	piperGuideLabel  *eui.ItemData
	piperGuidePrev   *eui.ItemData
	piperGuideNext   *eui.ItemData
)

func loadPiperGuideImages() bool {
	if piperGuideImages != nil {
		return true
	}
	dir := "piper_voice_add_tut"
	names := []string{"one.png", "two.png", "three.png"}
	for _, n := range names {
		p := dir + "/" + n
		f, err := os.Open(p)
		if err != nil {
			return false
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return false
		}
		item, _ := eui.NewImageItem(img.Bounds().Dx(), img.Bounds().Dy())
		item.Image = newImageFromImage(img)
		piperGuideImages = append(piperGuideImages, item)
	}
	return len(piperGuideImages) == 3
}

func showPiperGuidePage(page int) {
	if page < 0 {
		page = 0
	}
	if page >= len(piperGuideImages) {
		page = len(piperGuideImages) - 1
	}
	piperGuidePage = page

	piperGuideImg.Image = piperGuideImages[page].Image
	iw := piperGuideImages[page].Image.Bounds().Dx()
	ih := piperGuideImages[page].Image.Bounds().Dy()
	const maxW float32 = 640
	const maxH float32 = 480
	scale := float32(1.0)
	if float32(iw) > maxW {
		scale = maxW / float32(iw)
	}
	if float32(ih)*scale > maxH {
		scale = maxH / float32(ih)
	}
	piperGuideImg.Size = eui.Point{X: float32(iw) * scale, Y: float32(ih) * scale}
	piperGuideImg.Dirty = true

	piperGuidePrev.Disabled = page <= 0
	piperGuidePrev.Dirty = true
	piperGuideNext.Disabled = page >= len(piperGuideImages)-1
	piperGuideNext.Dirty = true

	piperGuideLabel.Text = strconv.Itoa(page+1) + " / " + strconv.Itoa(len(piperGuideImages))
	piperGuideLabel.Dirty = true

	piperGuideWin.Refresh()
}

func makePiperGuideWindow() {
	if piperGuideWin != nil {
		piperGuideWin.MarkOpen()
		return
	}
	if !loadPiperGuideImages() {
		return
	}

	piperGuideWin = eui.NewWindow()
	piperGuideWin.Title = "Piper Voice Guide"
	piperGuideWin.Closable = true
	piperGuideWin.Resizable = false
	piperGuideWin.AutoSize = true
	piperGuideWin.Movable = true
	piperGuideWin.SetZone(eui.HZoneCenter, eui.VZoneMiddleTop)

	flow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL}

	piperGuideImg, _ = eui.NewImageItem(1, 1)
	flow.AddItem(piperGuideImg)

	navFlow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true, Alignment: eui.ALIGN_CENTER}
	navFlow.Size = eui.Point{Y: 24}

	piperGuideLabel, _ = eui.NewText()
	piperGuideLabel.Text = ""
	piperGuideLabel.FontSize = 12
	piperGuideLabel.Size = eui.Point{X: 80, Y: 24}

	prevBtn, prevEvents := eui.NewButton()
	prevBtn.Text = "<"
	prevBtn.Size = eui.Point{X: 32, Y: 24}
	prevEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			showPiperGuidePage(piperGuidePage - 1)
		}
	}
	navFlow.AddItem(prevBtn)
	piperGuidePrev = prevBtn

	navFlow.AddItem(piperGuideLabel)

	nextBtn, nextEvents := eui.NewButton()
	nextBtn.Text = ">"
	nextBtn.Size = eui.Point{X: 32, Y: 24}
	nextEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			showPiperGuidePage(piperGuidePage + 1)
		}
	}
	navFlow.AddItem(nextBtn)
	piperGuideNext = nextBtn

	flow.AddItem(navFlow)

	piperGuideWin.AddItem(flow)
	piperGuideWin.AddWindow(false)

	showPiperGuidePage(0)
}
