package core

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// FileFilter defines a file extension filter for OpenFile and SaveFile dialogs.
type FileFilter struct {
	Name     string   `json:"name"`     // e.g. "Images"
	Patterns []string `json:"patterns"` // e.g. []string{"*.png", "*.jpg"}
}

// FileDialogOptions configures native file/directory selection dialogs.
type FileDialogOptions struct {
	Title       string       `json:"title,omitempty"`
	DefaultPath string       `json:"defaultPath,omitempty"`
	DefaultName string       `json:"defaultName,omitempty"`
	Filters     []FileFilter `json:"filters,omitempty"`
}

// MessageDialogOptions configures native alert/confirmation dialogs.
type MessageDialogOptions struct {
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
	Kind    string `json:"kind,omitempty"` // "info", "warning", "error", "question"
}

// NotificationOptions configures a native OS desktop notification.
type NotificationOptions struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Subtitle string `json:"subtitle,omitempty"`
	Icon     string `json:"icon,omitempty"`
	Urgency  string `json:"urgency,omitempty"` // "low", "normal", "critical"
}

// Dialog provides cross-platform native desktop dialogs (file pickers and message boxes)
// with zero external Go dependencies.
type Dialog struct {
	app *App
}

// Info displays an informational message dialog.
func (d *Dialog) Info(title, message string) error {
	return d.Message(MessageDialogOptions{Title: title, Message: message, Kind: "info"})
}

// Warn displays a warning message dialog.
func (d *Dialog) Warn(title, message string) error {
	return d.Message(MessageDialogOptions{Title: title, Message: message, Kind: "warning"})
}

// Error displays an error message dialog.
func (d *Dialog) Error(title, message string) error {
	return d.Message(MessageDialogOptions{Title: title, Message: message, Kind: "error"})
}

