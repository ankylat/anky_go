package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	depletionTime = 8 * time.Second
)

var (
	lastKeyStrokeTime time.Time
	textArea          *widget.Entry
	lifeBar           *canvas.Rectangle
	percentageLabel   *widget.Label
	timerStarted      bool
	mu                sync.Mutex
	lifeColors        = []color.NRGBA{
		{148, 0, 211, 255}, // violet
		{75, 0, 130, 255},  // indigo
		{0, 0, 255, 255},   // blue
		{0, 255, 0, 255},   // green
		{255, 255, 0, 255}, // yellow
		{255, 165, 0, 255}, // orange
		{255, 0, 0, 255},   // red
	}
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("anky")
	myWindow.Resize(fyne.NewSize(960, 600))

	dir := "writings"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			fmt.Println("Failed to create directory:", err)
			return
		}
	}

	bgImage := canvas.NewImageFromFile("librarian.jpeg")
	bgImage.FillMode = canvas.ImageFillStretch

	textArea = widget.NewMultiLineEntry()
	textArea.SetPlaceHolder("Write something...")
	textArea.Wrapping = fyne.TextWrapWord
	textArea.OnChanged = func(content string) {
		mu.Lock()
		lastKeyStrokeTime = time.Now()
		if !timerStarted {
			timerStarted = true
			go monitorKeystrokes(myWindow, dir)
		}
		lifeBar.FillColor = lifeColors[0]
		lifeBar.SetMinSize(fyne.NewSize(960, 20))
		percentageLabel.SetText("100%")
		mu.Unlock()
	}

	overlay := container.NewStack(
		canvas.NewRectangle(color.NRGBA{0, 0, 0, 153}),
		textArea,
	)

	percentageLabel = widget.NewLabel("100%")

	navBar := container.NewHBox(
		canvas.NewRectangle(color.NRGBA{0, 128, 0, 255}),
		widget.NewLabelWithStyle("ANKY", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		percentageLabel,
	)

	lifeBar = canvas.NewRectangle(lifeColors[0])
	lifeBar.SetMinSize(fyne.NewSize(960, 20))

	var setInitialLayout func()
	setInitialLayout = func() {

		viewWritingsButton := widget.NewButton("View Writings", func() {
			showWritingsList(myWindow, dir, bgImage, navBar, setInitialLayout)
		})

		content := container.NewVBox(
			container.NewStack(
				lifeBar,
				navBar,
			),
			layout.NewSpacer(),
			overlay,
			viewWritingsButton,
		)

		if len(textArea.Text) > 0 {
			content.Add(viewWritingsButton)
		}

		myWindow.SetContent(container.NewStack(
			bgImage,
			content,
		))
	}

	// Initially set up the layout
	setInitialLayout()

	myWindow.ShowAndRun()
}

func saveWriting(dir string) string {
	text := textArea.Text
	filename := getNextFilename(dir)
	filepath := filepath.Join(dir, filename)
	err := os.WriteFile(filepath, []byte(text), 0644)
	if err != nil {
		fmt.Println("Failed to save the file:", err)
	}
	return filepath
}

func getNextFilename(dir string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("Failed to read directory:", err)
		return "1.txt"
	}

	// Get the highest number from the filenames
	var numbers []int
	for _, file := range files {
		name := file.Name()
		numberStr := name[:len(name)-len(filepath.Ext(name))]
		number, err := strconv.Atoi(numberStr)
		if err == nil {
			numbers = append(numbers, number)
		}
	}
	sort.Ints(numbers)

	// Return the next sequential number
	if len(numbers) > 0 {
		return fmt.Sprintf("%d.txt", numbers[len(numbers)-1]+1)
	}
	return "1.txt"
}

func showWritingContent(myWindow fyne.Window, filepath string, dir string, bgImage *canvas.Image, navBar *fyne.Container, setInitialLayout func()) {
	contentBytes, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Println("Failed to read file:", err)
		return
	}
	content := string(contentBytes)

	backButton := widget.NewButton("Back", func() {
		resetSession()
		setInitialLayout()
	})

	writeAgainButton := widget.NewButton("Write Again", func() {
		resetSession()
		setInitialLayout()
	})

	writingLabel := widget.NewLabel(content)
	writingLabel.Wrapping = fyne.TextWrapWord

	buttons := container.NewHBox(
		layout.NewSpacer(),
		container.NewGridWithColumns(2,
			container.NewCenter(backButton),
			container.NewCenter(writeAgainButton),
		),
		layout.NewSpacer(),
	)

	contentContainer := container.NewVBox(
		container.NewStack(
			lifeBar,
			navBar,
		),
		container.New(
			layout.NewStackLayout(),
			canvas.NewRectangle(color.Black),
			container.NewVBox(writingLabel),
		),
		buttons,
	)

	myWindow.SetContent(container.NewStack(
		bgImage,
		contentContainer,
	))
}

func showWritingsList(myWindow fyne.Window, dir string, bgImage *canvas.Image, navBar *fyne.Container, setInitialLayout func()) {
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("Failed to read directory:", err)
		return
	}

	var writings []fyne.CanvasObject
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			continue
		}
		writing := string(content)
		title := trimText(writing, 30)
		writingButton := widget.NewButton(title, func() {
			showWritingContent(myWindow, filepath.Join(dir, file.Name()), dir, bgImage, navBar, setInitialLayout)
		})
		writings = append(writings, writingButton)
	}

	backButton := widget.NewButton("Back", func() {
		resetSession()
		setInitialLayout()
	})

	myWindow.SetContent(container.NewStack(
		bgImage,
		container.NewVBox(
			container.NewStack(
				lifeBar,
				navBar,
			),
			widget.NewLabel("Writings:"),
			container.NewVBox(writings...),
			backButton,
		),
	))
}

func trimText(text string, length int) string {
	if len(text) > length {
		return text[:length] + "..."
	}
	return text
}

func resetSession() {
	mu.Lock()
	defer mu.Unlock()
	timerStarted = false
	lastKeyStrokeTime = time.Time{}
}

func monitorKeystrokes(myWindow fyne.Window, dir string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		elapsed := time.Since(lastKeyStrokeTime).Seconds()
		if time.Duration(elapsed)*time.Second > depletionTime {
			mu.Unlock()
			filepath := saveWriting(dir)
			showWritingContent(myWindow, filepath, dir, canvas.NewImageFromFile("librarian.jpeg"), container.NewVBox(), func() {})
			break
		}
		percentage := 100 - int(elapsed*100/depletionTime.Seconds())
		lifeBar.SetMinSize(fyne.NewSize(float32(960*percentage/100), 20))
		colorIndex := int((elapsed / depletionTime.Seconds()) * float64(len(lifeColors)))
		if colorIndex >= len(lifeColors) {
			colorIndex = len(lifeColors) - 1
		}
		lifeBar.FillColor = lifeColors[colorIndex]
		percentageLabel.SetText(fmt.Sprintf("%d%%", percentage))
		fmt.Println("Current percentage:", percentage)
		lifeBar.Refresh()
		mu.Unlock()
		myWindow.Content().Refresh()
	}
}
