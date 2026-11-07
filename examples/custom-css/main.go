package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/carlosxfelipe/brisa"
)

//go:embed ui
var uiAssets embed.FS

type GreetRequest struct {
	Name string `json:"name" doc:"Developer name" example:"Ada Lovelace" validate:"required,min=2,max=50"`
}

type GreetResponse struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type RuntimeStats struct {
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	GoVersion  string `json:"goVersion"`
	Goroutines int    `json:"goroutines"`
	HeapMB     string `json:"heapMB"`
	Uptime     string `json:"uptime"`
}

func main() {
	headless := os.Getenv("BRISA_HEADLESS") == "1"
	startedAt := time.Now()

	app := brisa.New(brisa.Config{
		AppID:           "com.carlosxfelipe.brisa.customcss",
		Title:           "Brisa — Custom CSS Showcase",
		Version:         "1.0.0",
		Description:     "Standalone custom design system with DisableThemeCSS: true.",
		DisableThemeCSS: true,
		Assets:          uiAssets,
		AssetsSubDir:    "ui",
		Window: brisa.WindowConfig{
			Title:     "Brisa — Custom CSS Showcase",
			Width:     880,
			Height:    620,
			MinWidth:  640,
			MinHeight: 480,
			Headless:  headless,
		},
		Store: brisa.StoreOptions{
			MemoryOnly: true,
		},
	})

	app.Use(brisa.Recover(), brisa.Logger())

	brisa.Bind(app, "greet", func(ctx *brisa.Context, req GreetRequest) (GreetResponse, error) {
		return GreetResponse{
			Message:   fmt.Sprintf("Hello, %s! Styled 100%% with your own CSS.", req.Name),
			Timestamp: time.Now().Format("15:04:05"),
		}, nil
	}).Summary("Greet user with declarative validation")

	brisa.Bind(app, "stats", func(ctx *brisa.Context, _ brisa.Empty) (RuntimeStats, error) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		return RuntimeStats{
			OS:         runtime.GOOS,
			Arch:       runtime.GOARCH,
			GoVersion:  runtime.Version(),
			Goroutines: runtime.NumGoroutine(),
			HeapMB:     fmt.Sprintf("%.2f MB", float64(mem.HeapAlloc)/(1024*1024)),
			Uptime:     fmt.Sprintf("%ds", int(time.Since(startedAt).Seconds())),
		}, nil
	}).Summary("Return Go runtime statistics")

	app.OnReady(func(a *brisa.App) {
		log.Printf("🎨 Custom CSS Showcase running at %s (Inspector: %s/brisa/inspector)", a.BaseURL(), a.BaseURL())
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-a.Window().Done():
					return
				case <-ticker.C:
					var mem runtime.MemStats
					runtime.ReadMemStats(&mem)
					a.Emit("runtime:tick", RuntimeStats{
						OS:         runtime.GOOS,
						Arch:       runtime.GOARCH,
						GoVersion:  runtime.Version(),
						Goroutines: runtime.NumGoroutine(),
						HeapMB:     fmt.Sprintf("%.2f MB", float64(mem.HeapAlloc)/(1024*1024)),
						Uptime:     fmt.Sprintf("%ds", int(time.Since(startedAt).Seconds())),
					})
				}
			}
		}()
	})

	log.Fatal(app.Run())
}