// Message displays a native message dialog with fallback to the WebUI bridge.
func (d *Dialog) Message(opts MessageDialogOptions) error {
	if opts.Title == "" && d.app != nil {
		opts.Title = d.app.config.Title
	}
	if opts.Kind == "" {
		opts.Kind = "info"
	}

	switch runtime.GOOS {
	case "linux":
		if path, err := exec.LookPath("zenity"); err == nil {
			flag := "--info"
			switch opts.Kind {
			case "warning":
				flag = "--warning"
			case "error":
				flag = "--error"
			}
			cmd := exec.Command(path, flag, "--title="+opts.Title, "--text="+opts.Message, "--no-markup")
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
		if path, err := exec.LookPath("kdialog"); err == nil {
			flag := "--msgbox"
			if opts.Kind == "error" {
				flag = "--error"
			} else if opts.Kind == "warning" {
				flag = "--sorry"
			}
			cmd := exec.Command(path, flag, opts.Message, "--title", opts.Title)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	case "darwin":
		if path, err := exec.LookPath("osascript"); err == nil {
			script := fmt.Sprintf(`display dialog %q with title %q buttons {"OK"} default button "OK"`, opts.Message, opts.Title)
			if err := exec.Command(path, "-e", script).Run(); err == nil {
				return nil
			}
		}
	case "windows":
		if path, err := exec.LookPath("powershell"); err == nil {
			ps := fmt.Sprintf(`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show(%q, %q) | Out-Null`, opts.Message, opts.Title)
			if err := exec.Command(path, "-NoProfile", "-NonInteractive", "-Command", ps).Run(); err == nil {
				return nil
			}
		}
	}

	if d.app != nil {
		d.app.Emit("brisa:dialog:message", opts)
		return nil
	}
	return errors.New("brisa dialog: no native dialog utility available")
}

// Confirm displays a Yes/No confirmation dialog and returns true if confirmed.
func (d *Dialog) Confirm(title, message string) (bool, error) {
	if title == "" && d.app != nil {
		title = d.app.config.Title
	}

	switch runtime.GOOS {
	case "linux":
		if path, err := exec.LookPath("zenity"); err == nil {
			err := exec.Command(path, "--question", "--title="+title, "--text="+message, "--no-markup").Run()
			return err == nil, nil
		}
		if path, err := exec.LookPath("kdialog"); err == nil {
			err := exec.Command(path, "--yesno", message, "--title", title).Run()
			return err == nil, nil
		}
	case "darwin":
		if path, err := exec.LookPath("osascript"); err == nil {
			script := fmt.Sprintf(`display dialog %q with title %q buttons {"Cancel", "OK"} default button "OK"`, message, title)
			err := exec.Command(path, "-e", script).Run()
			return err == nil, nil
		}
	case "windows":
		if path, err := exec.LookPath("powershell"); err == nil {
			ps := fmt.Sprintf(`Add-Type -AssemblyName PresentationFramework; $res = [System.Windows.MessageBox]::Show(%q, %q, 'YesNo'); if ($res -eq 'Yes') { exit 0 } else { exit 1 }`, message, title)
			err := exec.Command(path, "-NoProfile", "-NonInteractive", "-Command", ps).Run()
			return err == nil, nil
		}
	}

	if d.app != nil && d.app.window != nil && d.app.window.HasCDP() {
		expr := fmt.Sprintf("window.confirm(%q)", message)
		res, err := d.app.window.Eval(expr)
		if err == nil {
			b, _ := res.(bool)
			return b, nil
		}
	}

	return false, errors.New("brisa dialog: no native confirm dialog utility available")
}

// OpenFile opens a file selection dialog and returns the selected file path (or empty string if canceled).
func (d *Dialog) OpenFile(opts FileDialogOptions) (string, error) {
	files, err := d.openFileInternal(opts, false, false)
	if err != nil || len(files) == 0 {
		return "", err
	}
	return files[0], nil
}

// OpenMultipleFiles opens a multi-file selection dialog and returns all selected paths.
func (d *Dialog) OpenMultipleFiles(opts FileDialogOptions) ([]string, error) {
	return d.openFileInternal(opts, true, false)
}

// OpenDirectory opens a folder selection dialog and returns the selected directory path.
func (d *Dialog) OpenDirectory(opts FileDialogOptions) (string, error) {
	files, err := d.openFileInternal(opts, false, true)
	if err != nil || len(files) == 0 {
		return "", err
	}
	return files[0], nil
}

// SaveFile opens a save file dialog and returns the chosen destination path.
func (d *Dialog) SaveFile(opts FileDialogOptions) (string, error) {
	if opts.Title == "" {
		opts.Title = "Save File"
	}

	switch runtime.GOOS {
	case "linux":
		if path, err := exec.LookPath("zenity"); err == nil {
			args := []string{"--file-selection", "--save", "--confirm-overwrite", "--title=" + opts.Title}
			if opts.DefaultName != "" {
				args = append(args, "--filename="+opts.DefaultName)
			} else if opts.DefaultPath != "" {
				args = append(args, "--filename="+opts.DefaultPath)
			}
			out, err := exec.Command(path, args...).Output()
			if err != nil {
				return "", nil // User canceled
			}
			return strings.TrimSpace(string(out)), nil
		}
	case "darwin":
		if path, err := exec.LookPath("osascript"); err == nil {
			script := fmt.Sprintf(`POSIX path of (choose file name with prompt %q)`, opts.Title)
			out, err := exec.Command(path, "-e", script).Output()
			if err != nil {
				return "", nil
			}
			return strings.TrimSpace(string(out)), nil
		}
	case "windows":
		if path, err := exec.LookPath("powershell"); err == nil {
			ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.SaveFileDialog; $f.Title = %q; if ($f.ShowDialog() -eq 'OK') { Write-Output $f.FileName }`, opts.Title)
			out, err := exec.Command(path, "-NoProfile", "-NonInteractive", "-Command", ps).Output()
			if err != nil {
				return "", nil
			}
			return strings.TrimSpace(string(out)), nil
		}
	}

	return "", errors.New("brisa dialog: no native save file dialog utility available")
}

func (d *Dialog) openFileInternal(opts FileDialogOptions, multiple, directory bool) ([]string, error) {
	if opts.Title == "" {
		if directory {
			opts.Title = "Select Folder"
		} else {
			opts.Title = "Select File"
		}
	}

	switch runtime.GOOS {
	case "linux":
		if path, err := exec.LookPath("zenity"); err == nil {
			args := []string{"--file-selection", "--title=" + opts.Title, "--separator=\n"}
			if directory {
				args = append(args, "--directory")
			}
			if multiple {
				args = append(args, "--multiple")
			}
			if opts.DefaultPath != "" {
				args = append(args, "--filename="+opts.DefaultPath)
			}
			for _, f := range opts.Filters {
				if len(f.Patterns) > 0 {
					args = append(args, fmt.Sprintf("--file-filter=%s | %s", f.Name, strings.Join(f.Patterns, " ")))
				}
			}
			out, err := exec.Command(path, args...).Output()
			if err != nil {
				return nil, nil // User canceled
			}
			return splitNonEmptyLines(string(out)), nil
		}
	case "darwin":
		if path, err := exec.LookPath("osascript"); err == nil {
			var script string
			if directory {
				script = fmt.Sprintf(`POSIX path of (choose folder with prompt %q)`, opts.Title)
			} else {
				script = fmt.Sprintf(`POSIX path of (choose file with prompt %q)`, opts.Title)
			}
			out, err := exec.Command(path, "-e", script).Output()
			if err != nil {
				return nil, nil
			}
			return splitNonEmptyLines(string(out)), nil
		}
	case "windows":
		if path, err := exec.LookPath("powershell"); err == nil {
			var ps string
			if directory {
				ps = fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.FolderBrowserDialog; $f.Description = %q; if ($f.ShowDialog() -eq 'OK') { Write-Output $f.SelectedPath }`, opts.Title)
			} else {
				multiStr := "$false"
				if multiple {
					multiStr = "$true"
				}
				ps = fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.OpenFileDialog; $f.Title = %q; $f.Multiselect = %s; if ($f.ShowDialog() -eq 'OK') { $f.FileNames }`, opts.Title, multiStr)
			}
			out, err := exec.Command(path, "-NoProfile", "-NonInteractive", "-Command", ps).Output()
			if err != nil {
				return nil, nil
			}
			return splitNonEmptyLines(string(out)), nil
		}
	}

	return nil, errors.New("brisa dialog: no native file dialog utility available")
}

func splitNonEmptyLines(s string) []string {
	var res []string
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			res = append(res, line)
		}
	}
	return res
}

// Clipboard provides cross-platform clipboard read/write operations with an in-memory fallback.
type Clipboard struct {
	mu       sync.RWMutex
	fallback string
	app      *App
}

// WriteText writes plain text to the system clipboard (and updates the internal fallback buffer).
func (c *Clipboard) WriteText(text string) error {
	c.mu.Lock()
	c.fallback = text
	c.mu.Unlock()

	switch runtime.GOOS {
	case "linux":
		for _, candidate := range [][]string{
			{"wl-copy"},
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
		} {
			if path, err := exec.LookPath(candidate[0]); err == nil {
				cmd := exec.Command(path, candidate[1:]...)
				cmd.Stdin = strings.NewReader(text)
				if err := cmd.Run(); err == nil {
					return nil
				}
			}
		}
	case "darwin":
		if path, err := exec.LookPath("pbcopy"); err == nil {
			cmd := exec.Command(path)
			cmd.Stdin = strings.NewReader(text)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	case "windows":
		if path, err := exec.LookPath("clip"); err == nil {
			cmd := exec.Command(path)
			cmd.Stdin = strings.NewReader(text)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	if c.app != nil {
		c.app.Emit("brisa:clipboard:write", Map{"text": text})
	}
	return nil
}

// ReadText reads plain text from the system clipboard (falling back to the internal buffer if headless).
func (c *Clipboard) ReadText() (string, error) {
	switch runtime.GOOS {
	case "linux":
		for _, candidate := range [][]string{
			{"wl-paste", "--no-newline"},
			{"xclip", "-selection", "clipboard", "-o"},
			{"xsel", "--clipboard", "--output"},
		} {
			if path, err := exec.LookPath(candidate[0]); err == nil {
				var out bytes.Buffer
				cmd := exec.Command(path, candidate[1:]...)
				cmd.Stdout = &out
				if err := cmd.Run(); err == nil {
					return out.String(), nil
				}
			}
		}
	case "darwin":
		if path, err := exec.LookPath("pbpaste"); err == nil {
			out, err := exec.Command(path).Output()
			if err == nil {
				return string(out), nil
			}
		}
	case "windows":
		if path, err := exec.LookPath("powershell"); err == nil {
			out, err := exec.Command(path, "-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard").Output()
			if err == nil {
				return strings.TrimSuffix(strings.TrimSuffix(string(out), "\n"), "\r"), nil
			}
		}
	}

	c.mu.RLock()
	val := c.fallback
	c.mu.RUnlock()
	return val, nil
}

// Notify sends a desktop notification using native OS tools and emits a WebUI notification event.
func Notify(app *App, opts NotificationOptions) error {
	if opts.Title == "" && app != nil {
		opts.Title = app.config.Title
	}
	if app != nil {
		app.Emit("brisa:notification", opts)
	}

	switch runtime.GOOS {
	case "linux":
		if path, err := exec.LookPath("notify-send"); err == nil {
			args := []string{}
			if opts.Urgency != "" {
				args = append(args, "-u", opts.Urgency)
			}
			if opts.Icon != "" {
				args = append(args, "-i", opts.Icon)
			}
			args = append(args, opts.Title, opts.Body)
			_ = exec.Command(path, args...).Start()
			return nil
		}
	case "darwin":
		if path, err := exec.LookPath("osascript"); err == nil {
			script := fmt.Sprintf(`display notification %q with title %q`, opts.Body, opts.Title)
			_ = exec.Command(path, "-e", script).Start()
			return nil
		}
	case "windows":
		if path, err := exec.LookPath("powershell"); err == nil {
			ps := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null; $xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); $text = $xml.GetElementsByTagName('text'); $text.Item(0).AppendChild($xml.CreateTextNode(%q)) | Out-Null; $text.Item(1).AppendChild($xml.CreateTextNode(%q)) | Out-Null; $toast = [Windows.UI.Notifications.ToastNotification]::new($xml); [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Brisa').Show($toast)`, opts.Title, opts.Body)
			_ = exec.Command(path, "-NoProfile", "-NonInteractive", "-Command", ps).Start()
			return nil
		}
	}
	return nil
}

// OpenExternal opens a URL or file path in the operating system's default application.
func OpenExternal(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("brisa: empty target for OpenExternal")
	}
	switch runtime.GOOS {
	case "linux":
		if path, err := exec.LookPath("xdg-open"); err == nil {
			return exec.Command(path, target).Start()
		}
	case "darwin":
		if path, err := exec.LookPath("open"); err == nil {
			return exec.Command(path, target).Start()
		}
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	}
	return fmt.Errorf("brisa: no system opener available for %q", target)
}
