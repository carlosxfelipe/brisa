package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/carlosxfelipe/brisa"
)

//go:embed ui assets/icon.png
var uiAssets embed.FS

// DTOs for REPR Commands

type CreateTaskRequest struct {
	Title    string `json:"title" doc:"Task title" example:"Ship Brisa v1.0" validate:"required,min=3,max=80"`
	Priority string `json:"priority" doc:"Task urgency" example:"high" validate:"required,oneof=low,medium,high"`
}

type Task struct {
	ID        string `json:"id" example:"tsk_1"`
	Title     string `json:"title" example:"Ship Brisa v1.0"`
	Priority  string `json:"priority" example:"high"`
	CreatedAt string `json:"createdAt" example:"2026-10-30T15:00:00Z"`
}

type TaskListResponse struct {
	Items []Task `json:"items"`
}

type DeleteTaskRequest struct {
	ID string `json:"id" doc:"Task ID to remove" example:"tsk_1" validate:"required"`
}

// Service-based RPC Binding Example

type SystemService struct{}

type SystemInfoResponse struct {
	OS        string `json:"os" example:"linux"`
	Arch      string `json:"arch" example:"amd64"`
	GoVersion string `json:"goVersion" example:"go1.26.1"`
	CPUs      int    `json:"cpus" example:"8"`
}

func (s *SystemService) Info() (SystemInfoResponse, error) {
	return SystemInfoResponse{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		CPUs:      runtime.NumCPU(),
	}, nil
}

func main() {
	headless := os.Getenv("BRISA_HEADLESS") == "1"

	app := brisa.New(brisa.Config{
		AppID:        "com.carlosxfelipe.brisa.studio",
		Title:        "Brisa Desktop Studio",
		Version:      "1.0.0",
		Description:  "Zero-dependency desktop application powered by Brisa.",
		Icon:         "assets/icon.png",
		Theme:        brisa.ThemeUbuntu, // Default: brisa.ThemeAdwaita
		Assets:       uiAssets,
		AssetsSubDir: "ui",
		Window: brisa.WindowConfig{
			Title:    "Brisa Desktop Studio",
			Width:    1120,
			Height:   740,
			Headless: headless,
		},
		Store: brisa.StoreOptions{
			EncryptionKey: "brisa-studio-local-secret-key",
		},
	})

	app.Use(brisa.Recover(), brisa.Logger())

	// 1. Bind struct service methods under "system.*"
	brisa.BindService(app, "system", &SystemService{})

	// 2. Group REPR commands under "tasks.*"
	tasks := app.Group("tasks").Tags("Tasks")

	brisa.Bind(tasks, "list", func(ctx *brisa.Context, _ brisa.Empty) (TaskListResponse, error) {
		var items []Task
		_, _ = ctx.Store().Get("tasks", &items)
		return TaskListResponse{Items: items}, nil
	}).Summary("List all saved tasks")

	brisa.Bind(tasks, "create", func(ctx *brisa.Context, req CreateTaskRequest) (Task, error) {
		var items []Task
		_, _ = ctx.Store().Get("tasks", &items)

		created := Task{
			ID:        fmt.Sprintf("tsk_%d", time.Now().UnixMilli()),
			Title:     req.Title,
			Priority:  req.Priority,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		items = append([]Task{created}, items...)
		if err := ctx.Store().Set("tasks", items); err != nil {
			return Task{}, err
		}
		ctx.Emit("tasks:created", created)
		return created, nil
	}).Summary("Create a new task with declarative validation")

	brisa.Bind(tasks, "remove", func(ctx *brisa.Context, req DeleteTaskRequest) (brisa.Map, error) {
		var items []Task
		_, _ = ctx.Store().Get("tasks", &items)

		filtered := make([]Task, 0, len(items))
		for _, item := range items {
			if item.ID != req.ID {
				filtered = append(filtered, item)
			}
		}
		_ = ctx.Store().Set("tasks", filtered)
		return brisa.Map{"deleted": req.ID}, nil
	}).Summary("Remove a task by ID")

	// 3. Background telemetry ticker emitting SSE events to the desktop UI
	startedAt := time.Now()
	app.OnReady(func(a *brisa.App) {
		log.Printf("🌬️ Brisa Studio running at %s (Inspector: %s/brisa/inspector)", a.BaseURL(), a.BaseURL())
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
					a.Emit("system:telemetry", brisa.Map{
						"goroutines": runtime.NumGoroutine(),
						"heapMB":     fmt.Sprintf("%.2f", float64(mem.HeapAlloc)/(1024*1024)),
						"uptimeSec":  int(time.Since(startedAt).Seconds()),
					})
				}
			}
		}()
	})

	app.OnStartup(func(ctx context.Context) error {
		return nil
	})

	log.Fatal(app.Run())
}
