package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gotk3/gotk3/gtk"
)

func promptEphemeralLaunch(chanb chan bool, sandbox string) {
	gtk.Init(nil)

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		fmt.Printf("Unable to create window: %v\n", err)
		os.Exit(1)
	}
	win.SetTitle("OZ Launch: " + sandbox)
	win.SetModal(true)
	win.SetKeepAbove(true)
	win.SetDecorated(true)
	win.SetUrgencyHint(true)
	win.SetDeletable(false)
	win.SetResizable(false)
	win.SetIconName(sandbox)

	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	headerbar, err := gtk.HeaderBarNew()
	if err != nil {
		fmt.Printf("Unable to create headerbar: %v\n", err)
		os.Exit(1)
	}
	headerbar.SetTitle("OZ: Launch")
	headerbar.SetSubtitle(sandbox)
	headerbar.SetShowCloseButton(false)

	win.SetTitlebar(headerbar)

	win.Add(promptEphemeralWindowWidget(chanb, sandbox, win))

	win.ShowAll()
	gtk.Main()

	chanb <- false
}

func promptEphemeralWindowWidget(chanb chan bool, sandbox string, win *gtk.Window) *gtk.Widget {
	grid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create grid: %v\n", err)
		os.Exit(1)
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)

	outerGrid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create grid: %v\n", err)
		os.Exit(1)
	}
	outerGrid.SetOrientation(gtk.ORIENTATION_HORIZONTAL)

	innerGrid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create grid: %v\n", err)
		os.Exit(1)
	}
	innerGrid.SetOrientation(gtk.ORIENTATION_VERTICAL)

	diagIcon, err := gtk.ImageNewFromIconName("dialog-question", gtk.ICON_SIZE_DIALOG)
	if err != nil {
		fmt.Printf("Unable to create label: %v\n", err)
		os.Exit(1)
	}

	topMsg := "Do you wish to start this sandbox in ephemeral mode?"
	topLabel, err := gtk.LabelNew(topMsg)
	if err != nil {
		fmt.Printf("Unable to create label: %v\n", err)
		os.Exit(1)
	}
	topLabel.SetMarkup("<span size=\"large\">" + topMsg + "</span>")

	nameLabel, err := gtk.LabelNew(sandbox)
	if err != nil {
		fmt.Printf("Unable to create label: %v\n", err)
		os.Exit(1)
	}
	nameLabel.SetMarkup("<span size=\"large\" weight=\"bold\">" + sandbox + "</span>")

	btnGrid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create btnGrid: %v\n", err)
		os.Exit(1)
	}
	btnGrid.SetOrientation(gtk.ORIENTATION_HORIZONTAL)
	btnGrid.SetColumnHomogeneous(true)

	btnCancel, err := gtk.ButtonNewWithLabel("No")
	if err != nil {
		fmt.Printf("Unable to create btnCancel: %v\n", err)
		os.Exit(1)
	}
	btnCancel.SetCanDefault(true)
	btnCancelStyle, _ := btnCancel.Container.Widget.GetStyleContext()
	btnCancelStyle.AddClass("suggested-action")

	btnYes, err := gtk.ButtonNewWithLabel("Yes")
	if err != nil {
		fmt.Printf("Unable to create btnYes: %v\n", err)
		os.Exit(1)
	}

	btnCancel.Connect("clicked", win.Destroy)
	btnYes.Connect("clicked", func() {
		chanb <- true
		win.Destroy()
	})

	btnGrid.Add(btnCancel)
	btnGrid.Add(btnYes)
	btnGrid.SetColumnSpacing(25)
	btnGrid.Container.Widget.SetMarginTop(25)

	innerGrid.SetRowSpacing(10)
	innerGrid.Add(topLabel)
	innerGrid.Add(nameLabel)

	//outerGrid.SetRowSpacing(25)
	outerGrid.Add(diagIcon)
	outerGrid.Add(innerGrid)

	grid.SetRowSpacing(10)
	grid.Container.Widget.SetMarginStart(15)
	grid.Container.Widget.SetMarginEnd(15)
	grid.Container.Widget.SetMarginTop(25)
	grid.Container.Widget.SetMarginBottom(15)
	grid.Add(outerGrid)
	grid.Add(btnGrid)

	topLabel.SetHExpand(true)
	nameLabel.SetHExpand(true)
	btnGrid.SetHExpand(true)

	return &grid.Container.Widget
}

