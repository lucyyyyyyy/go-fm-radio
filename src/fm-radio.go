package main

import (
	"encoding/json"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/driver/desktop"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

// Create a struct to manage all the stations and frequency
type frequency struct {
	CurrFreq string
	Stations struct {
		Stations []station `json:"stations"`
	}
}

// Struct to allow for json handling
type station struct {
	Name     string `json:"name"`
	Hz       string `json:"freq"`
	Location string `json:"loc"`
}

// Function to find if a slice contains a value
func find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

// Function to handle errors
func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Function to convert Hz to MHz
func hzToMhz(hz string) string {
	i, err := strconv.ParseFloat(hz, 32)
	handleErr(err)
	mhz := i / 1000000
	return strconv.FormatFloat(mhz, 'f', -1, 32)
}

// Function to convert MHz to Hz
func mhzToHz(mhz string) string {
	i, err := strconv.ParseFloat(mhz, 32)
	handleErr(err)
	hz := i * 1000000
	return strconv.FormatFloat(hz, 'f', -1, 32)
}

// Run the FM demodulator
func runScript(ch chan struct{}) {
	// Start a process:
	cmd := exec.Command("./rtl_fm_streamer")
	handleErr(cmd.Start())
	// Block until ready to kill
	<-ch
	// Kill it:
	handleErr(cmd.Process.Kill())
}

func RemoveLastChar(input string) string {
	// If input string length is 1 or less return empty string.
	if len(input) <= 1 {
		return ""
	}
	// Return string new string
	return string([]rune(input)[:len(input)-1])
}

// Run VLC
func (freq *frequency) startVLC(ch chan struct{}) {
	// Start a process:
	cmd := exec.Command("VLC", "http://live-radio01.mediahubaustralia.com/2TJW/mp3/", "-I dummy") //"http://localhost/"+freq.CurrFreq+"/0", "-I dummy")
	handleErr(cmd.Start())
	// Block until ready to kill
	<-ch
	// Kill it:
	handleErr(cmd.Process.Kill())
}

// Function to find names of known stations
func (freq *frequency) nameStation() (string, string) {
	// Iterate over the station slice
	for a := range freq.Stations.Stations {
		// If the current station is known, return the name and location else return unknown
		if freq.Stations.Stations[a].Hz == freq.CurrFreq {
			return freq.Stations.Stations[a].Name, freq.Stations.Stations[a].Location
		}
	}
	return "", "Unknown"
}

// Function to change frequency of current station
func (freq *frequency) changeFreq(newFreq string) {
	freq.CurrFreq = newFreq
	// Convert string to int for use with JSON RPC
	//i, err := strconv.Atoi(newFreq)
	//handleErr(err)
	//var result int
	// Connect to JSON RPC
	//conn, err := jsonrpc.Dial("tcp", "http://localhost:2354")
	//handleErr(err)
	// Call the change of frequency
	//err = conn.Call("SetFrequency", i, &result)
}

// Home Screen (secretly not having any controls on it so I don't have to deal with that breaking it)
func (freq *frequency) homeScreen(a fyne.App, currentFrequency fyne.CanvasObject, currentStation fyne.CanvasObject, currentLocation fyne.CanvasObject) fyne.CanvasObject {
	centeredTop := fyne.NewContainerWithLayout(layout.NewHBoxLayout(), layout.NewSpacer(), currentStation, layout.NewSpacer())
	centeredMidTop := fyne.NewContainerWithLayout(layout.NewHBoxLayout(), layout.NewSpacer(), currentLocation, layout.NewSpacer())
	centered := fyne.NewContainerWithLayout(layout.NewHBoxLayout(),
		layout.NewSpacer(), currentFrequency, layout.NewSpacer())
	spacer := fyne.NewContainerWithLayout(layout.NewHBoxLayout(), layout.NewSpacer())
	return fyne.NewContainerWithLayout(layout.NewVBoxLayout(), spacer, spacer, spacer, centeredTop, spacer, spacer, centeredMidTop, spacer, spacer, spacer, spacer, spacer, spacer, centered)
}

// Create station list screen
func (freq *frequency) stationsScreen(a fyne.App, mainTabs *widget.TabContainer, killVlc chan struct{}) fyne.CanvasObject {
	locationStation := make([]string, 0)
	// Iterate over the stations
	for b := range freq.Stations.Stations {
		_, found := find(locationStation, freq.Stations.Stations[b].Location)
		// Add any new location to the slice
		if !found {
			locationStation = append(locationStation, freq.Stations.Stations[b].Location)
		}
	}
	sort.Strings(locationStation)
	locationTabs := make([]*widget.TabItem, 0)
	// Iteration over locations
	for c := range locationStation {
		type nameHZ struct {
			Name string
			Hz   string
		}
		localStations := make([]nameHZ, 0)
		for d := range freq.Stations.Stations {
			if freq.Stations.Stations[d].Location == locationStation[c] {
				localStations = append(localStations, nameHZ{freq.Stations.Stations[d].Name, freq.Stations.Stations[d].Hz})
			}
		}
		contents := make([]fyne.CanvasObject, 0)
		for e := range localStations {
			contents = append(contents, widget.NewButton(localStations[e].Name, func() {
				freq.changeFreq(localStations[e].Hz)
				stationName, stationLocation := freq.nameStation()
				currentFrequency := canvas.NewText(hzToMhz(freq.CurrFreq)+" MHz", color.White)
				currentFrequency.TextSize = 30
				currentStation := canvas.NewText(stationName, color.White)
				currentStation.TextSize = 50
				currentLocation := canvas.NewText(stationLocation, color.White)
				currentLocation.TextSize = 40
				freq.homeScreen(a, currentFrequency, currentStation, currentLocation).Refresh()
				mainTabs.Remove(mainTabs.Items[0])
				mainTabs.Remove(mainTabs.Items[0])
				mainTabs.Remove(mainTabs.Items[0])
				mainTabs.Remove(mainTabs.Items[0])
				mainTabs.Append(widget.NewTabItemWithIcon("Home", theme.HomeIcon(), freq.homeScreen(a, currentFrequency, currentStation, currentLocation)))
				mainTabs.Append(widget.NewTabItemWithIcon("Stations", theme.MenuIcon(), freq.stationsScreen(a, mainTabs, killVlc)))
				mainTabs.Append(widget.NewTabItemWithIcon("Tuner", theme.RadioButtonCheckedIcon(), freq.tuner(a, mainTabs, killVlc)))
				mainTabs.Append(widget.NewTabItemWithIcon("Quit", theme.CancelIcon(), freq.quitFM(a, killVlc)))
				mainTabs.Refresh()
				mainTabs.SelectTabIndex(0)
			}))
		}
		// Create a new tab for each location
		lists := fyne.NewContainerWithLayout(layout.NewGridLayout(3), contents...)
		locationTabs = append(locationTabs, widget.NewTabItem(locationStation[c], fyne.NewContainerWithLayout(layout.NewVBoxLayout(), lists)))
	}
	list := widget.NewTabContainer(locationTabs...)
	list.SetTabLocation(widget.TabLocationLeading)
	return fyne.NewContainerWithLayout(layout.NewHBoxLayout(), list)
}

// Function to quit app
func (freq *frequency) quitFM(a fyne.App, killVlc chan struct{}) fyne.CanvasObject {
	quitWidget := widget.NewButton("Quit Radio", func() {
		killVlc <- struct{}{}
		// Save the current frequency
		handleErr(ioutil.WriteFile("frequency", []byte(freq.CurrFreq), 0777))
		a.Quit()
	})
	return quitWidget
}

// Function for tuner
func (freq *frequency) tuner(a fyne.App, mainTabs *widget.TabContainer, killVlc chan struct{}) fyne.CanvasObject {
	desired := widget.NewEntry()
	desired.SetText(hzToMhz(freq.CurrFreq))
	one := widget.NewButton("1", func() {
		desired.SetText(desired.Text + "1")
	})
	two := widget.NewButton("2", func() {
		desired.SetText(desired.Text + "2")
	})
	three := widget.NewButton("3", func() {
		desired.SetText(desired.Text + "3")
	})
	four := widget.NewButton("4", func() {
		desired.SetText(desired.Text + "4")
	})
	five := widget.NewButton("5", func() {
		desired.SetText(desired.Text + "5")
	})
	six := widget.NewButton("6", func() {
		desired.SetText(desired.Text + "6")
	})
	seven := widget.NewButton("7", func() {
		desired.SetText(desired.Text + "7")
	})
	eight := widget.NewButton("8", func() {
		desired.SetText(desired.Text + "8")
	})
	nine := widget.NewButton("9", func() {
		desired.SetText(desired.Text + "9")
	})
	zero := widget.NewButton("0", func() {
		desired.SetText(desired.Text + "0")
	})
	point := widget.NewButton(".", func() {
		desired.SetText(desired.Text + ".")
	})
	delete := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		desired.SetText(RemoveLastChar(desired.Text))
	})
	tuneButton := widget.NewButton("Tune to this frequency (in MHz)", func() {
		freq.changeFreq(mhzToHz(desired.Text))
		stationName, stationLocation := freq.nameStation()
		currentFrequency := canvas.NewText(hzToMhz(freq.CurrFreq)+" MHz", color.White)
		currentFrequency.TextSize = 30
		currentStation := canvas.NewText(stationName, color.White)
		currentStation.TextSize = 50
		currentLocation := canvas.NewText(stationLocation, color.White)
		currentLocation.TextSize = 40
		freq.homeScreen(a, currentFrequency, currentStation, currentLocation).Refresh()
		mainTabs.Remove(mainTabs.Items[0])
		mainTabs.Remove(mainTabs.Items[0])
		mainTabs.Remove(mainTabs.Items[0])
		mainTabs.Remove(mainTabs.Items[0])
		mainTabs.Append(widget.NewTabItemWithIcon("Home", theme.HomeIcon(), freq.homeScreen(a, currentFrequency, currentStation, currentLocation)))
		mainTabs.Append(widget.NewTabItemWithIcon("Stations", theme.MenuIcon(), freq.stationsScreen(a, mainTabs, killVlc)))
		mainTabs.Append(widget.NewTabItemWithIcon("Tuner", theme.RadioButtonCheckedIcon(), freq.tuner(a, mainTabs, killVlc)))
		mainTabs.Append(widget.NewTabItemWithIcon("Quit", theme.CancelIcon(), freq.quitFM(a, killVlc)))
		mainTabs.Refresh()
		mainTabs.SelectTabIndex(0)
	})
	bValues := fyne.NewContainerWithLayout(layout.NewGridLayoutWithColumns(3), one, two, three, four, five, six, seven, eight, nine, zero, point, delete)
	return fyne.NewContainerWithLayout(layout.NewVBoxLayout(), desired, bValues, tuneButton)
}

