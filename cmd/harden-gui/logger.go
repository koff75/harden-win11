package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// guiLogger écrit les events de la GUI dans un fichier que je peux lire
// pour debugger à distance. Path : %LOCALAPPDATA%\Harden-Win11\gui.log.
type guiLogger struct {
	mu     sync.Mutex
	file   *os.File
	logger *log.Logger
	path   string
}

var glog *guiLogger

func initLogger() {
	dir := logDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		// Fallback : pas de log fichier, on log juste sur stderr (visible
		// quand on lance harden-gui.exe depuis un terminal).
		log.Printf("logger: cannot create %s: %v", dir, err)
		return
	}
	path := filepath.Join(dir, "gui.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("logger: cannot open %s: %v", path, err)
		return
	}
	glog = &guiLogger{
		file:   f,
		logger: log.New(f, "", log.LstdFlags|log.Lmicroseconds),
		path:   path,
	}
	glog.logger.Println("=== gui startup ===")
	glog.logger.Printf("log file: %s", path)
}

func logDir() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, "Harden-Win11")
	}
	return filepath.Join(os.TempDir(), "Harden-Win11")
}

// LogPath retourne le path du fichier journal (utilisé par GetEngineInfo).
func LogPath() string {
	if glog == nil {
		return ""
	}
	return glog.path
}

// logf écrit un message formaté dans le fichier de log si dispo, sinon stderr.
func logf(format string, args ...any) {
	if glog == nil {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
		return
	}
	glog.mu.Lock()
	defer glog.mu.Unlock()
	glog.logger.Printf(format, args...)
}