func promptConfirmShell(chanb chan bool, sandbox string, id int) {
	gtk.Init(nil)

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		fmt.Printf("Unable to create window: %v\n", err)
		os.Exit(1)
	}
	win.SetTitle("OZ Launch Shell: " + sandbox)
	win.SetModal(true)
	win.SetKeepAbove(true)
	win.SetDecorated(true)
	win.SetUrgencyHint(true)
	win.SetDeletable(false)
	win.SetResizable(false)
	win.SetIconName("utilities-terminal")

	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	headerbar, err := gtk.HeaderBarNew()
	if err != nil {
		fmt.Printf("Unable to create headerbar: %v\n", err)
		os.Exit(1)
	}
	headerbar.SetTitle("OZ: Launch Shell")
	headerbar.SetSubtitle(sandbox)
	headerbar.SetShowCloseButton(false)

	win.SetTitlebar(headerbar)

	win.Add(promptConfirmShellWindowWidget(chanb, sandbox, id, win))

	win.ShowAll()
	gtk.Main()

	chanb <- false
}

func promptConfirmShellWindowWidget(chanb chan bool, sandbox string, id int, win *gtk.Window) *gtk.Widget {
	grid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create grid: %v\n", err)
		os.Exit(1)
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)

	outerGrid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create grid: %v\n", err)
		os.Exit(1)
	}
	outerGrid.SetOrientation(gtk.ORIENTATION_HORIZONTAL)

	innerGrid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create grid: %v\n", err)
		os.Exit(1)
	}
	innerGrid.SetOrientation(gtk.ORIENTATION_VERTICAL)

	warnIcon, err := gtk.ImageNewFromIconName("dialog-warning", gtk.ICON_SIZE_DIALOG)
	if err != nil {
		fmt.Printf("Unable to create label: %v\n", err)
		os.Exit(1)
	}

	topMsg := "Do you really want to open a shell?"
	topLabel, err := gtk.LabelNew(topMsg)
	if err != nil {
		fmt.Printf("Unable to create label: %v\n", err)
		os.Exit(1)
	}
	topLabel.SetMarkup("<span size=\"large\">" + topMsg + "</span>")

	sid := strconv.Itoa(id)
	nameLabel, err := gtk.LabelNew("#" + sid + ": " + sandbox)
	if err != nil {
		fmt.Printf("Unable to create label: %v\n", err)
		os.Exit(1)
	}
	nameLabel.SetMarkup("<span size=\"large\" weight=\"bold\">#" + sid + ": " + sandbox + "</span>")

	btnGrid, err := gtk.GridNew()
	if err != nil {
		fmt.Printf("Unable to create btnGrid: %v\n", err)
		os.Exit(1)
	}
	btnGrid.SetOrientation(gtk.ORIENTATION_HORIZONTAL)
	btnGrid.SetColumnHomogeneous(true)

	btnCancel, err := gtk.ButtonNewWithLabel("Cancel")
	if err != nil {
		fmt.Printf("Unable to create btnCancel: %v\n", err)
		os.Exit(1)
	}
	btnCancel.SetCanDefault(true)
	btnCancelStyle, _ := btnCancel.Container.Widget.GetStyleContext()
	btnCancelStyle.AddClass("suggested-action")

	btnYes, err := gtk.ButtonNewWithLabel("Yes")
	if err != nil {
		fmt.Printf("Unable to create btnYes: %v\n", err)
		os.Exit(1)
	}

	btnCancel.Connect("clicked", win.Destroy)
	btnYes.Connect("clicked", func() {
		chanb <- true
		win.Destroy()
	})

	btnGrid.Add(btnCancel)
	btnGrid.Add(btnYes)
	btnGrid.SetColumnSpacing(25)
	btnGrid.Container.Widget.SetMarginTop(25)

	innerGrid.SetRowSpacing(10)
	innerGrid.Add(topLabel)
	innerGrid.Add(nameLabel)

	//outerGrid.SetRowSpacing(25)
	outerGrid.Add(warnIcon)
	outerGrid.Add(innerGrid)

	grid.SetRowSpacing(10)
	grid.Container.Widget.SetMarginStart(15)
	grid.Container.Widget.SetMarginEnd(15)
	grid.Container.Widget.SetMarginTop(25)
	grid.Container.Widget.SetMarginBottom(15)
	grid.Add(outerGrid)
	grid.Add(btnGrid)

	topLabel.SetHExpand(true)
	nameLabel.SetHExpand(true)
	btnGrid.SetHExpand(true)

	return &grid.Container.Widget
}