func main() {
	freq := frequency{}
	killVlc := make(chan struct{})
	//killFm := make(chan struct{})
	//go runScript(killFm)
	// Check if frequency file exists
	if _, err := os.Stat("frequency"); err == nil {
		content, err := ioutil.ReadFile("frequency")
		handleErr(err)
		freq.CurrFreq = string(content)
	} else {
		// Reset to a default frequency if file doesn't exist
		freq.CurrFreq = "104100000"
	}
	// Check if stations file exists (if it doesn't it doesn't matter)
	if _, err := os.Stat("stations.json"); err == nil {
		stationsToSlice, err := ioutil.ReadFile("stations.json")
		handleErr(err)
		// Read from json into struct
		json.Unmarshal(stationsToSlice, &freq.Stations)
	}
	go freq.startVLC(killVlc)
	a := app.New()
	drv := fyne.CurrentApp().Driver()
	das := drv.(desktop.Driver)
	w := das.CreateSplashWindow()
	// Create side tabs
	stationName, stationLocation := freq.nameStation()
	currentFrequency := canvas.NewText(hzToMhz(freq.CurrFreq)+" MHz", color.White)
	currentFrequency.TextSize = 30
	currentStation := canvas.NewText(stationName, color.White)
	currentStation.TextSize = 50
	currentLocation := canvas.NewText(stationLocation, color.White)
	currentLocation.TextSize = 40
	tabs := widget.NewTabContainer(widget.NewTabItemWithIcon("Home", theme.HomeIcon(), freq.homeScreen(a, currentFrequency, currentStation, currentLocation)))
	tabs.Append(widget.NewTabItemWithIcon("Stations", theme.MenuIcon(), freq.stationsScreen(a, tabs, killVlc)))
	tabs.Append(widget.NewTabItemWithIcon("Tuner", theme.RadioButtonCheckedIcon(), freq.tuner(a, tabs, killVlc)))
	tabs.Append(widget.NewTabItemWithIcon("Quit", theme.CancelIcon(), freq.quitFM(a, killVlc)))
	tabs.SetTabLocation(widget.TabLocationTrailing)
	w.SetContent(tabs)
	w.Resize(fyne.NewSize(800, 480))
	w.ShowAndRun()
}
