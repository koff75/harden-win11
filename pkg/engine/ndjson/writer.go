// Package ndjson écrit des events NDJSON (Newline Delimited JSON) sur un io.Writer.
// Utilisé pour la sortie streaming du moteur, consommée par la GUI ou par jq.
package ndjson

import (
	"encoding/json"
	"io"
	"sync"
)

// Writer émet des events JSON ligne par ligne. Thread-safe : Emit peut
// être appelé depuis plusieurs goroutines simultanément, chaque event
// produit une ligne complète sans interleaving avec les autres.
type Writer struct {
	w  io.Writer
	mu sync.Mutex
}

// NewWriter retourne un Writer qui écrit sur w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Emit sérialise event en JSON compact suivi d'un \n. Le marshalling et le
// write sont protégés par un mutex pour garantir qu'aucune autre goroutine
// ne peut interleaver son output au milieu de cette ligne. Retourne la
// première erreur rencontrée (Marshal ou Write).
func (w *Writer) Emit(event any) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.w.Write(b)
	return err
}
